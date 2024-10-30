package bytees

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"code.byted.org/flow/eino/callbacks"
	"code.byted.org/flow/eino/components"
	"code.byted.org/flow/eino/components/embedding"
	"code.byted.org/flow/eino/components/retriever"
	"code.byted.org/flow/eino/schema"
	"code.byted.org/flow/flow-telemetry-common/go/obtag"
	"code.byted.org/lang/gg/gmap"
	tes "code.byted.org/toutiao/elastic/v7"
)

type RetrieverConfig struct {
	PSM     string        `json:"psm"`
	Cluster string        `json:"cluster"`
	Domain  string        `json:"domain"`
	Scheme  Scheme        `json:"scheme"`
	Timeout time.Duration `json:"timeout"`

	ClientOptions    []tes.ClientOptionFunc
	TransportOptions []tes.TransportOpt

	Index          string   `json:"index"`
	TopK           int      `json:"topK"`
	ScoreThreshold *float64 `json:"score_threshold"`

	// SearchMode controls how to search
	// see: https://cloud.bytedance.net/docs/ses/docs/6614e634c6a2250303cdeb04/6630ae80a4561c02e780c33f
	SearchMode SearchMode `json:"search_mode"`
	// Fields below need to be provided when SearchMode is SearchModeKNN or SearchModeKNNWithFilters
	// Embedding vectorization method for VectorFieldKV.Value
	Embedding embedding.Embedder
}

type Retriever struct {
	config *RetrieverConfig
	client *tes.Client
}

func NewRetriever(ctx context.Context, conf *RetrieverConfig) (*Retriever, error) {
	var esOptions []any
	if len(conf.ClientOptions) > 0 {
		esOptions = append(esOptions, iter(conf.ClientOptions, func(_ int, t tes.ClientOptionFunc) any { return t })...)
	}
	if len(conf.TransportOptions) > 0 {
		esOptions = append(esOptions, iter(conf.TransportOptions, func(_ int, t tes.TransportOpt) any { return t })...)
	}

	var (
		esClient *tes.Client
		err      error
	)

	if len(conf.PSM) != 0 && conf.Scheme == HTTPS {
		esClient, err = tes.NewPSMHTTPSClient(conf.PSM, conf.Cluster, conf.Timeout, esOptions...)
	} else if len(conf.PSM) != 0 {
		esClient, err = tes.NewPSMClient(conf.PSM, conf.Cluster, conf.Timeout, esOptions...)
	} else if len(conf.Domain) != 0 {
		esClient, err = tes.NewDomainClient(conf.Domain, conf.Timeout, esOptions...)
	} else {
		err = fmt.Errorf("NewElasticsearch fail since without PSM & Domain")
	}

	if err != nil {
		return nil, err
	}

	if conf.TopK == 0 {
		conf.TopK = defaultTopK
	}

	r := &Retriever{
		config: conf,
		client: esClient,
	}

	return r, nil
}

func (r *Retriever) Retrieve(ctx context.Context, query string, opts ...retriever.Option) (docs []*schema.Document, err error) {
	defer func() {
		if err != nil {
			ctx = callbacks.OnError(ctx, err)
		}
	}()

	options := retriever.GetCommonOptions(&retriever.Options{
		Index:          &r.config.Index,
		TopK:           &r.config.TopK,
		ScoreThreshold: r.config.ScoreThreshold,
		Embedding:      r.config.Embedding,
		DSLInfo:        map[string]interface{}{},
	}, opts...)

	ctx = callbacks.OnStart(ctx, &retriever.CallbackInput{
		Query:          query,
		TopK:           *options.TopK,
		ScoreThreshold: options.ScoreThreshold,
		// TODO: show dsl filter
		Extra: map[string]any{
			obtag.ESName:    r.config.PSM,
			obtag.ESCluster: r.config.Cluster,
			obtag.ESIndex:   *options.Index,
		},
	})

	var q tes.Query

	switch r.config.SearchMode {
	case SearchModeContentMatch:
		q = tes.NewMatchQuery(DocFieldNameContent, query)
	case SearchModeKNN:
		simpleQuery, err := r.makeKNNQuery(ctx, query, options)
		if err != nil {
			return nil, err
		}

		if filters, ok := getDSLFilterField(options.DSLInfo); ok {
			fsq := tes.NewFunctionScoreQuery().
				Query(simpleQuery).
				Boost(1).
				ScoreMode("max").
				BoostMode("multiply")

			for i := range filters {
				fsq.Add(filters[i].Query, filters[i].ScoreFunction)
			}

			if options.ScoreThreshold != nil {
				fsq.MinScore(*options.ScoreThreshold)
			}

			q = fsq
		} else {
			q = simpleQuery
		}
	case SearchModeRawStringQuery:
		q = tes.NewRawStringQuery(query)
	default: // unexpected
		return nil, fmt.Errorf("[bytees retriever] search mode invalid, mode=%d", r.config.SearchMode)
	}

	ss := r.client.Search().Index(*options.Index).Query(q).Size(*options.TopK)
	if options.ScoreThreshold != nil {
		ss = ss.MinScore(*options.ScoreThreshold)
	}

	result, err := ss.Do(ctx)
	if err != nil {
		return nil, err
	}

	if result.Error != nil {
		return nil, fmt.Errorf("[bytees retriever] resp error not nil, reason=%v", result.Error.Reason)
	}

	if result.Hits == nil { // unexpected
		return nil, fmt.Errorf("[bytees retriever] resp hits is nil")
	}

	docs, err = r.parseSearchResult(result)
	if err != nil {
		return nil, err
	}

	ctx = callbacks.OnEnd(ctx, &retriever.CallbackOutput{Docs: docs})

	return docs, nil
}

func (r *Retriever) makeKNNQuery(ctx context.Context, rawQuery string, options *retriever.Options) (tes.Query, error) {
	emb := options.Embedding

	var rq []VectorFieldKV
	if err := json.Unmarshal([]byte(rawQuery), &rq); err != nil {
		return nil, err
	}

	if len(rq) == 0 {
		return nil, fmt.Errorf("[makeKNNQuery] invalid query, raw query=%s", rawQuery)
	}

	texts := iter(rq, func(i int, kv VectorFieldKV) string {
		return kv.Value
	})

	vectors, err := emb.EmbedStrings(r.makeEmbeddingCtx(ctx, emb), texts)
	if err != nil {
		return nil, err
	}

	if len(vectors) != len(texts) {
		return nil, fmt.Errorf("[makeKNNQuery] invalid return length of vector, got=%d, expected=%d", len(vectors), len(texts))
	}

	var rawQueries []tes.Query

	for i, pair := range rq {
		kb, err := json.Marshal(&knnQuery{
			map[string]vectorQuery{string(pair.Key): {vectors[i], *options.TopK}},
		})
		if err != nil {
			return nil, err
		}

		rawQueries = append(rawQueries, tes.NewRawStringQuery(string(kb)))
	}

	if len(rawQueries) == 1 {
		return rawQueries[0], nil
	}

	return tes.NewBoolQuery().Should(rawQueries...), nil
}

func (r *Retriever) parseSearchResult(result *tes.SearchResult) (docs []*schema.Document, err error) {
	docs = make([]*schema.Document, 0, len(result.Hits.Hits))
	for _, hit := range result.Hits.Hits {
		var raw map[string]any
		if err = json.Unmarshal(hit.Source, &raw); err != nil {
			return nil, fmt.Errorf("[parseSearchResult] unexpected hit source type, source=%v", string(hit.Source))
		}

		content, ok := raw[DocFieldNameContent].(string)
		if !ok {
			return nil, fmt.Errorf("[parseSearchResult] content type not string, raw=%v", raw)
		}

		doc := &schema.Document{
			ID:      hit.Id,
			Content: content,
			MetaData: map[string]any{
				docExtraKeyEsFields: gmap.FilterKeys(raw, func(s string) bool {
					return s != DocFieldNameContent
				}),
			},
		}

		if hit.Score != nil {
			doc.WithScore(*hit.Score)
		}

		docs = append(docs, doc)
	}

	return docs, nil
}

func (r *Retriever) makeEmbeddingCtx(ctx context.Context, emb embedding.Embedder) context.Context {
	runInfo := &callbacks.RunInfo{
		Component: components.ComponentOfEmbedding,
	}

	if embType, ok := components.GetType(emb); ok {
		runInfo.Type = embType
	}

	runInfo.Name = runInfo.Type + string(runInfo.Component)

	return callbacks.SwitchRunInfo(ctx, runInfo)
}

func (r *Retriever) GetType() string {
	return typ
}

func (r *Retriever) IsCallbacksEnabled() bool {
	return true
}
