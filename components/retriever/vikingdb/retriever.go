package vikingdb

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"code.byted.org/flow/eino/callbacks"
	"code.byted.org/flow/eino/components"
	"code.byted.org/flow/eino/components/embedding"
	"code.byted.org/flow/eino/components/retriever"
	"code.byted.org/flow/eino/schema"
	viking "code.byted.org/lagrange/viking_go_client"
)

const (
	defaultTopK     = 10
	defaultSubIndex = "default"
)

type RetrieverConfig struct {
	Name   string        `json:"name"`
	Token  string        `json:"token"`
	Region viking.Region `json:"region"`

	EmbeddingConfig EmbeddingConfig `json:"embedding_config"`

	Index string `json:"index"`
	// SubIndex will be set with "default" if len is zero
	SubIndex       string   `json:"sub_index"`
	TopK           int      `json:"top_k"`
	ScoreThreshold *float64 `json:"score_threshold"`

	// option
	FilterDSL map[string]any `json:"filter_dsl"`
}

type EmbeddingConfig struct {
	// UseBuiltin Use built-in vectorization method, only available in Region_CN
	// Check the currently supported vectorization methods and conduct tests in the VikingDB vector library on byterec platform.
	// See: https://bytedance.larkoffice.com/wiki/UhCPwrAogi4p2Ukhb9dc74AInSh
	UseBuiltin bool `json:"use_builtin"`

	// UseSparse and SparseLogitAlpha for HNSW hybrid vector retrieval
	// See: https://bytedance.larkoffice.com/wiki/F3iJwhUa9i3WYFkdOOUcEddFnAh
	UseSparse        bool    `json:"use_sparse"`
	SparseLogitAlpha float64 `json:"sparse_logit_alpha"`

	// Embedding when UseBuiltin is false
	// If Embedding from here or from retriever.Option is provided, it will take precedence over built-in vectorization methods
	Embedding embedding.Embedder
}

type Retriever struct {
	client *viking.VikingDbClient
	config *RetrieverConfig
}

func NewRetriever(ctx context.Context, conf *RetrieverConfig) (*Retriever, error) {
	if conf.EmbeddingConfig.UseBuiltin {
		if conf.Region != viking.Region_CN {
			return nil, fmt.Errorf("[VikingDBRetriever] built-in vectorization method not support in non-CN regions")
		} else if conf.EmbeddingConfig.Embedding != nil {
			return nil, fmt.Errorf("[VikingDBRetriever] no need to provide Embedding when UseBuiltin vectorization method")
		}
	} else if conf.EmbeddingConfig.Embedding == nil {
		return nil, fmt.Errorf("[NewRetriever] embedding not provided")
	}

	return &Retriever{
		client: viking.NewVikingDbClient(conf.Name, conf.Token, conf.Region),
		config: conf,
	}, nil
}

func (r *Retriever) Retrieve(ctx context.Context, query string, opts ...retriever.Option) (docs []*schema.Document, err error) {
	cbm, cbmOK := callbacks.ManagerFromCtx(ctx)

	defer func() {
		if err != nil && cbmOK {
			cbm.OnError(ctx, err)
		}
	}()

	if cbmOK {
		ctx = cbm.OnStart(ctx, &retriever.CallbackInput{Query: query})
	}

	options := retriever.GetCommonOptions(&retriever.Options{
		Index:          r.config.Index,
		SubIndex:       r.config.SubIndex,
		TopK:           r.config.TopK,
		ScoreThreshold: r.config.ScoreThreshold,
		DSLInfo:        r.config.FilterDSL,
	}, opts...)

	useBuiltinEmbedding := r.config.EmbeddingConfig.UseBuiltin && options.Embedding == nil

	var (
		dense  []float64
		sparse [][]interface{}
	)

	if useBuiltinEmbedding {
		dense, sparse, err = r.embeddingRaw(ctx, query, options)
	} else {
		dense, err = r.embeddingQuery(ctx, query, options)
	}

	if err != nil {
		return nil, err
	}

	docs, err = r.retrieverWithVector(ctx, dense, sparse, options)
	if err != nil {
		return nil, err
	}

	if cbmOK {
		cbm.OnEnd(ctx, &retriever.CallbackOutput{Docs: docs})
	}

	return docs, nil
}

func (r *Retriever) embeddingRaw(ctx context.Context, query string, options *retriever.Options) (dense []float64, sparse [][]interface{}, err error) {
	data := []*viking.RawData{
		{RawData: map[string]interface{}{"text": query}},
	}

	if !r.config.EmbeddingConfig.UseSparse {
		denseEmbeddings, _, err := r.client.RawEmbedding(data)
		if err != nil {
			return nil, nil, err
		}

		if len(denseEmbeddings) != 1 {
			return nil, nil, fmt.Errorf("[embeddingRaw] invalid return length of dense embeddings, got=%d, expected=1", len(denseEmbeddings))
		}

		return f32To64(denseEmbeddings[0]), nil, nil
	}

	denseEmbeddings, sparseEmbeddings, _, err := r.client.RawEmbeddingWithSparse(data)
	if err != nil {
		return nil, nil, err
	}

	if len(denseEmbeddings) != 1 {
		return nil, nil, fmt.Errorf("[embeddingRaw] invalid return length of dense embeddings, got=%d, expected=1", len(denseEmbeddings))
	}

	if len(sparseEmbeddings) != 1 {
		return nil, nil, fmt.Errorf("[embeddingRaw] invalid return length of sparse embeddings, got=%d, expected=1", len(sparseEmbeddings))
	}

	rawSparse := sparseEmbeddings[0]
	sparse = make([][]interface{}, len(rawSparse))
	for i := range rawSparse {
		element, ok := rawSparse[i].(string)
		if !ok {
			return nil, nil, fmt.Errorf("[embeddingRaw] invalid sparse embedding element type, expected=string, got=%T", rawSparse[i])
		}
		element = element[1 : len(element)-1]
		parts := strings.Split(element, ",")
		strValue := strings.TrimSpace(parts[0])
		strValue = strValue[1 : len(strValue)-1]
		floatValue := strings.TrimSpace(parts[1])
		float, err := strconv.ParseFloat(floatValue, 64)
		if err != nil { // unexpected
			return nil, nil, fmt.Errorf("[embeddingRaw] parse sparse embedding element failed, element=%v, err=%w", element, err)
		}

		sparse[i] = []interface{}{strValue, float}
	}

	return f32To64(denseEmbeddings[0]), sparse, nil
}

func (r *Retriever) embeddingQuery(ctx context.Context, query string, options *retriever.Options) (vector []float64, err error) {
	emb := r.config.EmbeddingConfig.Embedding
	if options.Embedding != nil {
		emb = options.Embedding
	}

	vectors, err := emb.EmbedStrings(r.makeEmbeddingCtx(ctx, emb), []string{query})
	if err != nil {
		return nil, err
	}

	if len(vectors) != 1 { // unexpected
		return nil, fmt.Errorf("[embeddingQuery] invalid return length of vector, got=%d, expected=1", len(vectors))
	}

	return vectors[0], nil
}

func (r *Retriever) retrieverWithVector(ctx context.Context, dense []float64, sparse [][]interface{}, options *retriever.Options) (docs []*schema.Document, err error) {
	req := &viking.RecallRequest{
		Index:            options.Index,
		SubIndex:         options.SubIndex,
		Embedding:        f64To32(dense),
		SparseEmbedding:  sparse,
		SparseLogitAlpha: r.config.EmbeddingConfig.SparseLogitAlpha,
		TopK:             int32(options.TopK),
		DslInfo:          options.DSLInfo,
	}

	if len(req.SubIndex) == 0 {
		req.SubIndex = defaultSubIndex
	}

	if req.TopK == 0 {
		req.TopK = defaultTopK
	}

	resp, _, err := r.client.Recall(req)
	if err != nil {
		return nil, err
	} else if resp == nil { // nothing recalled
		return []*schema.Document{}, nil
	}

	docs = make([]*schema.Document, 0, len(resp.Result))
	for _, result := range resp.Result {
		if options.ScoreThreshold != nil && result.Scores < *options.ScoreThreshold {
			continue
		}

		doc := &schema.Document{
			ID:      strconv.FormatUint(result.LabelLower64, 10),
			Content: result.Attrs,
		}

		docs = append(docs,
			doc.WithScore(result.Scores).
				WithVikingExtraInfo(result.ExtraInfos).
				WithVikingDSLInfo(req.DslInfo))
	}

	return docs, nil
}

func (r *Retriever) makeEmbeddingCtx(ctx context.Context, emb embedding.Embedder) context.Context {
	cbm, ok := callbacks.ManagerFromCtx(ctx)
	if !ok {
		return ctx
	}

	runInfo := &callbacks.RunInfo{
		Component: components.ComponentOfEmbedding,
	}

	if embType, ok := components.GetType(emb); ok {
		runInfo.Type = embType
	}

	runInfo.Name = runInfo.Type + string(runInfo.Component)

	return callbacks.CtxWithManager(ctx, cbm.WithRunInfo(runInfo))
}

func (r *Retriever) GetType() string {
	return typ
}

func (r *Retriever) IsCallbacksEnabled() bool {
	return true
}
