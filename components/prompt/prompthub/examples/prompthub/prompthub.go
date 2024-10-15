package main

import (
	"context"
	"os"

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
	chatTpl, err := prompthub.NewPromptHub(ctx, &prompthub.Config{
		Key:          "flow.eino.test",
		Version:      nil, // 默认是发布版本的 latest
		FornaxClient: fornaxClient,
	})
	if err != nil {
		logs.Errorf("prompthub.NewPromptHub failed, err=%w", err)
		return
	}

	// 调用 PromptHub，进行请求的渲染
	msgs, err := chatTpl.Format(ctx, map[string]any{
		"role":      "百科全书",
		"name":      "Alice",
		"tool_name": "google_search",
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
