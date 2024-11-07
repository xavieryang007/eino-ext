package llmgateway

import (
	"code.byted.org/flow/eino-ext/components/model/llmgateway/internal/utils"
	"code.byted.org/flow/eino/schema"
)

func GetRawResp(message *schema.Message) (string, bool) {
	if message == nil || message.Extra == nil {
		return "", false
	}

	resp, ok := message.Extra[utils.RawResp].(string)
	return resp, ok
}

func SetMessageID(message *schema.Message, messageID string) {
	if message == nil {
		return
	}

	if message.Extra == nil {
		message.Extra = make(map[string]interface{})
	}
	message.Extra[utils.MessageID] = messageID
}

func SetExtra(message *schema.Message, extra map[string]string) {
	if message == nil {
		return
	}

	if message.Extra == nil {
		message.Extra = make(map[string]interface{})
	}
	message.Extra[utils.Extra] = extra
}
