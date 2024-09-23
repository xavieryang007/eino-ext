package vikingdb

import (
	"context"
	"fmt"
	"testing"

	. "github.com/bytedance/mockey"
	"github.com/smartystreets/goconvey/convey"
	"go.uber.org/mock/gomock"

	"code.byted.org/flow/eino/components/retriever"
	viking "code.byted.org/lagrange/viking_go_client"
	"code.byted.org/lang/gg/gptr"

	"code.byted.org/flow/eino-ext/components/retriever/vikingdb/internal/mock/embedding"
)

func TestNewRetriever(t *testing.T) {
	PatchConvey("test NewRetriever", t, func() {
		ctx := context.Background()
		ctrl := gomock.NewController(t)
		emb := embedding.NewMockEmbedder(ctrl)

		r, err := NewRetriever(ctx, &RetrieverConfig{EmbeddingConfig: EmbeddingConfig{Embedding: nil}})
		convey.So(err, convey.ShouldNotBeNil)
		convey.So(r, convey.ShouldBeNil)

		r, err = NewRetriever(ctx, &RetrieverConfig{EmbeddingConfig: EmbeddingConfig{Embedding: emb}})
		convey.So(err, convey.ShouldBeNil)
		convey.So(r, convey.ShouldNotBeNil)
	})
}

func TestEmbeddingQuery(t *testing.T) {
	PatchConvey("test embeddingQuery", t, func() {
		ctx := context.Background()
		ctrl := gomock.NewController(t)
		emb := embedding.NewMockEmbedder(ctrl)
		query := "mock_query"
		r, err := NewRetriever(ctx, &RetrieverConfig{EmbeddingConfig: EmbeddingConfig{Embedding: emb}})
		convey.So(err, convey.ShouldBeNil)
		convey.So(r, convey.ShouldNotBeNil)

		PatchConvey("test EmbedStrings failed", func() {
			emb.EXPECT().EmbedStrings(gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("mock err")).Times(1)

			vector, err := r.embeddingQuery(ctx, query, &retriever.Options{})
			convey.So(err, convey.ShouldNotBeNil)
			convey.So(vector, convey.ShouldBeNil)
		})

		PatchConvey("test vector length invalid", func() {
			emb.EXPECT().EmbedStrings(gomock.Any(), gomock.Any()).Return([][]float64{}, nil).Times(1)

			vector, err := r.embeddingQuery(ctx, query, &retriever.Options{})
			convey.So(err, convey.ShouldNotBeNil)
			convey.So(vector, convey.ShouldBeNil)
		})

		PatchConvey("test success", func() {
			emb.EXPECT().EmbedStrings(gomock.Any(), gomock.Any()).Return([][]float64{{1.1, 1.2, 3.5}}, nil).Times(1)

			vector, err := r.embeddingQuery(ctx, query, &retriever.Options{})
			convey.So(err, convey.ShouldBeNil)
			convey.So(vector, convey.ShouldNotBeNil)
		})
	})
}

func TestRetrieverWithVector(t *testing.T) {
	PatchConvey("test retrieverWithVector", t, func() {
		ctx := context.Background()
		ctrl := gomock.NewController(t)
		emb := embedding.NewMockEmbedder(ctrl)
		vector := []float64{1.1, 1.2, 3.5}
		r, err := NewRetriever(ctx, &RetrieverConfig{EmbeddingConfig: EmbeddingConfig{Embedding: emb}, ScoreThreshold: gptr.Of(20.24)})
		convey.So(err, convey.ShouldBeNil)
		convey.So(r, convey.ShouldNotBeNil)

		PatchConvey("test Recall failed", func() {
			Mock(GetMethod(r.client, "Recall")).Return(nil, "", fmt.Errorf("mock err")).Build()

			resp, err := r.retrieverWithVector(ctx, vector, nil, &retriever.Options{})
			convey.So(err, convey.ShouldNotBeNil)
			convey.So(resp, convey.ShouldBeNil)
		})

		PatchConvey("test Recalled nothing", func() {
			Mock(GetMethod(r.client, "Recall")).Return(nil, "", nil).Build()

			resp, err := r.retrieverWithVector(ctx, vector, nil, &retriever.Options{})
			convey.So(err, convey.ShouldBeNil)
			convey.So(resp, convey.ShouldNotBeNil)
			convey.So(len(resp), convey.ShouldEqual, 0)
		})

		PatchConvey("test success", func() {
			Mock(GetMethod(r.client, "Recall")).Return(&viking.RecallResp{
				Code:    0,
				Message: "",
				Result: []*viking.SimpleRecallRpcResult{
					{
						Scores:       30.30,
						LabelLower64: 12345,
						Attrs:        "mock1",
					},
					{
						Scores:       10.30,
						LabelLower64: 12346,
						Attrs:        "mock2",
					},
				},
			}, "", nil).Build()

			resp, err := r.retrieverWithVector(ctx, vector, nil, &retriever.Options{})
			convey.So(err, convey.ShouldBeNil)
			convey.So(resp, convey.ShouldNotBeNil)
			convey.So(len(resp), convey.ShouldEqual, 1)
		})
	})
}

func TestEmbeddingRaw(t *testing.T) {
	PatchConvey("test embeddingRaw", t, func() {
		ctx := context.Background()
		mockClient := &viking.VikingDbClient{}
		Mock(viking.NewVikingDbClient).Return(mockClient).Build()

		query := "mock_query"
		dense := []float64{1.1, 1.2, 3.5}
		sparse := []interface{}{
			`"you", 0.3`,
			`"looks", 0.4`,
			`"good", 0.5`,
		}

		PatchConvey("test RawEmbedding error", func() {
			Mock(GetMethod(mockClient, "RawEmbedding")).Return(nil, "", fmt.Errorf("mock err")).Build()

			r, err := NewRetriever(ctx, &RetrieverConfig{EmbeddingConfig: EmbeddingConfig{UseBuiltin: true, UseSparse: false}})
			convey.So(err, convey.ShouldBeNil)
			convey.So(r, convey.ShouldNotBeNil)

			d, s, err := r.embeddingRaw(ctx, query, &retriever.Options{})
			convey.So(err, convey.ShouldNotBeNil)
			convey.So(s, convey.ShouldBeNil)
			convey.So(d, convey.ShouldBeNil)

		})

		PatchConvey("test RawEmbedding dense length invalid", func() {
			Mock(GetMethod(mockClient, "RawEmbedding")).Return([][]float32{{}, {}}, "", nil).Build()

			r, err := NewRetriever(ctx, &RetrieverConfig{EmbeddingConfig: EmbeddingConfig{UseBuiltin: true, UseSparse: false}})
			convey.So(err, convey.ShouldBeNil)
			convey.So(r, convey.ShouldNotBeNil)

			d, s, err := r.embeddingRaw(ctx, query, &retriever.Options{})
			convey.So(err, convey.ShouldNotBeNil)
			convey.So(s, convey.ShouldBeNil)
			convey.So(d, convey.ShouldBeNil)
		})

		PatchConvey("test RawEmbedding success", func() {
			Mock(GetMethod(mockClient, "RawEmbedding")).Return([][]float32{f64To32(dense)}, "", nil).Build()

			r, err := NewRetriever(ctx, &RetrieverConfig{EmbeddingConfig: EmbeddingConfig{UseBuiltin: true, UseSparse: false}})
			convey.So(err, convey.ShouldBeNil)
			convey.So(r, convey.ShouldNotBeNil)

			d, s, err := r.embeddingRaw(ctx, query, &retriever.Options{})
			convey.So(err, convey.ShouldBeNil)
			convey.So(s, convey.ShouldBeNil)
			convey.So(len(d), convey.ShouldEqual, len(dense))
		})

		PatchConvey("test RawEmbeddingWithSparse error", func() {
			Mock(GetMethod(mockClient, "RawEmbeddingWithSparse")).Return(nil, nil, "", fmt.Errorf("mock err")).Build()

			r, err := NewRetriever(ctx, &RetrieverConfig{EmbeddingConfig: EmbeddingConfig{UseBuiltin: true, UseSparse: true}})
			convey.So(err, convey.ShouldBeNil)
			convey.So(r, convey.ShouldNotBeNil)

			d, s, err := r.embeddingRaw(ctx, query, &retriever.Options{})
			convey.So(err, convey.ShouldNotBeNil)
			convey.So(s, convey.ShouldBeNil)
			convey.So(d, convey.ShouldBeNil)
		})

		PatchConvey("test RawEmbeddingWithSparse dense length error", func() {
			Mock(GetMethod(mockClient, "RawEmbeddingWithSparse")).Return([][]float32{{}, {}}, nil, "", nil).Build()

			r, err := NewRetriever(ctx, &RetrieverConfig{EmbeddingConfig: EmbeddingConfig{UseBuiltin: true, UseSparse: true}})
			convey.So(err, convey.ShouldBeNil)
			convey.So(r, convey.ShouldNotBeNil)

			d, s, err := r.embeddingRaw(ctx, query, &retriever.Options{})
			convey.So(err, convey.ShouldNotBeNil)
			convey.So(s, convey.ShouldBeNil)
			convey.So(d, convey.ShouldBeNil)
		})

		PatchConvey("test RawEmbeddingWithSparse sparse length error", func() {
			Mock(GetMethod(mockClient, "RawEmbeddingWithSparse")).Return([][]float32{f64To32(dense)}, [][]interface{}{{}, {}}, "", nil).Build()

			r, err := NewRetriever(ctx, &RetrieverConfig{EmbeddingConfig: EmbeddingConfig{UseBuiltin: true, UseSparse: true}})
			convey.So(err, convey.ShouldBeNil)
			convey.So(r, convey.ShouldNotBeNil)

			d, s, err := r.embeddingRaw(ctx, query, &retriever.Options{})
			convey.So(err, convey.ShouldNotBeNil)
			convey.So(s, convey.ShouldBeNil)
			convey.So(d, convey.ShouldBeNil)
		})

		PatchConvey("test RawEmbeddingWithSparse success", func() {
			Mock(GetMethod(mockClient, "RawEmbeddingWithSparse")).Return([][]float32{f64To32(dense)}, [][]interface{}{sparse}, "", nil).Build()

			r, err := NewRetriever(ctx, &RetrieverConfig{EmbeddingConfig: EmbeddingConfig{UseBuiltin: true, UseSparse: true}})
			convey.So(err, convey.ShouldBeNil)
			convey.So(r, convey.ShouldNotBeNil)

			d, s, err := r.embeddingRaw(ctx, query, &retriever.Options{})
			convey.So(err, convey.ShouldBeNil)
			convey.So(len(s), convey.ShouldEqual, len(sparse))
			convey.So(len(d), convey.ShouldEqual, len(dense))
		})
	})
}
