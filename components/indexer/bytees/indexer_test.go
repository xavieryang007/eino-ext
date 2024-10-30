package bytees

import (
	"context"
	"fmt"
	"testing"

	. "github.com/bytedance/mockey"
	"github.com/smartystreets/goconvey/convey"

	"code.byted.org/flow/eino/components/embedding"
	"code.byted.org/flow/eino/components/indexer"
	"code.byted.org/flow/eino/schema"
)

func TestDefaultQueryRequest(t *testing.T) {
	PatchConvey("test defaultQueryRequest", t, func() {
		ctx := context.Background()
		i := &Indexer{
			config: &IndexerConfig{
				Index: "mock_index",
			},
		}

		d1 := &schema.Document{
			ID:       "123",
			Content:  "mock_content_1",
			MetaData: nil,
		}

		d2 := &schema.Document{
			ID:       "456",
			Content:  "mock_content_2",
			MetaData: nil,
		}

		SetExtraDataFields(d2, map[string]interface{}{
			"ext_field_1": 123,
		})

		docs := []*schema.Document{d1, d2}

		brs, err := i.defaultQueryRequest(ctx, docs, &indexer.Options{})
		convey.So(err, convey.ShouldBeNil)
		convey.So(len(brs), convey.ShouldEqual, 2)
		convey.So(brs[0].String(), convey.ShouldEqual, "{\"index\":{\"_index\":\"mock_index\",\"_id\":\"123\"}}\n{\"eino_doc_content\":\"mock_content_1\"}")
		convey.So(brs[1].String(), convey.ShouldEqual, "{\"index\":{\"_index\":\"mock_index\",\"_id\":\"456\"}}\n{\"eino_doc_content\":\"mock_content_2\",\"ext_field_1\":123}")
	})
}

func TestKnnQueryRequest(t *testing.T) {
	PatchConvey("test knnQueryRequest", t, func() {
		ctx := context.Background()
		d1 := &schema.Document{
			ID:       "123",
			Content:  "mock_content_1",
			MetaData: nil,
		}

		d2 := &schema.Document{
			ID:       "456",
			Content:  "mock_content_2",
			MetaData: nil,
		}

		SetExtraDataFields(d1, map[string]interface{}{
			"ext_field_1": 123,
			"ext_field_2": "123",
			"ext_field_3": "123",
		})

		SetExtraDataFields(d2, map[string]interface{}{
			"ext_field_1": "321",
			"ext_field_2": "321",
		})

		docs := []*schema.Document{d1, d2}
		emb := &mockEmbedding{}
		opt := &indexer.Options{
			Embedding: emb,
		}

		PatchConvey("test vector field name not found", func() {
			i := &Indexer{
				config: &IndexerConfig{
					Index: "mock_index",
					VectorFields: []VectorFieldKV{
						DefaultVectorFieldKV(DocFieldNameContent),
						DefaultVectorFieldKV("ext_field_3"),
					},
				},
			}

			brs, err := i.knnQueryRequest(ctx, docs, opt)
			convey.So(err, convey.ShouldNotBeNil)
			convey.So(err.Error(), convey.ShouldContainSubstring, "[knnQueryRequest] field name not found or type incorrect")
			convey.So(brs, convey.ShouldBeNil)
		})

		PatchConvey("test vector field type incorrect", func() {
			i := &Indexer{
				config: &IndexerConfig{
					Index: "mock_index",
					VectorFields: []VectorFieldKV{
						DefaultVectorFieldKV(DocFieldNameContent),
						DefaultVectorFieldKV("ext_field_1"),
					},
				},
			}

			brs, err := i.knnQueryRequest(ctx, docs, opt)
			convey.So(err, convey.ShouldNotBeNil)
			convey.So(err.Error(), convey.ShouldContainSubstring, "[knnQueryRequest] field name not found or type incorrect")
			convey.So(brs, convey.ShouldBeNil)
		})

		PatchConvey("test embedding error", func() {
			Mock(GetMethod(emb, "EmbedStrings")).Return(nil, fmt.Errorf("mock err")).Build()
			i := &Indexer{
				config: &IndexerConfig{
					Index: "mock_index",
					VectorFields: []VectorFieldKV{
						DefaultVectorFieldKV(DocFieldNameContent),
						DefaultVectorFieldKV("ext_field_2"),
					},
				},
			}

			brs, err := i.knnQueryRequest(ctx, docs, opt)
			convey.So(err, convey.ShouldNotBeNil)
			convey.So(err.Error(), convey.ShouldContainSubstring, "mock err")
			convey.So(brs, convey.ShouldBeNil)
		})

		PatchConvey("test embedding resp length invalid", func() {
			Mock(GetMethod(emb, "EmbedStrings")).Return([][]float64{{1.1}}, nil).Build()
			i := &Indexer{
				config: &IndexerConfig{
					Index: "mock_index",
					VectorFields: []VectorFieldKV{
						DefaultVectorFieldKV(DocFieldNameContent),
						DefaultVectorFieldKV("ext_field_2"),
					},
				},
			}

			brs, err := i.knnQueryRequest(ctx, docs, opt)
			convey.So(err, convey.ShouldNotBeNil)
			convey.So(err.Error(), convey.ShouldContainSubstring, "[knnQueryRequest] invalid vector length")
			convey.So(brs, convey.ShouldBeNil)
		})

		PatchConvey("test success", func() {
			i := &Indexer{
				config: &IndexerConfig{
					Index: "mock_index",
					VectorFields: []VectorFieldKV{
						DefaultVectorFieldKV(DocFieldNameContent),
						DefaultVectorFieldKV("ext_field_2"),
					},
				},
			}

			brs, err := i.knnQueryRequest(ctx, docs, opt)
			convey.So(err, convey.ShouldBeNil)
			convey.So(len(brs), convey.ShouldEqual, 2)
			convey.So(brs[0].String(), convey.ShouldEqual, "{\"index\":{\"_index\":\"mock_index\",\"_id\":\"123\"}}\n{\"eino_doc_content\":\"mock_content_1\",\"ext_field_1\":123,\"ext_field_2\":\"123\",\"ext_field_3\":\"123\",\"vector_eino_doc_content\":[1.1],\"vector_ext_field_2\":[2.1]}")
			convey.So(brs[1].String(), convey.ShouldEqual, "{\"index\":{\"_index\":\"mock_index\",\"_id\":\"456\"}}\n{\"eino_doc_content\":\"mock_content_2\",\"ext_field_1\":\"321\",\"ext_field_2\":\"321\",\"vector_eino_doc_content\":[1.1],\"vector_ext_field_2\":[2.1]}")
		})
	})
}

type mockEmbedding struct{}

func (m mockEmbedding) EmbedStrings(ctx context.Context, texts []string, opts ...embedding.Option) ([][]float64, error) {
	return [][]float64{{1.1}, {2.1}}, nil
}
