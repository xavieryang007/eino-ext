package llmgateway

import (
	"code.byted.org/flow/eino/schema"

	"code.byted.org/flow/eino-ext/components/model/llmgateway/internal/utils"
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

func copyMap[M1 ~map[K]V, M2 ~map[K]V, K comparable, V any](dst M1, src M2) {
	for k, v := range src {
		dst[k] = v
	}
}
