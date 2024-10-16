package fornax

import (
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/stretchr/testify/assert"

	"code.byted.org/flow/eino/schema"
)

func Test_convertModelMessage(t *testing.T) {
	msg := &schema.Message{
		Role:    schema.Assistant,
		Content: "",
		MultiContent: []schema.ChatMessagePart{
			{
				Type:     schema.ChatMessagePartTypeText,
				Text:     "asd",
				ImageURL: nil,
			},
			{
				Type: schema.ChatMessagePartTypeImageURL,
				Text: "",
				ImageURL: &schema.ChatMessageImageURL{
					URL:    "mock_url",
					Detail: "mock_detail",
				},
			},
		},
		Name: "",
		ToolCalls: []schema.ToolCall{
			{
				Index: nil,
				ID:    "asd",
				Function: schema.FunctionCall{
					Name:      "mock_name",
					Arguments: "mock_args",
				},
			},
		},
		ToolCallID: "qwe",
		Extra:      nil,
	}

	obMsg := convertModelMessage(msg)
	assert.NotNil(t, obMsg)
}

func Test_convertTool(t *testing.T) {
	tool := &schema.ToolInfo{
		Name: "name",
		Desc: "desc",
		ParamsOneOf: schema.ParamsOneOf{
			OpenAPIV3: &openapi3.Schema{
				Type:        "array",
				UniqueItems: true,
				Items: (&openapi3.Schema{
					Type: "object",
					Properties: openapi3.Schemas{
						"key1": openapi3.NewFloat64Schema().NewRef(),
					},
				}).NewRef(),
			},
		},
	}

	obTool := convertTool(tool)
	assert.NotNil(t, obTool)
}

func Test_convertToolChoice(t *testing.T) {
	assert.NotNil(t, convertToolChoice("asd"))
	assert.NotNil(t, convertToolChoice(&schema.ToolInfo{Name: "mock_tool"}))
	assert.Nil(t, convertToolChoice(123))
}
