package bytees

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	. "github.com/bytedance/mockey"
	"github.com/smartystreets/goconvey/convey"

	"code.byted.org/flow/eino/components/embedding"
	"code.byted.org/flow/eino/components/retriever"
	"code.byted.org/flow/eino/schema"
	"code.byted.org/lang/gg/gptr"
	tes "code.byted.org/toutiao/elastic/v7"
)

func TestMakeKNNQuery(t *testing.T) {
	PatchConvey("test makeKNNQuery", t, func() {
		ctx := context.Background()
		emb := &mockEmbedding{}
		opt := &retriever.Options{
			TopK:      gptr.Of(10),
			Embedding: emb,
		}
		r := &Retriever{}

		PatchConvey("test json unmarshal error", func() {
			q, err := r.makeKNNQuery(ctx, "asd", opt)
			convey.So(err, convey.ShouldNotBeNil)
			convey.So(q, convey.ShouldBeNil)
		})

		PatchConvey("test invalid vector field kv", func() {
			rawQuery, err := NewKNNQuery()
			convey.So(err, convey.ShouldBeNil)

			q, err := r.makeKNNQuery(ctx, rawQuery, opt)
			convey.So(err, convey.ShouldNotBeNil)
			convey.So(err.Error(), convey.ShouldContainSubstring, "invalid query")
			convey.So(q, convey.ShouldBeNil)
		})

		PatchConvey("test embedding error", func() {
			Mock(GetMethod(emb, "EmbedStrings")).Return(nil, fmt.Errorf("mock err")).Build()
			rawQuery, err := NewKNNQuery([]VectorFieldKV{
				{GetDefaultVectorFieldKeyContent(), "mock_content"},
				{GetDefaultVectorFieldKey("ext_field_2"), "mock_extra_2"},
			}...)
			convey.So(err, convey.ShouldBeNil)

			q, err := r.makeKNNQuery(ctx, rawQuery, opt)
			convey.So(err, convey.ShouldNotBeNil)
			convey.So(err.Error(), convey.ShouldContainSubstring, "mock err")
			convey.So(q, convey.ShouldBeNil)
		})

		PatchConvey("test embedding resp invalid length", func() {
			Mock(GetMethod(emb, "EmbedStrings")).Return([][]float64{{1.1}}, nil).Build()
			rawQuery, err := NewKNNQuery([]VectorFieldKV{
				{GetDefaultVectorFieldKeyContent(), "mock_content"},
				{GetDefaultVectorFieldKey("ext_field_2"), "mock_extra_2"},
			}...)
			convey.So(err, convey.ShouldBeNil)

			q, err := r.makeKNNQuery(ctx, rawQuery, opt)
			convey.So(err, convey.ShouldNotBeNil)
			convey.So(err.Error(), convey.ShouldContainSubstring, "invalid return length of vector")
			convey.So(q, convey.ShouldBeNil)
		})

		PatchConvey("test single query", func() {
			Mock(GetMethod(emb, "EmbedStrings")).Return([][]float64{{1.1}}, nil).Build()
			rawQuery, err := NewKNNQuery([]VectorFieldKV{
				{GetDefaultVectorFieldKeyContent(), "mock_content"},
			}...)
			convey.So(err, convey.ShouldBeNil)

			q, err := r.makeKNNQuery(ctx, rawQuery, opt)
			convey.So(err, convey.ShouldBeNil)
			convey.So(q, convey.ShouldNotBeNil)

			src, err := q.Source()
			convey.So(err, convey.ShouldBeNil)
			b, err := json.Marshal(src)
			convey.So(err, convey.ShouldBeNil)
			convey.So(string(b), convey.ShouldEqual, "{\"knn\":{\"vector_eino_doc_content\":{\"k\":10,\"vector\":[1.1]}}}")
		})

		PatchConvey("test multiple queries", func() {
			rawQuery, err := NewKNNQuery([]VectorFieldKV{
				{GetDefaultVectorFieldKeyContent(), "mock_content"},
				{GetDefaultVectorFieldKey("ext_field_2"), "mock_extra_2"},
			}...)
			convey.So(err, convey.ShouldBeNil)

			q, err := r.makeKNNQuery(ctx, rawQuery, opt)
			convey.So(err, convey.ShouldBeNil)
			convey.So(q, convey.ShouldNotBeNil)

			src, err := q.Source()
			convey.So(err, convey.ShouldBeNil)
			b, err := json.Marshal(src)
			convey.So(err, convey.ShouldBeNil)
			convey.So(string(b), convey.ShouldEqual, "{\"bool\":{\"should\":[{\"knn\":{\"vector_eino_doc_content\":{\"k\":10,\"vector\":[1.1]}}},{\"knn\":{\"vector_ext_field_2\":{\"k\":10,\"vector\":[2.1]}}}]}}")
		})
	})
}

func TestParseSearchResult(t *testing.T) {
	PatchConvey("test parseSearchResult", t, func() {
		r := &Retriever{}

		PatchConvey("test hit.Source invalid", func() {
			docs, err := r.parseSearchResult(&tes.SearchResult{Hits: &tes.SearchHits{Hits: []*tes.SearchHit{
				{Id: "123", Source: []byte("asdasd"), Score: gptr.Of(1.1)},
			}}})
			convey.So(err, convey.ShouldNotBeNil)
			convey.So(err.Error(), convey.ShouldContainSubstring, "[parseSearchResult] unexpected hit source type")
			convey.So(docs, convey.ShouldBeNil)
		})

		PatchConvey("test content type invalid", func() {
			b, err := json.Marshal(map[string]interface{}{
				DocFieldNameContent:                             123,
				string(GetDefaultVectorFieldKeyContent()):       []float64{1.1, 1.2},
				"ext_field_2":                                   "mock_extra_2",
				string(GetDefaultVectorFieldKey("ext_field_2")): []float64{2.1, 2.2},
			})
			convey.So(err, convey.ShouldBeNil)

			docs, err := r.parseSearchResult(&tes.SearchResult{Hits: &tes.SearchHits{Hits: []*tes.SearchHit{
				{Id: "123", Source: b, Score: gptr.Of(1.1)},
			}}})
			convey.So(err, convey.ShouldNotBeNil)
			convey.So(err.Error(), convey.ShouldContainSubstring, "[parseSearchResult] content type not string")
			convey.So(docs, convey.ShouldBeNil)
		})

		PatchConvey("test success", func() {
			b, err := json.Marshal(map[string]interface{}{
				DocFieldNameContent:                             "mock_content",
				string(GetDefaultVectorFieldKeyContent()):       []float64{1.1, 1.2},
				"ext_field_2":                                   "mock_extra_2",
				string(GetDefaultVectorFieldKey("ext_field_2")): []float64{2.1, 2.2},
			})
			convey.So(err, convey.ShouldBeNil)

			docs, err := r.parseSearchResult(&tes.SearchResult{Hits: &tes.SearchHits{Hits: []*tes.SearchHit{
				{Id: "123", Source: b, Score: gptr.Of(1.1)},
			}}})
			convey.So(err, convey.ShouldBeNil)
			convey.So(len(docs), convey.ShouldEqual, 1)

			expEsFields := map[string]interface{}{
				string(GetDefaultVectorFieldKeyContent()): []any{1.1, 1.2},
				"ext_field_2": "mock_extra_2",
				string(GetDefaultVectorFieldKey("ext_field_2")): []any{2.1, 2.2},
			}

			exp := &schema.Document{
				ID:      "123",
				Content: "mock_content",
				MetaData: map[string]any{
					docExtraKeyEsFields: expEsFields,
				},
			}
			exp.WithScore(1.1)
			convey.So(docs[0], convey.ShouldEqual, exp)
			fields, ok := GetExtraDataFields(docs[0])
			convey.So(ok, convey.ShouldBeTrue)
			convey.So(fields, convey.ShouldEqual, expEsFields)
		})
	})
}

func TestRetrieve(t *testing.T) {
	PatchConvey("test Retrieve", t, func() {
		ctx := context.Background()
		emb := &mockEmbedding{}
		cli := &tes.Client{}
		mockSearch := tes.NewSearchService(cli)

		PatchConvey("test knn with filter", func() {
			b, err := json.Marshal(map[string]interface{}{
				DocFieldNameContent:                             "mock_content",
				string(GetDefaultVectorFieldKeyContent()):       []float64{1.1, 1.2},
				"ext_field_2":                                   "mock_extra_2",
				string(GetDefaultVectorFieldKey("ext_field_2")): []float64{2.1, 2.2},
			})
			convey.So(err, convey.ShouldBeNil)
			searchResult := &tes.SearchResult{Hits: &tes.SearchHits{Hits: []*tes.SearchHit{
				{Id: "123", Source: b, Score: gptr.Of(1.1)},
			}}}

			Mock(GetMethod(cli, "Search")).Return(mockSearch).Build()
			Mock(GetMethod(mockSearch, "Do")).Return(searchResult, nil).Build()

			r := &Retriever{
				config: &RetrieverConfig{
					Index:          "mock_index",
					TopK:           10,
					ScoreThreshold: gptr.Of(0.1),
					SearchMode:     SearchModeKNN,
					Embedding:      emb,
				},
				client: cli,
			}

			query, err := NewKNNQuery(
				VectorFieldKV{Key: GetDefaultVectorFieldKeyContent(), Value: "mock_content"},
				VectorFieldKV{Key: GetDefaultVectorFieldKey("ext_field_2"), Value: "mock_extra_2"},
			)
			convey.So(err, convey.ShouldBeNil)

			dsl := JoinDSLFilters(map[string]interface{}{},
				DSLPair{
					Query:         tes.NewRawStringQuery("{\"match\":{\"extra_field\":\"mock_value\"}}"),
					ScoreFunction: tes.NewWeightFactorFunction(10),
				},
			)

			docs, err := r.Retrieve(ctx, query, retriever.WithDSLInfo(dsl))
			convey.So(err, convey.ShouldBeNil)
			convey.So(len(docs), convey.ShouldEqual, 1)

			expEsFields := map[string]interface{}{
				string(GetDefaultVectorFieldKeyContent()): []any{1.1, 1.2},
				"ext_field_2": "mock_extra_2",
				string(GetDefaultVectorFieldKey("ext_field_2")): []any{2.1, 2.2},
			}

			exp := &schema.Document{
				ID:      "123",
				Content: "mock_content",
				MetaData: map[string]any{
					docExtraKeyEsFields: expEsFields,
				},
			}
			exp.WithScore(1.1)
			convey.So(docs[0], convey.ShouldEqual, exp)
			fields, ok := GetExtraDataFields(docs[0])
			convey.So(ok, convey.ShouldBeTrue)
			convey.So(fields, convey.ShouldEqual, expEsFields)
		})
	})
}

type mockEmbedding struct{}

func (m mockEmbedding) EmbedStrings(ctx context.Context, texts []string, opts ...embedding.Option) ([][]float64, error) {
	return [][]float64{{1.1}, {2.1}}, nil
}
