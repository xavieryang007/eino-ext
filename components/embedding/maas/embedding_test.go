package maas

import (
	"context"
	"fmt"
	"testing"

	"code.byted.org/flow/eino/components/embedding"
	. "github.com/bytedance/mockey"
	"github.com/smartystreets/goconvey/convey"
	"github.com/volcengine/volcengine-go-sdk/service/arkruntime"
	"github.com/volcengine/volcengine-go-sdk/service/arkruntime/model"
)

func Test_EmbedStrings(t *testing.T) {
	PatchConvey("test buildClient", t, func() {
		buildClient(&EmbeddingConfig{
			AccessKey:  "mock",
			SecretKey:  "mock",
			BaseURL:    "mock",
			Dimensions: 1,
			Model:      "mock",
			Region:     "mock",
			User:       "mock",
		})

		buildClient(&EmbeddingConfig{
			APIKey:     "mock",
			Dimensions: 1,
			Model:      "mock",
			User:       "mock",
		})
	})
	PatchConvey("test EmbedStrings", t, func() {
		ctx := context.Background()
		mockCli := &arkruntime.Client{}
		Mock(buildClient).Return(mockCli).Build()

		embedder, err := NewEmbedder(ctx, &EmbeddingConfig{})
		convey.So(err, convey.ShouldBeNil)

		PatchConvey("test embedding error", func() {
			Mock(GetMethod(mockCli, "CreateEmbeddings")).Return(nil, fmt.Errorf("mock err")).Build()

			vector, err := embedder.EmbedStrings(ctx, []string{"asd"})
			convey.So(err, convey.ShouldNotBeNil)
			convey.So(len(vector), convey.ShouldEqual, 0)
		})

		PatchConvey("test embedding success", func() {
			Mock(GetMethod(mockCli, "CreateEmbeddings")).Return(model.EmbeddingResponse{
				Data: []model.Embedding{
					{
						Embedding: []float32{1, 2, 3},
						Index:     0,
						Object:    "embedding",
					},
				},
				Usage: model.Usage{
					CompletionTokens: 1,
					PromptTokens:     2,
					TotalTokens:      3,
				},
			}, nil).Build()

			vector, err := embedder.EmbedStrings(ctx, []string{"asd"}, embedding.WithModel("mock"))
			convey.So(err, convey.ShouldBeNil)
			convey.So(len(vector), convey.ShouldEqual, 1)
		})
	})
}
