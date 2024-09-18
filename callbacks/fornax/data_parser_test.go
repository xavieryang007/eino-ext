package fornax

import (
	"context"
	"fmt"
	"strings"
	"testing"

	. "github.com/bytedance/mockey"
	"github.com/smartystreets/goconvey/convey"

	"code.byted.org/flow/eino/callbacks"
	"code.byted.org/flow/eino/components"
	"code.byted.org/flow/eino/components/model"
	"code.byted.org/flow/eino/schema"
	"code.byted.org/flow/flow-telemetry-common/go/obtag"
)

func Test_ParseInput(t *testing.T) {
	PatchConvey("test ParseInput", t, func() {
		ctx := context.Background()
		parser := defaultDataParser{}
		tags := parser.ParseInput(ctx, &callbacks.RunInfo{Component: components.ComponentOfChatModel}, &model.CallbackInput{
			Messages: []*schema.Message{
				{
					Role:    schema.Assistant,
					Content: "asd",
					Name:    "name",
				},
				{
					Role:    schema.Assistant,
					Content: "qwe",
					Name:    "name",
				},
			},
			Config: &model.Config{
				Model: "mock_name",
			},
		})

		convey.So(tags, convey.ShouldNotBeNil)
		convey.So(tags[obtag.Input], convey.ShouldNotBeZeroValue)
		convey.So(tags[obtag.ModelName], convey.ShouldEqual, "mock_name")
	})
}

func Test_ParseOutput(t *testing.T) {
	PatchConvey("test ParseOutput", t, func() {
		ctx := context.Background()
		parser := defaultDataParser{}
		tags := parser.ParseOutput(ctx, &callbacks.RunInfo{Component: components.ComponentOfChatModel}, &model.CallbackOutput{
			Message: &schema.Message{
				Role:    schema.Assistant,
				Content: "asd",
				Name:    "name",
			},
			Config: &model.Config{
				Model: "mock_name",
			},
			TokenUsage: &model.TokenUsage{
				PromptTokens:     1,
				CompletionTokens: 2,
				TotalTokens:      3,
			},
		})

		convey.So(tags, convey.ShouldNotBeNil)
		convey.So(tags[obtag.Output], convey.ShouldNotBeZeroValue)
		convey.So(tags[obtag.ModelName], convey.ShouldEqual, "mock_name")
		convey.So(tags[obtag.InputTokens], convey.ShouldEqual, 1)
		convey.So(tags[obtag.OutputTokens], convey.ShouldEqual, 2)
		convey.So(tags[obtag.Tokens], convey.ShouldEqual, 3)
	})
}

func Test_ParseStreamInput(t *testing.T) {
	PatchConvey("test ParseStreamInput", t, func() {
		ctx := context.Background()
		parser := defaultDataParser{}
		makeStream := func(outputs []*model.CallbackInput, err error) *schema.StreamReader[callbacks.CallbackInput] {
			st := schema.NewStream[callbacks.CallbackInput](1)
			go func() {
				defer func() {
					st.Finish()
				}()

				for i := range outputs {
					if closed := st.Send(outputs[i], nil); closed {
						break
					}
				}

				if err != nil {
					st.Send(nil, err)
				}
			}()

			return st.AsReader()
		}

		PatchConvey("test error", func() {
			info := &callbacks.RunInfo{Component: components.ComponentOfChatModel}
			reader := makeStream([]*model.CallbackInput{
				{
					Messages: []*schema.Message{
						{
							Role:    schema.Assistant,
							Content: "asd",
							Name:    "name",
						},
					},
					Config: &model.Config{
						Model: "mock_model_name",
					},
				},
			}, fmt.Errorf("mock err"))

			tags := parser.ParseStreamInput(ctx, info, reader)
			reader.Close()
			convey.So(tags, convey.ShouldNotBeNil)
			convey.So(tags[obtag.Error], convey.ShouldEqual, "mock err")
			convey.So(tags[obtag.StatusCode], convey.ShouldEqual, obtag.VErrDefault)
		})

		PatchConvey("test success", func() {
			info := &callbacks.RunInfo{Component: components.ComponentOfChatModel}
			reader := makeStream([]*model.CallbackInput{
				{
					Messages: []*schema.Message{
						{
							Role:    schema.Assistant,
							Content: "asd",
							Name:    "name",
						},
					},
					Config: &model.Config{
						Model: "mock_model_name",
					},
				},
				{
					Messages: []*schema.Message{
						{
							Role:    schema.Assistant,
							Content: "asd",
							Name:    "name",
						},
					},
					Config: &model.Config{
						Model: "mock_model_name",
					},
				},
			}, nil)

			tags := parser.ParseStreamInput(ctx, info, reader)
			reader.Close()
			convey.So(tags, convey.ShouldNotBeNil)
			convey.So(tags[obtag.Input], convey.ShouldNotBeZeroValue)
		})
	})
}

func Test_ParseStreamOutput(t *testing.T) {
	PatchConvey("test ParseStreamOutput", t, func() {
		ctx := context.Background()
		parser := defaultDataParser{}
		makeStream := func(outputs []*model.CallbackOutput, err error) *schema.StreamReader[callbacks.CallbackOutput] {
			st := schema.NewStream[callbacks.CallbackOutput](1)
			go func() {
				defer func() {
					st.Finish()
				}()

				for i := range outputs {
					if closed := st.Send(outputs[i], nil); closed {
						break
					}
				}

				if err != nil {
					st.Send(nil, err)
				}
			}()

			return st.AsReader()
		}

		PatchConvey("test error", func() {
			info := &callbacks.RunInfo{Component: components.ComponentOfChatModel}
			reader := makeStream([]*model.CallbackOutput{
				{
					Message: &schema.Message{
						Role:    schema.Assistant,
						Content: "asd",
						Name:    "name",
					},
					Config: &model.Config{
						Model: "mock_model_name",
					},
					TokenUsage: nil,
				},
			}, fmt.Errorf("mock err"))

			tags := parser.ParseStreamOutput(ctx, info, reader)
			reader.Close()
			convey.So(tags, convey.ShouldNotBeNil)
			convey.So(tags[obtag.Error], convey.ShouldEqual, "mock err")
			convey.So(tags[obtag.StatusCode], convey.ShouldEqual, obtag.VErrDefault)
		})

		PatchConvey("test ComponentOfChatModel", func() {
			info := &callbacks.RunInfo{Component: components.ComponentOfChatModel}
			reader := makeStream([]*model.CallbackOutput{
				{
					Message: &schema.Message{
						Role:    schema.Assistant,
						Content: "asd",
						Name:    "name",
					},
					Config: &model.Config{
						Model: "mock_model_name",
					},
					TokenUsage: nil,
				},
				{
					Message: &schema.Message{
						Role:    schema.Assistant,
						Content: "qwe",
						Name:    "name",
					},
					Config: &model.Config{
						Model: "mock_model_name",
					},
					TokenUsage: &model.TokenUsage{
						PromptTokens:     1,
						CompletionTokens: 2,
						TotalTokens:      3,
					},
				},
			}, nil)

			tags := parser.ParseStreamOutput(ctx, info, reader)
			reader.Close()
			convey.So(tags, convey.ShouldNotBeNil)
			convey.So(tags[obtag.ModelName], convey.ShouldEqual, "mock_model_name")
			convey.So(tags[obtag.Output], convey.ShouldNotBeZeroValue)
			convey.So(tags[obtag.InputTokens], convey.ShouldEqual, 1)
			convey.So(tags[obtag.OutputTokens], convey.ShouldEqual, 2)
			convey.So(tags[obtag.Tokens], convey.ShouldEqual, 3)
		})
	})
}

func Test_parseAny(t *testing.T) {
	ctx := context.Background()
	PatchConvey("test parseAny", t, func() {
		convey.So(len(parseAny(ctx, []*schema.Message{
			{
				Role:    schema.Assistant,
				Content: "a",
				Name:    "name",
			},
			{
				Role:    schema.Assistant,
				Content: "b",
				Name:    "name",
			},
		})), convey.ShouldNotBeZeroValue)

		convey.So(len(parseAny(ctx, []*schema.Message{
			{
				Role:    schema.Assistant,
				Content: "a",
				Name:    "name",
			},
			{
				Role:    schema.System,
				Content: "b",
				Name:    "name",
			},
		})), convey.ShouldNotBeZeroValue)

		convey.So(len(parseAny(ctx, &schema.Message{
			Role:    schema.Assistant,
			Content: "a",
			Name:    "name",
		})), convey.ShouldNotBeZeroValue)

		convey.So(len(parseAny(ctx, []*schema.Message{
			{
				Role:    schema.Assistant,
				Content: "a",
				Name:    "name",
			},
			{
				Role:    schema.User,
				Content: "b",
				Name:    "name",
			},
			{
				Role:    schema.System,
				Content: "c",
				Name:    "name",
			}, {
				Role:    "",
				Content: "d",
				Name:    "name",
			},
		})), convey.ShouldNotBeZeroValue)

		sb := strings.Builder{}
		sb.WriteString("asd")
		convey.So(len(parseAny(ctx, sb)), convey.ShouldNotBeZeroValue)

		convey.So(len(parseAny(ctx, map[string]any{"asd": 1})), convey.ShouldNotBeZeroValue)

		convey.So(len(parseAny(ctx, "asd")), convey.ShouldNotBeZeroValue)

	})
}
