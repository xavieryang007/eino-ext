// Package fornaxknowledge implement eino components/retriever.
// For more information, please refers to https://bytedance.larkoffice.com/wiki/DgBmwgzLViZPxrkvz4wc41GpnKj.
package fornaxknowledge

import (
	"context"
	"fmt"

	"github.com/bytedance/sonic"

	"code.byted.org/flow/eino/callbacks"
	"code.byted.org/flow/eino/components/retriever"
	"code.byted.org/flow/eino/schema"
	"code.byted.org/flowdevops/fornax_sdk"
	"code.byted.org/flowdevops/fornax_sdk/domain"
	fknowledge "code.byted.org/flowdevops/fornax_sdk/domain/knowledge"
	"code.byted.org/lang/gg/gptr"
)

const typ = "FornaxKnowledge"

var (
	// RankerRRF (Rank-Reciprocal Rank Fusion) is a method in information retrieval that combines multiple ranked lists from different search engines to produce a more accurate and comprehensive ranking.
	// It assigns a score to each document based on its reciprocal rank in different lists and sums these scores to determine the final ranking, thus improving search effectiveness and robustness.
	RankerRRF = "rrf"
	// RankerIntersection is a method in information retrieval that combines multiple ranked lists by identifying and using only the common documents that appear in all lists.
	// The final ranking prioritizes documents that are consistently retrieved by different systems, assuming commonality indicates higher relevance.
	RankerIntersection = "intersection"
)

var fornaxRanker = map[string]*fknowledge.Ranker{
	"rrf": {
		Type: gptr.Of("rrf"),
	},
	"intersect": {
		Type: gptr.Of("intersection"),
	},
}

// Converter converts fornax knowledge item to schema.Document.
type Converter func(ctx context.Context, docs []*fknowledge.Item) ([]*schema.Document, error)

// Config is the configuration for knowledge retriever.
type Config struct {
	// identity for fornax space.
	AK string `json:"ak"`
	SK string `json:"sk"`

	// knowledge keys, e.g. flow.eino.docs
	KnowledgeKeys []string `json:"knowledge_keys"`

	Channels []*fknowledge.Channel `json:"channels"`

	// RankerRRF or RankerIntersection.
	Rank   string             `json:"rank"`
	ranker *fknowledge.Ranker `json:"-"`

	Filter *string `json:"filter"`

	TopK      *int32    `json:"top_k,omitempty"` // default 3
	Converter Converter `json:"-"`               // default defaultConverter
}

// NewKnowledgeRetriever creates a knowledge retriever.
func NewKnowledgeRetriever(ctx context.Context, config *Config) (*Knowledge, error) {
	const defaultTopK = int32(3)

	if config.TopK == nil {
		config.TopK = ptrOf(defaultTopK)
	}
	if config.Converter == nil {
		config.Converter = defaultConverter
	}

	ranker := fornaxRanker[config.Rank]
	config.ranker = ranker

	client, err := fornax_sdk.NewClient(&domain.Config{
		Identity: &domain.Identity{
			AK: config.AK,
			SK: config.SK,
		},
	})
	if err != nil {
		return nil, err
	}

	r := &Knowledge{
		client: client,
		config: config,
	}

	return r, nil
}

// Knowledge implement eino retriever for fornax knowledge.
type Knowledge struct {
	config *Config

	client fornax_sdk.IClient
}

func (k *Knowledge) Retrieve(ctx context.Context, input string, opts ...retriever.Option) (docs []*schema.Document, err error) {
	var (
		cbm, cbmOK = callbacks.ManagerFromCtx(ctx)
	)

	extra := map[string]any{
		"knowledge_keys": k.config.KnowledgeKeys,
		"channels":       k.config.Channels,
		"rank":           k.config.ranker,
		"filter":         k.config.Filter,
		"query":          input,
	}

	defer func() {
		if err != nil && cbmOK {
			_ = cbm.OnError(ctx, err)
		}
	}()

	baseTopK := int(dereferenceOrZero(k.config.TopK))

	opt := retriever.GetCommonOptions(&retriever.Options{
		TopK: &baseTopK,
	}, opts...)

	extra["top_k"] = dereferenceOrZero(opt.TopK)

	req := &fknowledge.RetrieveKnowledgeParams{
		Query:         input,
		TopK:          int32(dereferenceOrZero(opt.TopK)),
		KnowledgeKeys: k.config.KnowledgeKeys,
		Channels:      k.config.Channels,
		Rank:          k.config.ranker,
		Filter:        k.config.Filter,
	}

	if cbmOK {
		ctx = cbm.OnStart(ctx, &retriever.CallbackInput{
			Query:          input,
			TopK:           int(req.TopK),
			Filter:         dereferenceOrZero(req.Filter),
			ScoreThreshold: nil, // not support multiple channels threshold
			Extra:          extra,
		})
	}

	result, err := k.client.RetrieveKnowledge(ctx, req)
	if err != nil {
		return nil, err
	}

	if result.Data == nil || len(result.Data.Items) == 0 {
		return make([]*schema.Document, 0), nil
	}

	retrieveDocs, err := k.config.Converter(ctx, result.Data.Items)
	if err != nil {
		return nil, err
	}

	if cbmOK {
		_ = cbm.OnEnd(ctx, &retriever.CallbackOutput{Docs: retrieveDocs, Extra: extra})
	}

	return retrieveDocs, nil
}

func (k *Knowledge) GetType() string {
	return typ
}

func (k *Knowledge) IsCallbacksEnabled() bool {
	return true
}

func defaultConverter(ctx context.Context, docs []*fknowledge.Item) ([]*schema.Document, error) {
	results := make([]*schema.Document, 0, len(docs))
	for idx, doc := range docs {
		meta := map[string]interface{}{}
		if len(doc.SliceMeta) > 0 {
			err := sonic.UnmarshalString(doc.SliceMeta, &meta)
			if err != nil {
				return nil, fmt.Errorf("unmarshal slice meta fail: %s", err)
			}
		}
		meta["_score"] = doc.Score
		meta["_index"] = idx

		results = append(results, &schema.Document{
			ID:       doc.DocID,
			Content:  doc.Slice,
			MetaData: meta,
		})
	}

	return results, nil
}
