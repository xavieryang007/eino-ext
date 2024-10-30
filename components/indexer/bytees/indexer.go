package bytees

import (
	"context"
	"fmt"
	"time"

	"code.byted.org/flow/eino/callbacks"
	"code.byted.org/flow/eino/components"
	"code.byted.org/flow/eino/components/embedding"
	"code.byted.org/flow/eino/components/indexer"
	"code.byted.org/flow/eino/schema"
	"code.byted.org/lang/gg/gslice"
	tes "code.byted.org/toutiao/elastic/v7"
)

type IndexerConfig struct {
	PSM     string        `json:"psm"`
	Cluster string        `json:"cluster"`
	Domain  string        `json:"domain"`
	Scheme  Scheme        `json:"scheme"`
	Timeout time.Duration `json:"timeout"`

	ClientOptions    []tes.ClientOptionFunc
	TransportOptions []tes.TransportOpt

	Index     string `json:"index"`
	BatchSize int    `json:"batch_size"`

	// UseKNN hnsw search
	// see: https://cloud.bytedance.net/docs/ses/docs/6614e634c6a2250303cdeb04/6630ae80a4561c02e780c33f
	UseKNN bool `json:"use_knn"`
	// Fields below need to be provided when UseKNN is true
	// VectorFields set vector fields for knn
	VectorFields []VectorFieldKV `json:"vector_fields"`
	// Embedding vectorization method for doc content when UseKNN is true
	// and no need to provide when UseKNN is false
	Embedding embedding.Embedder
}

type Indexer struct {
	config *IndexerConfig
	client *tes.Client
}

func NewIndexer(ctx context.Context, conf *IndexerConfig) (*Indexer, error) {
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

	if len(conf.VectorFields) == 0 {
		conf.VectorFields = []VectorFieldKV{
			DefaultVectorFieldKV(DocFieldNameContent),
		}
	}

	if conf.BatchSize == 0 {
		conf.BatchSize = defaultBatchSize
	}

	i := &Indexer{
		config: conf,
		client: esClient,
	}

	return i, nil
}

func (i *Indexer) Store(ctx context.Context, docs []*schema.Document, opts ...indexer.Option) (ids []string, err error) {
	defer func() {
		if err != nil {
			ctx = callbacks.OnError(ctx, err)
		}
	}()

	ctx = callbacks.OnStart(ctx, &indexer.CallbackInput{Docs: docs})

	bulk := i.client.Bulk()

	options := indexer.GetCommonOptions(&indexer.Options{
		Embedding: i.config.Embedding,
	}, opts...)

	for _, slice := range gslice.Chunk(docs, i.config.BatchSize) {
		var brs []tes.BulkableRequest

		if i.config.UseKNN {
			brs, err = i.knnQueryRequest(ctx, slice, options)
		} else {
			brs, err = i.defaultQueryRequest(ctx, slice, options)
		}

		if err != nil {
			return nil, fmt.Errorf("[bytees indexer] make bulk request failed, %w", err)
		}

		bulk.Add(brs...)
	}

	resp, err := bulk.Do(ctx)
	if err != nil {
		return nil, err
	}

	if resp.Errors {
		errs := iter(resp.Failed(), func(i int, t *tes.BulkResponseItem) string {
			return fmt.Sprintf("docs[%d] bulk index error, err=%v\n", i, t.Error.Reason)
		})

		return nil, fmt.Errorf("[bytees indexer] bulk failed, errs:\n%v", errs)
	}

	ids = iter(docs, func(idx int, t *schema.Document) string {
		return t.ID
	})

	ctx = callbacks.OnEnd(ctx, &indexer.CallbackOutput{IDs: ids})

	return ids, nil
}

func (i *Indexer) defaultQueryRequest(ctx context.Context, docs []*schema.Document, options *indexer.Options) (brs []tes.BulkableRequest, err error) {
	brs = iter(docs, func(idx int, doc *schema.Document) tes.BulkableRequest {
		return tes.NewBulkIndexRequest().
			Index(i.config.Index).
			OpType(opTypeIndex).
			Id(doc.ID).
			Doc(toESDoc(doc))
	})

	return brs, nil
}

func (i *Indexer) knnQueryRequest(ctx context.Context, docs []*schema.Document, options *indexer.Options) (brs []tes.BulkableRequest, err error) {
	emb := options.Embedding

	brs, err = iterWithErr(docs, func(idx int, doc *schema.Document) (tes.BulkableRequest, error) {
		texts := make([]string, 0, len(i.config.VectorFields))
		for _, kv := range i.config.VectorFields {
			str, ok := kv.fieldName.Find(doc)
			if !ok {
				return nil, fmt.Errorf("[knnQueryRequest] field name not found or type incorrect, name=%s, doc=%v", kv.fieldName, doc)
			}

			texts = append(texts, str)
		}

		vectors, embErr := emb.EmbedStrings(i.makeEmbeddingCtx(ctx, emb), texts)
		if embErr != nil {
			return nil, embErr
		}

		if len(vectors) != len(texts) {
			return nil, fmt.Errorf("[knnQueryRequest] invalid vector length, got=%d, expected=%d", len(vectors), len(texts))
		}

		mp := toESDoc(doc)
		for vIdx, kv := range i.config.VectorFields {
			mp[string(kv.key)] = vectors[vIdx]
		}

		return tes.NewBulkIndexRequest().
			Index(i.config.Index).
			OpType(opTypeIndex).
			Id(doc.ID).
			Doc(mp), nil
	})

	if err != nil {
		return nil, err
	}

	return brs, nil
}

func (i *Indexer) makeEmbeddingCtx(ctx context.Context, emb embedding.Embedder) context.Context {
	runInfo := &callbacks.RunInfo{
		Component: components.ComponentOfEmbedding,
	}

	if embType, ok := components.GetType(emb); ok {
		runInfo.Type = embType
	}

	runInfo.Name = runInfo.Type + string(runInfo.Component)

	return callbacks.SwitchRunInfo(ctx, runInfo)
}

func (i *Indexer) GetType() string {
	return typ
}

func (i *Indexer) IsCallbacksEnabled() bool {
	return true
}
