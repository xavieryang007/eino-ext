package fornax

import (
	"context"
	"fmt"
	"testing"
	"time"

	. "github.com/bytedance/mockey"
	"github.com/smartystreets/goconvey/convey"
	"github.com/stretchr/testify/assert"

	"code.byted.org/flow/eino/callbacks"
	"code.byted.org/flow/eino/components"
	"code.byted.org/flow/eino/components/model"
	"code.byted.org/flow/eino/schema"
	"code.byted.org/flowdevops/fornax_sdk"
	"code.byted.org/flowdevops/fornax_sdk/infra/ob"
)

func TestNewTraceCallbackHandler(t *testing.T) {
	tracer := newTraceCallbackHandler(&fornax_sdk.Client{},
		&options{
			injectTraceSwitch: func(ctx context.Context, runInfo *callbacks.RunInfo) (needInject bool) {
				return true
			},
		})
	assert.NotNil(t, tracer)
}

func TestTraceIntegration(t *testing.T) {
	PatchConvey("test integration", t, func() {
		mockCli := &fornax_sdk.Client{}
		mockSpan := &ob.FornaxSpanImpl{}

		tracer := &einoTracer{
			client: mockCli,
			parser: &defaultDataParser{},
			needInject: func(ctx context.Context, runInfo *callbacks.RunInfo) (needInject bool) {
				return true
			},
		}

		Mock(GetMethod(mockCli, "StartSpan")).To(func(ctx context.Context, name, spanType string, opts ...ob.FornaxStartSpanOption) (ob.FornaxSpan, context.Context, error) {
			return mockSpan, ctx, nil
		}).Build()
		Mock(GetMethod(mockCli, "GetSpanFromContext")).Return(mockSpan).Build()
		Mock(GetMethod(mockSpan, "SetTag")).Return().Build()
		Mock(fornax_sdk.InjectTrace).To(func(ctx context.Context) (context.Context, error) {
			return ctx, fmt.Errorf("skip err")
		}).Build()

		info := &callbacks.RunInfo{
			Name:      "mock_name",
			Type:      "OpenAI",
			Component: components.ComponentOfChatModel,
		}

		PatchConvey("test invoke success", func() {
			ci := &model.CallbackInput{
				Messages: []*schema.Message{
					{
						Role:    schema.User,
						Content: "hello",
					},
				},
				Config: &model.Config{
					Model:       "mock_model_name",
					MaxTokens:   123,
					Temperature: 1,
					TopP:        0.7,
				},
			}

			ctx := tracer.OnStart(context.Background(), info, ci)
			span := tracer.client.GetSpanFromContext(ctx)
			convey.So(span, convey.ShouldNotBeNil)

			_, ok := span.(*ob.FornaxSpanImpl)
			convey.So(ok, convey.ShouldBeTrue)

			Mock(GetMethod(span, "Finish")).Return().Build()

			co := &model.CallbackOutput{
				Message: &schema.Message{
					Role:    schema.Assistant,
					Content: "hello",
				},

				Config: &model.Config{
					Model:       "mock_model_name",
					MaxTokens:   123,
					Temperature: 1,
					TopP:        0.7,
				},
				TokenUsage: &model.TokenUsage{
					PromptTokens:     1,
					CompletionTokens: 2,
					TotalTokens:      3,
				},
			}

			ctx = tracer.OnEnd(ctx, info, co)
		})

		PatchConvey("test invoke error", func() {
			ci := &model.CallbackInput{
				Messages: []*schema.Message{
					{
						Role:    schema.User,
						Content: "hello",
					},
				},
				Config: &model.Config{
					Model:       "mock_model_name",
					MaxTokens:   123,
					Temperature: 1,
					TopP:        0.7,
				},
			}

			ctx := tracer.OnStart(context.Background(), info, ci)
			span := tracer.client.GetSpanFromContext(ctx)
			convey.So(span, convey.ShouldNotBeNil)

			_, ok := span.(*ob.FornaxSpanImpl)
			convey.So(ok, convey.ShouldBeTrue)

			Mock(GetMethod(span, "Finish")).Return().Build()

			ctx = tracer.OnError(ctx, info, fmt.Errorf("mock err"))
		})

		PatchConvey("test stream success", func() {
			cfg := &model.Config{
				Model:       "mock_model_name",
				MaxTokens:   123,
				Temperature: 1,
				TopP:        0.7,
			}

			ci := &model.CallbackInput{
				Messages: []*schema.Message{
					{
						Role:    schema.User,
						Content: "hello",
					},
				},
				Config: cfg,
			}

			ctx := tracer.OnStart(context.Background(), info, ci)
			span := tracer.client.GetSpanFromContext(ctx)
			convey.So(span, convey.ShouldNotBeNil)

			_, ok := span.(*ob.FornaxSpanImpl)
			convey.So(ok, convey.ShouldBeTrue)

			Mock(GetMethod(span, "Finish")).Return().Build()

			sr, sw := schema.Pipe[callbacks.CallbackOutput](1)
			go func() {
				defer sw.Close()

				for i := 0; i < 10; i++ {
					closed := sw.Send(&model.CallbackOutput{
						Message: &schema.Message{Role: schema.Assistant, Content: fmt.Sprint(i)},
						Config:  cfg,
						TokenUsage: &model.TokenUsage{
							PromptTokens:     1,
							CompletionTokens: 2,
							TotalTokens:      3,
						},
					}, nil)

					if closed {
						break
					}
				}
			}()

			ctx = tracer.OnEndWithStreamOutput(ctx, info, sr)

			time.Sleep(time.Second * 1)
		})

		PatchConvey("test stream recv error", func() {
			cfg := &model.Config{
				Model:       "mock_model_name",
				MaxTokens:   123,
				Temperature: 1,
				TopP:        0.7,
			}

			ci := &model.CallbackInput{
				Messages: []*schema.Message{
					{
						Role:    schema.User,
						Content: "hello",
					},
				},
				Config: cfg,
			}

			ctx := tracer.OnStart(context.Background(), info, ci)
			span := tracer.client.GetSpanFromContext(ctx)
			convey.So(span, convey.ShouldNotBeNil)

			_, ok := span.(*ob.FornaxSpanImpl)
			convey.So(ok, convey.ShouldBeTrue)

			Mock(GetMethod(span, "Finish")).Return().Build()

			sr, sw := schema.Pipe[callbacks.CallbackOutput](1)

			go func() {
				_ = sw.Send(nil, fmt.Errorf("mock err"))
				sw.Close()
			}()

			ctx = tracer.OnEndWithStreamOutput(ctx, info, sr)

			time.Sleep(time.Second * 1)
		})

		PatchConvey("test collect success", func() {
			sr, sw := schema.Pipe[callbacks.CallbackInput](1)
			go func() {
				defer sw.Close()

				str := "hello"
				for i := 0; i < len(str); i++ {
					if closed := sw.Send(&schema.Message{Role: schema.User, Content: str[i : i+1]}, nil); closed {
						break
					}
				}
			}()

			ctx := tracer.OnStartWithStreamInput(context.Background(), info, sr)
			convey.So(ctx.Value(traceStreamInputAsyncKey{}), convey.ShouldNotBeNil)

			span := tracer.client.GetSpanFromContext(ctx)
			convey.So(span, convey.ShouldNotBeNil)

			_, ok := span.(*ob.FornaxSpanImpl)
			convey.So(ok, convey.ShouldBeTrue)

			Mock(GetMethod(span, "Finish")).Return().Build()

			ctx = tracer.OnEnd(ctx, info, &model.CallbackOutput{
				Message: &schema.Message{
					Role:    schema.Assistant,
					Content: "hello",
				},

				Config: &model.Config{
					Model:       "mock_model_name",
					MaxTokens:   123,
					Temperature: 1,
					TopP:        0.7,
				},
				TokenUsage: &model.TokenUsage{
					PromptTokens:     1,
					CompletionTokens: 2,
					TotalTokens:      3,
				},
			})

			ch, ok := ctx.Value(traceStreamInputAsyncKey{}).(streamInputAsyncVal)
			convey.So(ok, convey.ShouldBeTrue)

			_, ok = <-ch
			convey.So(ok, convey.ShouldBeFalse)
		})
	})
}
