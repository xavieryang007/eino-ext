package prompthub

import (
	"context"
	"reflect"
	"testing"

	"go.uber.org/mock/gomock"

	"code.byted.org/flow/eino/callbacks"
	"code.byted.org/flow/eino/components"
	prompt2 "code.byted.org/flow/eino/components/prompt"
	"code.byted.org/flow/eino/compose"
	"code.byted.org/flow/eino/schema"
	"code.byted.org/flow/flow-telemetry-common/go/obtag"
	"code.byted.org/flowdevops/fornax_sdk/domain/prompt"
	"code.byted.org/lang/gg/gptr"

	"code.byted.org/flow/eino-ext/components/prompt/prompthub/internal/mock"
)

func TestMessageConv(t *testing.T) {
	origMessage := &prompt.Message{
		ID:          0,
		MessageType: prompt.MessageTypeUser,
		Content:     "content",
		ToolCallID:  gptr.Of("tool call id"),
		ToolCalls: []*prompt.ToolCallCombine{
			{ToolCall: &prompt.ToolCall{
				ID:   "tool call id",
				Type: 0,
				FunctionCall: &prompt.FunctionCall{
					Name:      "function call",
					Arguments: gptr.Of("argument"),
				},
			}},
		},
		Parts: []*prompt.ContentPart{
			{
				Type:  prompt.ContentTypeImage,
				Text:  gptr.Of("text"),
				Image: &prompt.Image{URL: "image"},
			},
		},
	}
	targetMessage := &schema.Message{
		Role:    schema.User,
		Content: "content",
		MultiContent: []schema.ChatMessagePart{
			{
				Type: schema.ChatMessagePartTypeImageURL,
				Text: "text",
				ImageURL: &schema.ChatMessageImageURL{
					URL: "image",
				},
			},
		},
		ToolCalls: []schema.ToolCall{
			{
				ID: "tool call id",
				Function: schema.FunctionCall{
					Name:      "function call",
					Arguments: "argument",
				},
			},
		},
		ToolCallID: "tool call id",
	}
	nMessage, err := messageConv(origMessage)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(nMessage, targetMessage) {
		t.Fatal("message conv fail")
	}

	binaryMessage := &prompt.Message{
		Parts: []*prompt.ContentPart{
			{
				Type: prompt.ContentTypeBinary,
			},
		},
	}
	_, err = messageConv(binaryMessage)
	if err == nil {
		t.Fatal("binary message haven't reported error")
	}
}

func TestCallOptions(t *testing.T) {
	opts := []prompt2.Option{
		WithUserID("user id"),
		WithDeviceID("device id"),
		WithKV(map[string]any{
			"key": "value",
		}),
	}

	o := prompt2.GetImplSpecificOptions[options](nil, opts...)
	if *o.UserID != "user id" || *o.DeviceID != "device id" || o.KV["key"] != "value" {
		t.Fatal("test prompt hub call option fail")
	}
}

func TestCallback(t *testing.T) {
	h := callbacks.HandlerBuilder{
		OnStartFn: func(ctx context.Context, info *callbacks.RunInfo, input callbacks.CallbackInput) context.Context {
			if info.Component != components.ComponentOfPrompt || info.Type != "PromptHub" {
				return ctx
			}
			nIn := prompt2.ConvCallbackInput(input)
			if nIn.Variables["key1"] != "value1" {
				t.Fatal("callback input is unexpected")
			}
			if nIn.Extra[obtag.PromptKey] != "test prompt key" {
				t.Fatal("callback input is unexpected")
			}
			if nIn.Extra[obtag.PromptVersion] != "version" {
				t.Fatal("callback input is unexpected")
			}
			return ctx
		},
		OnEndFn: func(ctx context.Context, info *callbacks.RunInfo, output callbacks.CallbackOutput) context.Context {
			if info.Component != components.ComponentOfPrompt || info.Type != "PromptHub" {
				return ctx
			}
			nOut := prompt2.ConvCallbackOutput(output)
			if nOut.Result[0].Content != "value1" {
				t.Fatal("callback output is unexpected")
			}
			if nOut.Templates[0].(*schema.Message).Content != "{{key1}}" {
				t.Fatal("callback output is unexpected")
			}
			if nOut.Extra[obtag.PromptKey] != "test prompt key" {
				t.Fatal("callback output is unexpected")
			}
			if nOut.Extra[obtag.PromptVersion] != "version" {
				t.Fatal("callback output is unexpected")
			}
			return ctx
		},
	}
	ctx := context.Background()
	g := compose.NewGraph[map[string]any, []*schema.Message]()

	version := "version"
	ctrl := gomock.NewController(t)
	mockFornaxClient := mock.NewMockIClient(ctrl)
	mockFornaxClient.EXPECT().GetPrompt(gomock.Any(), gomock.Any()).Return(&prompt.GetPromptResult{Prompt: &prompt.Prompt{
		PromptText: &prompt.PromptText{
			SystemPrompt: &prompt.Message{
				MessageType: prompt.MessageTypeSystem,
				Content:     "{{key1}}",
			},
			UserPrompt: nil,
		},
	}}, nil).Times(1)

	tpl, err := NewPromptHub(ctx, &Config{
		Key:          "test prompt key",
		Version:      &version,
		FornaxClient: mockFornaxClient,
	})
	if err != nil {
		t.Fatal(err)
	}
	err = g.AddChatTemplateNode("1", tpl)
	if err != nil {
		t.Fatal(err)
	}

	err = g.AddEdge(compose.START, "1")
	if err != nil {
		t.Fatal(err)
	}
	err = g.AddEdge("1", compose.END)
	if err != nil {
		t.Fatal(err)
	}

	r, err := g.Compile(ctx)
	if err != nil {
		t.Fatal(err)
	}

	_, err = r.Invoke(ctx, map[string]interface{}{"key1": "value1"}, compose.WithCallbacks(&h))
	if err != nil {
		t.Fatal(err)
	}
}
