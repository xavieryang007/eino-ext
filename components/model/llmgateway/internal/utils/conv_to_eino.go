package utils

import (
	"code.byted.org/flow/eino/components/model"
	"code.byted.org/flow/eino/schema"
	"code.byted.org/gopkg/lang/v2/slicex"
	"code.byted.org/overpass/stone_llm_gateway/kitex_gen/stone/llm/gateway"
)

func ToEinoMessage(resp *gateway.ChatCompletion) (*schema.Message, bool) {
	if resp == nil {
		return nil, false
	}

	for _, choice := range resp.Choices {
		if choice.Index != 0 {
			continue
		}

		if choice.Delta == nil {
			res := &schema.Message{
				ResponseMeta: &schema.ResponseMeta{
					FinishReason: choice.FinishReason,
					Usage:        ToEinoUsage(resp.Usage),
				},
			}
			return res, true
		}

		msg := choice.Delta

		ext := make(map[string]any)
		if rawData := resp.RawData; rawData != nil {
			ext[RawResp] = *rawData
		}
		if len(msg.Extra) > 0 {
			for k, v := range msg.Extra {
				ext[k] = v
			}
		}

		res := &schema.Message{
			Role:       toEinoRole(msg.Role),
			Content:    msg.Content,
			Name:       EmptyIfNil(msg.Name),
			ToolCalls:  toEinoToolCalls(msg.ToolCalls, ext),
			ToolCallID: EmptyIfNil(msg.ToolCallId),
			ResponseMeta: &schema.ResponseMeta{
				FinishReason: choice.FinishReason,
				Usage:        ToEinoUsage(resp.Usage),
			},
			Extra: ext,
		}

		return res, true
	}

	if resp.Usage != nil {
		ext := make(map[string]any)
		if rawData := resp.RawData; rawData != nil {
			ext[RawResp] = *rawData
		}

		finishReason := ""
		for _, choice := range resp.Choices {
			if choice.FinishReason != "" {
				finishReason = choice.FinishReason
				break
			}
		}

		res := &schema.Message{

			ResponseMeta: &schema.ResponseMeta{
				FinishReason: finishReason,
				Usage:        ToEinoUsage(resp.Usage),
			},
			Extra: ext,
		}

		return res, true
	}

	return nil, false
}

func ToEinoUsage(usage *gateway.Usage) *schema.TokenUsage {
	if usage == nil {
		return nil
	}

	return &schema.TokenUsage{
		CompletionTokens: int(usage.CompletionTokens),
		PromptTokens:     int(usage.PromptTokens),
		TotalTokens:      int(usage.TotalTokens),
	}
}

func ToModelCallbackUsage(respMeta *schema.ResponseMeta) *model.TokenUsage {
	if respMeta == nil {
		return nil
	}
	usage := respMeta.Usage
	if usage == nil {
		return nil
	}
	return &model.TokenUsage{
		CompletionTokens: usage.CompletionTokens,
		PromptTokens:     usage.PromptTokens,
		TotalTokens:      usage.TotalTokens,
	}
}

func toEinoRole(role string) schema.RoleType {
	switch role {
	case RoleUser:
		return schema.User
	case RoleAssistant:
		return schema.Assistant
	case RoleSystem:
		return schema.System
	case RoleTool:
		return schema.Tool
	default:
		return schema.RoleType(role)
	}
}

func toEinoToolCalls(toolCalls []*gateway.ToolCall, ext map[string]any) []schema.ToolCall {
	if len(toolCalls) == 0 {
		return nil
	}

	return slicex.Map(toolCalls, func(t *gateway.ToolCall) schema.ToolCall {
		return toEinoToolCall(t, ext)
	})
}

func toEinoToolCall(toolCall *gateway.ToolCall, ext map[string]any) schema.ToolCall {
	if toolCall == nil {
		return schema.ToolCall{}
	}

	res := schema.ToolCall{
		Function: schema.FunctionCall{},
		Index:    IntPtr(&toolCall.Index),
		ID:       toolCall.Id,
		Type:     toolCall.Type,
	}

	if toolCall.Function_ != nil {
		res.Function.Arguments = toolCall.Function_.Arguments
		res.Function.Name = toolCall.Function_.Name
	}

	if v, ok := toolCall.Extra[Thought]; ok {
		res.Extra = make(map[string]any)
		res.Extra[Thought] = v
	}
	return res
}
