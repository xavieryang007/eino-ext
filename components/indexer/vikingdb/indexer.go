package vikingdb

import (
	"context"
	"fmt"

	"code.byted.org/flow/eino/callbacks"
	"code.byted.org/flow/eino/components"
	"code.byted.org/flow/eino/components/embedding"
	"code.byted.org/flow/eino/components/indexer"
	"code.byted.org/flow/eino/schema"
	"code.byted.org/gopkg/logs/v2"
	viking "code.byted.org/lagrange/viking_go_client"
	"code.byted.org/lang/gg/gslice"
)

const (
	defaultAddBatchSize = 5
	defaultSubIndex     = "default"
)

type IndexerConfig struct {
	Name   string        `json:"name"`
	Token  string        `json:"token"`
	Region viking.Region `json:"region"`

	EmbeddingConfig EmbeddingConfig `json:"embedding_config"`

	// SubIndexes will be set with []string{"default"} if len is zero
	SubIndexes []string `json:"sub_indexes"`

	AddBatchSize int `json:"add_batch_size"`
}

type EmbeddingConfig struct {
	// UseBuiltin Use built-in vectorization method, only available in Region_CN
	// Check the currently supported vectorization methods and conduct tests in the VikingDB vector library on byterec platform.
	// See: https://bytedance.larkoffice.com/wiki/UhCPwrAogi4p2Ukhb9dc74AInSh
	UseBuiltin bool `json:"use_builtin"`

	// Embedding when UseBuiltin is false
	// If Embedding from here or from indexer.Option is provided, it will take precedence over built-in vectorization methods
	Embedding embedding.Embedder
}

type Indexer struct {
	client *viking.VikingDbClient
	config *IndexerConfig
}

func NewIndexer(_ context.Context, conf *IndexerConfig) (*Indexer, error) {
	if conf.EmbeddingConfig.UseBuiltin {
		if conf.Region != viking.Region_CN {
			return nil, fmt.Errorf("[VikingDBIndexer] built-in vectorization method not support in non-CN regions")
		} else if conf.EmbeddingConfig.Embedding != nil {
			return nil, fmt.Errorf("[VikingDBIndexer] no need to provide Embedding when UseBuiltin vectorization method")
		}
	} else if conf.EmbeddingConfig.Embedding == nil {
		return nil, fmt.Errorf("[NewIndexer] embedding not provided")
	}

	if conf.AddBatchSize == 0 {
		conf.AddBatchSize = defaultAddBatchSize
	}

	return &Indexer{
		client: viking.NewVikingDbClient(conf.Name, conf.Token, conf.Region),
		config: conf,
	}, nil
}

func (i *Indexer) Store(ctx context.Context, docs []*schema.Document, opts ...indexer.Option) (ids []string, err error) {
	cbm, cbmOK := callbacks.ManagerFromCtx(ctx)

	defer func() {
		if err != nil {
			logs.CtxError(ctx, "[Store] failed, set ids=%v, err=%v", ids, err)

			if cbmOK {
				cbm.OnError(ctx, err)
			}
		}
	}()

	if cbmOK {
		ctx = cbm.OnStart(ctx, &indexer.CallbackInput{Docs: docs})
	}

	options := indexer.GetCommonOptions(&indexer.Options{}, opts...)

	useBuiltinEmbedding := i.config.EmbeddingConfig.UseBuiltin && options.Embedding == nil

	for _, chunks := range gslice.Chunk(docs, i.config.AddBatchSize) {
		var chunkIds []string

		if useBuiltinEmbedding {
			chunkIds, err = i.storeRaw(ctx, chunks, options)
			if err != nil {
				return nil, err
			}
		} else {
			var docsWithEmbedding []*schema.Document

			// embedding
			docsWithEmbedding, err = i.embeddingDocs(ctx, chunks, options)
			if err != nil {
				return nil, err
			}

			// store
			chunkIds, err = i.storeWithVector(ctx, docsWithEmbedding, options)
			if err != nil {
				return nil, err
			}
		}

		ids = append(ids, chunkIds...)
	}

	if cbmOK {
		cbm.OnEnd(ctx, &indexer.CallbackOutput{IDs: ids})
	}

	return ids, nil
}

func (i *Indexer) storeRaw(_ context.Context, docs []*schema.Document, options *indexer.Options) (ids []string, err error) {
	subIndexes := i.config.SubIndexes
	if len(options.SubIndexes) > 0 {
		subIndexes = options.SubIndexes
	}

	rawDataList := make([]*viking.RawData, len(docs))
	dataList := make([]*viking.VikingDbData, len(docs))
	for j := range docs {
		doc := docs[j]
		subIndexes = gslice.Uniq(append(subIndexes, doc.SubIndexes()...))
		if len(subIndexes) == 0 {
			subIndexes = []string{defaultSubIndex}
		}

		rawDataList[j] = &viking.RawData{
			RawData: map[string]interface{}{"text": doc.Content},
		}

		id, err := parseUint64(doc.ID)
		if err != nil {
			return nil, err
		}

		dataList[j] = &viking.VikingDbData{
			LabelLower: id,
			Context:    subIndexes,
			Attrs:      doc.Content,
			DslInfo:    doc.VikingDSLInfo(),
		}
	}

	ids, _, err = i.client.AddRawData(rawDataList, dataList)
	if err != nil {
		return nil, err
	}

	return ids, nil
}

func (i *Indexer) embeddingDocs(ctx context.Context, docs []*schema.Document, options *indexer.Options) ([]*schema.Document, error) {
	emb := i.config.EmbeddingConfig.Embedding
	if options.Embedding != nil {
		emb = options.Embedding
	}

	contents := make([]string, len(docs))
	for j := range docs {
		contents[j] = docs[j].Content
	}

	vectors, err := emb.EmbedStrings(i.makeEmbeddingCtx(ctx, emb), contents)
	if err != nil {
		return nil, err
	}

	if len(vectors) != len(docs) { // unexpected
		return nil, fmt.Errorf("[embeddingDocs] invalid return length of vector, got=%d, expected=%d", len(vectors), len(docs))
	}

	for j := range docs {
		docs[j].WithVector(vectors[j])
	}

	return docs, nil
}

func (i *Indexer) storeWithVector(_ context.Context, dvs []*schema.Document, options *indexer.Options) (ids []string, err error) {
	subIndexes := i.config.SubIndexes
	if len(options.SubIndexes) > 0 {
		subIndexes = options.SubIndexes
	}

	dataList := make([]*viking.VikingDbData, len(dvs))
	for j := range dvs {
		dv := dvs[j]
		subIndexes = gslice.Uniq(append(subIndexes, dv.SubIndexes()...))
		if len(subIndexes) == 0 {
			subIndexes = []string{defaultSubIndex}
		}

		id, err := parseUint64(dv.ID)
		if err != nil {
			return nil, err
		}
		dataList[j] = viking.NewVikingDbDataF32(
			id,
			subIndexes,
			f64To32(dv.Vector()),
			viking.WithAttrs(dv.Content),
			viking.WithDslInfo(dv.VikingDSLInfo()))
	}

	ids, _, err = i.client.AddData(dataList)
	if err != nil {
		return nil, err
	}

	return ids, nil
}

func (i *Indexer) makeEmbeddingCtx(ctx context.Context, emb embedding.Embedder) context.Context {
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

func (i *Indexer) GetType() string {
	return typ
}

func (i *Indexer) IsCallbacksEnabled() bool {
	return true
}
