package prompthub

import (
	"context"
	"fmt"
	"reflect"

	"code.byted.org/flow/eino/components/prompt"
	"code.byted.org/flow/eino/schema"
	"code.byted.org/flowdevops/fornax_sdk"
	prompt2 "code.byted.org/flowdevops/fornax_sdk/domain/prompt"
)

type Config struct {
	Key          string
	Version      *string
	FornaxClient *fornax_sdk.Client
}

func NewPromptHub(_ context.Context, conf *Config) (prompt.ChatTemplate, error) {
	if conf == nil {
		return nil, fmt.Errorf("new prompt hub fail because conf is empty")
	}
	if conf.FornaxClient == nil {
		return nil, fmt.Errorf("new prompt hub fail because fornax client in conf is empty")
	}
	return &promptHub{cli: conf.FornaxClient, key: conf.Key, version: conf.Version}, nil
}

type promptHub struct {
	cli     *fornax_sdk.Client
	key     string
	version *string
}

func (p *promptHub) Format(ctx context.Context, vs map[string]any, opts ...prompt.Option) ([]*schema.Message, error) {
	o := prompt.GetImplSpecificOptions[options](nil, opts...)
	var getPromptOptions []prompt2.Option
	if o.UserID != nil || o.DeviceID != nil || len(o.KV) != 0 {
		grayContext := prompt2.NewGrayContext()
		if o.UserID != nil {
			grayContext.SetUserID(*o.UserID)
		}
		if o.DeviceID != nil {
			grayContext.SetDeviceID(*o.DeviceID)
		}
		for k, v := range o.KV {
			grayContext.SetKV(k, v)
		}
		getPromptOptions = append(getPromptOptions, prompt2.WithGrayContext(grayContext))
	}

	template, err := p.cli.GetPrompt(ctx, &prompt2.GetPromptParam{
		Key:     p.key,
		Version: p.version,
	}, getPromptOptions...)
	if err != nil {
		return nil, fmt.Errorf("get prompt from prompt service fail: %w", err)
	}
	messages := make([]schema.MessagesTemplate, 0)
	// prompt may be not nil but a zero value if prompt doesn't exist.
	if template != nil && template.GetPrompt().GetPromptText().GetSystemPrompt() != nil && !reflect.ValueOf(template.GetPrompt().GetPromptText().GetSystemPrompt()).Elem().IsZero() {
		m, err := messageConv(template.GetPrompt().GetPromptText().GetSystemPrompt())
		if err != nil {
			return nil, err
		}
		messages = append(messages, m)
	}
	// prompt may be not nil but a zero value if prompt doesn't exist.
	if template != nil && template.GetPrompt().GetPromptText().GetUserPrompt() != nil && !reflect.ValueOf(template.GetPrompt().GetPromptText().GetUserPrompt()).Elem().IsZero() {
		m, err := messageConv(template.GetPrompt().GetPromptText().GetUserPrompt())
		if err != nil {
			return nil, err
		}
		messages = append(messages, m)
	}
	tpl := prompt.FromMessages(schema.Jinja2, messages...)
	return tpl.Format(ctx, vs, opts...)
}

func messageConv(orig *prompt2.Message) (*schema.Message, error) {
	if orig == nil {
		return nil, nil
	}
	var err error
	ret := &schema.Message{
		Content: orig.Content,
	}
	ret.Role, err = messageTypeConv(orig.MessageType)
	if err != nil {
		return nil, err
	}
	if orig.ToolCallID != nil {
		ret.ToolCallID = *orig.ToolCallID
	}
	ret.ToolCalls = toolCallsConv(orig.ToolCalls)
	ret.MultiContent, err = partsConv(orig.Parts)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func messageTypeConv(t prompt2.MessageType) (schema.RoleType, error) {
	switch t {
	case prompt2.MessageTypeSystem:
		return schema.System, nil
	case prompt2.MessageTypeUser:
		return schema.User, nil
	case prompt2.MessageTypeAssistant:
		return schema.Assistant, nil
	case prompt2.MessageTypeTool:
		return schema.Tool, nil
	default:
		return "", fmt.Errorf("unknown message type from fornax prompthub: %v", t)
	}
}

func toolCallsConv(tools []*prompt2.ToolCallCombine) []schema.ToolCall {
	ret := make([]schema.ToolCall, 0, len(tools))
	for _, tool := range tools {
		if tool == nil || tool.ToolCall == nil {
			continue
		}
		tc := schema.ToolCall{
			ID: tool.ToolCall.ID,
		}
		if tool.ToolCall.FunctionCall != nil {
			tc.Function.Name = tool.ToolCall.FunctionCall.Name
			if tool.ToolCall.FunctionCall.Arguments != nil {
				tc.Function.Arguments = *tool.ToolCall.FunctionCall.Arguments
			}
		}
		ret = append(ret, tc)
	}
	return ret
}

func partsConv(parts []*prompt2.ContentPart) ([]schema.ChatMessagePart, error) {
	var err error
	ret := make([]schema.ChatMessagePart, 0, len(parts))
	for _, part := range parts {
		if part == nil {
			continue
		}
		cmp := schema.ChatMessagePart{}
		cmp.Type, err = chatMessagePartTypeConv(part.Type)
		if err != nil {
			return nil, err
		}
		if part.Image != nil {
			cmp.ImageURL = &schema.ChatMessageImageURL{
				URL: part.Image.URL,
			}
		}
		if part.Text != nil {
			cmp.Text = *part.Text
		}
		ret = append(ret, cmp)
	}
	return ret, nil
}

func chatMessagePartTypeConv(t prompt2.ContentType) (schema.ChatMessagePartType, error) {
	switch t {
	case prompt2.ContentTypeText:
		return schema.ChatMessagePartTypeText, nil
	case prompt2.ContentTypeImage:
		return schema.ChatMessagePartTypeImageURL, nil
	case prompt2.ContentTypeBinary:
		return "", fmt.Errorf("chat message part type[binary] isn't supported")
	default:
		return "", fmt.Errorf("unknown chat message part type from fornax prompthub: %v", t)
	}
}
