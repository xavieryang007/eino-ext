package fornax

import (
	"context"
	"testing"

	. "github.com/bytedance/mockey"
	"github.com/smartystreets/goconvey/convey"

	"code.byted.org/data/mario_collector"
	"code.byted.org/flow/eino/callbacks"
	"code.byted.org/flow/eino/components"
	"code.byted.org/flow/eino/components/model"
	"code.byted.org/flow/eino/compose"
	"code.byted.org/flow/eino/schema"
	"code.byted.org/flowdevops/fornax_sdk"
	"code.byted.org/flowdevops/fornax_sdk/infra/ob"
)

func Test_FornaxMetrics_ChatModel(t *testing.T) {
	mockers := []*Mocker{
		Mock(mario_collector.NewMarioCollector).Return(&mario_collector.MarioCollector{}).Build(),
		Mock((*mario_collector.MarioCollector).CollectEvent).Return(nil).Build(),
		Mock((*fornax_sdk.Client).GetSpaceID).Return(123).Build(),
	}

	defer func() {
		for _, mocker := range mockers {
			mocker.UnPatch()
		}
	}()

	if err := ob.MustInit(nil); err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	client := &fornax_sdk.Client{}
	handler := newMetricsCallbackHandler(client, &options{})

	PatchConvey("test einoMetrics ChatModel", t, func() {
		info := &callbacks.RunInfo{
			Name:      "mock_name",
			Type:      "mock_type",
			Component: components.ComponentOfChatModel,
		}

		PatchConvey("test non stream", func() {
			c1 := handler.OnStart(ctx, info, &model.CallbackInput{
				Config: &model.Config{
					Model: "mock_model",
				},
			})
			convey.So(c1.Value(metricsVariablesKey{}).(*metricsVariablesValue), convey.ShouldNotBeNil)

			output := &model.CallbackOutput{
				Message: &schema.Message{
					Role:    schema.Assistant,
					Content: "asd",
					Name:    "name",
				},
				Config: &model.Config{
					Model: "mock_model",
				},
				TokenUsage: &model.TokenUsage{
					PromptTokens:     1,
					CompletionTokens: 2,
					TotalTokens:      3,
				},
			}

			handler.OnEnd(c1, nil, &model.CallbackOutput{})
			handler.OnEnd(ctx, info, output)
			handler.OnEnd(c1, info, output)

			handler.OnError(c1, nil, nil)
			handler.OnError(ctx, info, nil)
		})

		PatchConvey("test stream", func() {
			sri, _ := schema.Pipe[callbacks.CallbackInput](1)
			c1 := handler.OnStartWithStreamInput(ctx, nil, sri)
			convey.So(c1.Value(metricsVariablesKey{}).(*metricsVariablesValue), convey.ShouldNotBeNil)

			info := &callbacks.RunInfo{
				Name:      "mock_name",
				Type:      "mock_type",
				Component: components.ComponentOfChatModel,
			}

			sro, sw := schema.Pipe[callbacks.CallbackOutput](1)
			go func() {
				for i := 0; i < 2; i++ {
					sw.Send(&model.CallbackOutput{
						TokenUsage: &model.TokenUsage{
							PromptTokens:     0,
							CompletionTokens: 0,
							TotalTokens:      0,
						},
					}, nil)
				}

				sw.Close()
			}()

			handler.OnEndWithStreamOutput(c1, info, sro)
		})
	})

	PatchConvey("test Graph", t, func() {
		info := &callbacks.RunInfo{
			Name:      "mock_name",
			Type:      "mock_type",
			Component: compose.ComponentOfGraph,
		}

		c1 := handler.OnStart(ctx, info, "asd")

		val, ok := getMetricsVariablesValue(c1)
		convey.So(ok, convey.ShouldBeTrue)
		convey.So(val, convey.ShouldNotBeNil)

		name := getMetricsGraphName(c1)
		convey.So(name, convey.ShouldEqual, info.Name)

		handler.OnEnd(c1, nil, "qwe")
		handler.OnEnd(ctx, info, "qwe")
		handler.OnEnd(c1, info, "qwe")

		handler.OnError(c1, nil, nil)
		handler.OnError(ctx, info, nil)
	})
}
