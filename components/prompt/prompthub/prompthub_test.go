package prompthub

import (
	"reflect"
	"testing"

	prompt2 "code.byted.org/flow/eino/components/prompt"
	"code.byted.org/flow/eino/schema"
	"code.byted.org/flowdevops/fornax_sdk/domain/prompt"
	"code.byted.org/lang/gg/gptr"
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
