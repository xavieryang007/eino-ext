package vikingdb

import (
	"context"
	"fmt"
	"testing"

	. "github.com/bytedance/mockey"
	"github.com/smartystreets/goconvey/convey"
	"go.uber.org/mock/gomock"

	"code.byted.org/flow/eino/components/indexer"
	"code.byted.org/flow/eino/schema"

	"code.byted.org/flow/eino-ext/components/indexer/vikingdb/internal/mock/embedding"
)

func TestNewIndexer(t *testing.T) {
	PatchConvey("test NewIndexer", t, func() {
		ctx := context.Background()
		ctrl := gomock.NewController(t)
		emb := embedding.NewMockEmbedder(ctrl)

		idx, err := NewIndexer(ctx, &IndexerConfig{EmbeddingConfig: EmbeddingConfig{Embedding: nil}})
		convey.So(err, convey.ShouldNotBeNil)
		convey.So(idx, convey.ShouldBeNil)

		idx, err = NewIndexer(ctx, &IndexerConfig{EmbeddingConfig: EmbeddingConfig{Embedding: emb}})
		convey.So(err, convey.ShouldBeNil)
		convey.So(idx, convey.ShouldNotBeNil)
	})
}

func TestEmbeddingDocs(t *testing.T) {
	PatchConvey("test embeddingDocs", t, func() {
		ctx := context.Background()
		ctrl := gomock.NewController(t)
		emb := embedding.NewMockEmbedder(ctrl)
		docs := []*schema.Document{{Content: "asd"}, {Content: "qwe"}}
		idx, err := NewIndexer(ctx, &IndexerConfig{AddBatchSize: 1, EmbeddingConfig: EmbeddingConfig{Embedding: emb}})
		convey.So(err, convey.ShouldBeNil)
		convey.So(idx, convey.ShouldNotBeNil)

		PatchConvey("test embedding failed", func() {
			emb.EXPECT().EmbedStrings(gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("mock err")).Times(1)

			resp, err := idx.embeddingDocs(ctx, docs, &indexer.Options{})
			convey.So(err, convey.ShouldNotBeNil)
			convey.So(resp, convey.ShouldBeNil)
		})

		PatchConvey("test resp vector size invalid", func() {
			emb.EXPECT().EmbedStrings(gomock.Any(), gomock.Any()).Return([][]float64{{1.0}}, nil).Times(1)

			resp, err := idx.embeddingDocs(ctx, docs, &indexer.Options{})
			convey.So(err, convey.ShouldNotBeNil)
			convey.So(resp, convey.ShouldBeNil)
		})

		PatchConvey("test success", func() {
			emb.EXPECT().EmbedStrings(gomock.Any(), gomock.Any()).Return([][]float64{{1.0}, {2.1}}, nil).Times(1)

			resp, err := idx.embeddingDocs(ctx, docs, &indexer.Options{})
			convey.So(err, convey.ShouldBeNil)
			convey.So(resp, convey.ShouldNotBeNil)
		})
	})
}

func TestStoreWithVector(t *testing.T) {
	PatchConvey("test storeWithVector", t, func() {
		ctx := context.Background()
		ctrl := gomock.NewController(t)
		emb := embedding.NewMockEmbedder(ctrl)

		d1 := &schema.Document{ID: "123", Content: "asd"}
		d2 := &schema.Document{ID: "124", Content: "qwe"}

		dvs := []*schema.Document{
			d1.WithVector([]float64{1.1, 1.2}),
			d2.WithVector([]float64{2.1, 2.2}),
		}

		idx, err := NewIndexer(ctx, &IndexerConfig{AddBatchSize: 1, EmbeddingConfig: EmbeddingConfig{Embedding: emb}})
		convey.So(err, convey.ShouldBeNil)
		convey.So(idx, convey.ShouldNotBeNil)

		PatchConvey("test AddData failed", func() {
			Mock(GetMethod(idx.client, "AddData")).Return(nil, "", fmt.Errorf("mock err")).Build()

			resp, err := idx.storeWithVector(ctx, dvs, &indexer.Options{})
			convey.So(err, convey.ShouldNotBeNil)
			convey.So(resp, convey.ShouldBeNil)
		})

		PatchConvey("test AddData success", func() {
			Mock(GetMethod(idx.client, "AddData")).Return([]string{"0321"}, "", nil).Build()

			resp, err := idx.storeWithVector(ctx, dvs, &indexer.Options{})
			convey.So(err, convey.ShouldBeNil)
			convey.So(resp, convey.ShouldNotBeNil)
		})
	})
}

func TestStoreRaw(t *testing.T) {
	PatchConvey("test storeRaw", t, func() {
		ctx := context.Background()
		docs := []*schema.Document{
			{ID: "123", Content: "asd"},
			{ID: "124", Content: "qwe"},
		}

		idx, err := NewIndexer(ctx, &IndexerConfig{AddBatchSize: 1, EmbeddingConfig: EmbeddingConfig{UseBuiltin: true}})
		convey.So(err, convey.ShouldBeNil)
		convey.So(idx, convey.ShouldNotBeNil)

		PatchConvey("test AddRawData failed", func() {
			Mock(GetMethod(idx.client, "AddRawData")).Return(nil, "", fmt.Errorf("mock err")).Build()

			resp, err := idx.storeRaw(ctx, docs, &indexer.Options{})
			convey.So(err, convey.ShouldNotBeNil)
			convey.So(resp, convey.ShouldBeNil)
		})

		PatchConvey("test AddRawData success", func() {
			Mock(GetMethod(idx.client, "AddRawData")).Return([]string{"0321"}, "", nil).Build()

			resp, err := idx.storeRaw(ctx, docs, &indexer.Options{})
			convey.So(err, convey.ShouldBeNil)
			convey.So(resp, convey.ShouldNotBeNil)
		})
	})
}
