package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"code.byted.org/flow/eino/components/document"
	"code.byted.org/overpass/larkai_workflow_facade/kitex_gen/larkai/workflow/facade/entity"

	"code.byted.org/flow/eino-ext/components/document/loadersplitter/idp"
)

func main() {
	ctx := context.Background()

	larkDocSplitter := idp.NewIDPLoaderSplitter(&idp.Config{
		AppID:     5, // boe test account, if you have your own account, please replace it
		SecretKey: "d2c10e508cef11ee964eacde48001122",

		Strategy: entity.ChunkingStrategy_SEMANTIC,
		FileType: idp.FileTypeOfLarkDoc,

		LarkDocConfig: &idp.LarkDocConfig{
			AppID:     os.Getenv("LARK_APP_ID"),     // <= export env of your lark app id, you can get from https://open.feishu.cn/app/
			AppSecret: os.Getenv("LARK_APP_SECRET"), // <= export env of your lark app secret
		},

		Timeout: time.Second * 600,
	})

	docs, err := larkDocSplitter.LoadAndSplit(ctx, document.Source{
		URI: os.Getenv("LARK_DOC_URL"), // <= export env of your lark doc uri, may be like https://bytedance.feishu.cn/docx/xxxxxxx
	}, idp.WithLarkDocAccessKey(os.Getenv("LARK_ACCESS_KEY"))) // <= export env of your lark access key, you can generate one for debug from https://open.feishu.cn/api-explorer/

	if err != nil {
		panic(err)
	}

	for _, doc := range docs {
		fmt.Println(doc.Content)
	}

	fmt.Println("finish")
}
