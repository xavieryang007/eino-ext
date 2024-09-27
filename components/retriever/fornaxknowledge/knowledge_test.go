package fornaxknowledge

import (
	"context"
	"testing"

	"github.com/bytedance/mockey"
	"github.com/smartystreets/goconvey/convey"
	"go.uber.org/mock/gomock"

	"code.byted.org/flow/eino/compose"
	"code.byted.org/flow/eino/schema"
	"code.byted.org/flowdevops/fornax_sdk"
	fknowledge "code.byted.org/flowdevops/fornax_sdk/domain/knowledge"

	"code.byted.org/flow/eino-ext/components/retriever/fornaxknowledge/internal/mock/fornax"
)

func TestKnowledge(t *testing.T) {

	mockey.PatchConvey("test AddDocuments", t, func() {
		ctx := context.Background()
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockFornaxCli := fornax.NewMockIClient(ctrl)

		defer mockey.Mock(fornax_sdk.NewClient).Return(nil, nil).Build().UnPatch()

		r, err := NewKnowledgeRetriever(ctx, &Config{
			AK: "ak",
			SK: "sk",
		})
		convey.So(err, convey.ShouldBeNil)

		r.client = mockFornaxCli

		mockey.PatchConvey("test success", func() {
			mockFornaxCli.EXPECT().RetrieveKnowledge(gomock.Any(), gomock.Any()).Return(&fknowledge.RetrieveKnowledgeResult{
				Data: &fknowledge.RecallData{
					Items: []*fknowledge.Item{
						{DocID: "1", Slice: "001", SliceMeta: `{}`},
						{DocID: "2", Slice: "002"},
					},
				},
			}, nil).Times(1)

			docs, err := r.Retrieve(ctx, "test")
			convey.So(err, convey.ShouldBeNil)

			convey.So(len(docs), convey.ShouldEqual, 2)
			convey.So(docs[0].ID, convey.ShouldEqual, "1")
			convey.So(docs[0].Content, convey.ShouldEqual, "001")
			convey.So(docs[1].ID, convey.ShouldEqual, "2")
			convey.So(docs[1].Content, convey.ShouldEqual, "002")
		})
	})

}

func TestKnowledgeStream(t *testing.T) {

	mockey.PatchConvey("test AddDocuments with stream", t, func() {
		ctx := context.Background()
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockFornaxCli := fornax.NewMockIClient(ctrl)

		defer mockey.Mock(fornax_sdk.NewClient).Return(nil, nil).Build().UnPatch()

		r, err := NewKnowledgeRetriever(ctx, &Config{
			AK: "ak",
			SK: "sk",
		})
		convey.So(err, convey.ShouldBeNil)

		r.client = mockFornaxCli

		mockey.PatchConvey("test success", func() {
			mockFornaxCli.EXPECT().RetrieveKnowledge(gomock.Any(), gomock.Any()).Return(&fknowledge.RetrieveKnowledgeResult{
				Data: &fknowledge.RecallData{
					Items: []*fknowledge.Item{
						{DocID: "1", Slice: "001", SliceMeta: `{}`},
						{DocID: "2", Slice: "002"},
					},
				},
			}, nil).Times(1)

			c := compose.NewChain[string, []*schema.Document]()
			c.AppendRetriever(r)

			run, err := c.Compile(ctx)
			convey.So(err, convey.ShouldBeNil)

			docsReader, err := run.Stream(ctx, "test")
			convey.So(err, convey.ShouldBeNil)

			defer docsReader.Close()

			docs := make([]*schema.Document, 0)

			for {
				doc, err := docsReader.Recv()
				if err != nil {
					break
				}
				docs = append(docs, doc...)
			}

			convey.So(len(docs), convey.ShouldEqual, 2)
			convey.So(docs[0].ID, convey.ShouldEqual, "1")
			convey.So(docs[0].Content, convey.ShouldEqual, "001")
			convey.So(docs[1].ID, convey.ShouldEqual, "2")
			convey.So(docs[1].Content, convey.ShouldEqual, "002")
		})
	})

}
