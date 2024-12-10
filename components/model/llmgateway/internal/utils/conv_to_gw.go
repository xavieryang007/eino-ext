package utils

import (
	"fmt"

	"github.com/getkin/kin-openapi/openapi3"

	"code.byted.org/flow/eino/schema"
	"code.byted.org/gopkg/lang/conv"
	"code.byted.org/overpass/stone_llm_gateway/kitex_gen/stone/llm/gateway"
)

func ToGWMessages(msgs []*schema.Message) ([]*gateway.Message, error) {
	res := make([]*gateway.Message, 0, len(msgs))
	for _, m := range msgs {
		msg, err := toGWMessage(m)
		if err != nil {
			return nil, err
		}
		res = append(res, msg)
	}
	return res, nil
}

func ToGWTools(tools []*schema.ToolInfo) ([]*gateway.Tool, error) {
	res := make([]*gateway.Tool, 0, len(tools))

	for _, t := range tools {
		gwTool, err := toGWTool(t)
		if err != nil {
			return nil, err
		}
		res = append(res, gwTool)
	}

	return res, nil
}

func toGWMessage(msg *schema.Message) (*gateway.Message, error) {
	if msg == nil {
		return nil, nil
	}

	mc, e := toGWMultiContent(msg.MultiContent)
	if e != nil {
		return nil, e
	}

	message := &gateway.Message{
		Name:       toStringPtr(msg.Name),
		Role:       toGWRole(msg.Role),
		Content:    msg.Content,
		Contents:   mc,
		ToolCalls:  toGWToolCalls(msg.ToolCalls),
		ToolCallId: toStringPtr(msg.ToolCallID),
	}

	if msg.Extra == nil {
		msg.Extra = make(map[string]any)
	}
	if mID, ok := msg.Extra[MessageID].(string); ok {
		message.MessageId = &mID
	}
	if extra, ok := msg.Extra[Extra].(map[string]string); ok {
		message.Extra = extra
	}

	return message, nil
}

func toGWMultiContent(mc []schema.ChatMessagePart) ([]*gateway.MessageMultiPart, error) {
	if len(mc) == 0 {
		return nil, nil
	}

	ret := make([]*gateway.MessageMultiPart, 0, len(mc))
	for _, m := range mc {
		part := m
		switch part.Type {
		case schema.ChatMessagePartTypeText:
			ret = append(ret, &gateway.MessageMultiPart{
				Type: gateway.ChatMessagePartType_Text,
				Text: conv.StringPtr(part.Text),
			})
		case schema.ChatMessagePartTypeImageURL:
			if part.ImageURL == nil {
				return nil, fmt.Errorf("image_url should not be nil")
			}
			detail := string(part.ImageURL.Detail)
			ret = append(ret, &gateway.MessageMultiPart{
				Type: gateway.ChatMessagePartType_ImageURL,
				ImageUrl: &gateway.MessageMultiPartImageURL{
					MimeType: part.ImageURL.MIMEType,
					Url:      part.ImageURL.URL,
					Uri:      &part.ImageURL.URI,
					Detail:   &detail,
				},
			})
		case schema.ChatMessagePartTypeFileURL:
			if part.FileURL == nil {
				return nil, fmt.Errorf("file_url should not be nil")
			}
			ret = append(ret, &gateway.MessageMultiPart{
				Type: gateway.ChatMessagePartType_FileURL,
				FileUrl: &gateway.MessageMultiPartFileURL{
					MimeType: part.FileURL.MIMEType,
					Url:      part.FileURL.URL,
					Uri:      &part.FileURL.URI,
					Name:     &part.FileURL.Name,
				},
			})
		default:
			return nil, fmt.Errorf("unsupported chat message part type: %s", part.Type)
		}
	}

	return ret, nil
}

func toGWRole(role schema.RoleType) string {
	switch role {
	case schema.User:
		return RoleUser
	case schema.Assistant:
		return RoleAssistant
	case schema.System:
		return RoleSystem
	case schema.Tool:
		return RoleTool
	default:
		return string(role)
	}
}

func toGWToolCalls(toolCalls []schema.ToolCall) []*gateway.ToolCall {
	res := make([]*gateway.ToolCall, 0, len(toolCalls))

	for _, t := range toolCalls {
		res = append(res, toGWToolCall(t))
	}

	return res
}

func toGWToolCall(toolCall schema.ToolCall) *gateway.ToolCall {
	res := gateway.NewToolCall()
	res.Id = toolCall.ID
	res.Type = toolCall.Type
	res.Function_ = &gateway.FunctionCall{
		Arguments: toolCall.Function.Arguments,
		Name:      toolCall.Function.Name,
	}

	if toolCall.Index != nil {
		res.Index = int64(*toolCall.Index)
	}
	if v, ok := toolCall.Extra[Thought].(string); ok {
		res.Extra = make(map[string]string)
		res.Extra[Thought] = v
	}
	return res
}

func toGWTool(tool *schema.ToolInfo) (*gateway.Tool, error) {
	if tool == nil {
		return nil, nil
	}

	res := gateway.NewTool()
	res.Type = "function"
	res.Function_ = gateway.NewFunction()
	res.Function_.Name = tool.Name
	res.Function_.Description = toStringPtr(tool.Desc)

	stdSchema, err := tool.ToOpenAPIV3()
	if err != nil {
		return nil, fmt.Errorf("get openapi3 schema failed: %w", err)
	}

	res.Function_.Parameters = convertOpenAPI3SchemaToGWSchema(stdSchema)
	return res, nil
}

// convertOpenAPI3SchemaToGWSchema convert *openapi3.Schema to *gateway.JSONSchema
func convertOpenAPI3SchemaToGWSchema(sc *openapi3.Schema) *gateway.JSONSchema {
	if sc == nil {
		return &gateway.JSONSchema{
			Type:       openapi3.TypeObject,
			Properties: make(map[string]*gateway.JSONSchema),
		}
	}

	// init JSONSchema
	gwSchema := &gateway.JSONSchema{
		Type:        sc.Type,
		Properties:  make(map[string]*gateway.JSONSchema),
		Required:    sc.Required,
		Description: toStringPtr(sc.Description),
		Enum:        nil,
	}

	// convert enums
	if len(sc.Enum) > 0 {
		strEnums := make([]string, 0, len(sc.Enum))
		for _, enumValue := range sc.Enum {
			// 假设枚举值是字符串类型（根据 OpenAPI 规范的常见用法）
			if strValue, ok := enumValue.(string); ok {
				strEnums = append(strEnums, strValue)
			}
		}
		gwSchema.Enum = strEnums
	}

	// deal Properties
	if sc.Properties != nil {
		for propName, propSchemaRef := range sc.Properties {
			gwSchema.Properties[propName] = convertOpenAPI3SchemaToGWSchema(propSchemaRef.Value)
		}
	}

	// deal Items
	if sc.Items != nil {
		gwSchema.Items = convertOpenAPI3SchemaToGWSchema(sc.Items.Value)
	}

	return gwSchema
}
