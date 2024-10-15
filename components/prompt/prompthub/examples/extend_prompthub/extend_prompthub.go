package main

import (
	"context"
	"os"

	"code.byted.org/flow/eino/components/prompt"
	"code.byted.org/flow/eino/schema"
	"code.byted.org/flowdevops/fornax_sdk"
	"code.byted.org/flowdevops/fornax_sdk/domain"
	"code.byted.org/gopkg/logs/v2"

	"code.byted.org/flow/eino-ext/components/prompt/prompthub"
)

// 本地执行时，需配置如下环境变量，以构建正确的请求环境
// CONSUL_HTTP_HOST=x.x.x.x
// RUNTIME_IDC_NAME=boe
func main() {
	ctx := context.Background()
	ak := os.Getenv("FORNAX_AK")
	sk := os.Getenv("FORNAX_SK")

	// 可以从 Fornax 样板间获取对应的 AK、SK
	// https://fornax.bytedance.net/space/7369587359924355073/manage/space
	identity := &domain.Identity{
		AK: ak,
		SK: sk,
	}

	config := &domain.Config{
		Identity: identity,
	}

	fornaxClient, err := fornax_sdk.NewClient(config)
	if err != nil {
		logs.Errorf("fornax_sdk.NewClient failed, err=%w", err)
		return
	}

	// 创建指定 PromptKey 的 PromptHub
	// https://fornax.bytedance.net/space/7369587359924355073/prompt/develop/7425840012932939777
	chatTpl, err := newExtendPromptHub(ctx, &prompthub.Config{
		Key:          "flow.eino.test",
		Version:      nil, // 默认是发布版本的 latest
		FornaxClient: fornaxClient,
	})
	if err != nil {
		logs.Errorf("newExtendPromptHub failed, err=%w", err)
		return
	}

	// 调用 PromptHub，进行请求的渲染
	msgs, err := chatTpl.Format(ctx, map[string]any{
		"role":            "百科全书",
		"name":            "Alice",
		"tool_name":       "google_search",
		"message_history": []*schema.Message{},
		"user_query":      []*schema.Message{schema.UserMessage("地球有多种")},
	})
	if err != nil {
		logs.Errorf("chatTpl.Format failed, err=%w", err)
		return
	}

	logs.Infof("Formated Message Length: %v", len(msgs))
	for idx, msg := range msgs {
		logs.Infof("Formated Message %d: %v", idx+1, msg)
	}
}

func newExtendPromptHub(ctx context.Context, conf *prompthub.Config) (prompt.ChatTemplate, error) {
	chatTpl, err := prompthub.NewPromptHub(ctx, &prompthub.Config{
		Key:          conf.Key,
		Version:      conf.Version,
		FornaxClient: conf.FornaxClient,
	})
	if err != nil {
		return nil, err
	}

	return &extendPromptHub{
		ph: chatTpl,
	}, nil
}

type extendPromptHub struct {
	ph prompt.ChatTemplate
}

func (e *extendPromptHub) Format(ctx context.Context, vs map[string]any, opts ...prompt.Option) ([]*schema.Message, error) {
	spMsgs, err := e.ph.Format(ctx, vs, opts...)
	if err != nil {
		return nil, err
	}

	const (
		placeholderOfSystemTemplate = "system_template"
		placeholderOfMessageHistory = "message_history"
		placeholderOfUserQuery      = "user_query"
	)

	tpl := prompt.FromMessages(schema.Jinja2,
		schema.MessagesPlaceholder(placeholderOfSystemTemplate, false),
		schema.MessagesPlaceholder(placeholderOfMessageHistory, false),
		schema.MessagesPlaceholder(placeholderOfUserQuery, false),
	)

	return tpl.Format(ctx, map[string]any{
		placeholderOfSystemTemplate: spMsgs,
		placeholderOfMessageHistory: vs[placeholderOfMessageHistory],
		placeholderOfUserQuery:      vs[placeholderOfUserQuery],
	}, opts...)
}

func (e *extendPromptHub) GetType() string {
	return "ExtendPromptHub"
}

func (e *extendPromptHub) IsCallbacksEnabled() bool {
	return true
}
