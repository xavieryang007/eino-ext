package main

import (
	"context"
	"fmt"
	"net/http"

	"code.byted.org/flow/eino-ext/components/document/loader/url"
	"code.byted.org/flow/eino/components/document"
)

// url loader can load file from dir path, because you can implement the Client

func main() {
	staticDir := "../testdata"
	ctx := context.Background()
	loader, err := url.NewLoader(ctx, &url.LoaderConfig{
		Client: &http.Client{
			Transport: http.NewFileTransport(http.Dir(staticDir)),
		},
	})
	if err != nil {
		panic(err)
	}

	docs, err := loader.Load(ctx, document.Source{
		URI: "file:///test.html",
	})
	if err != nil {
		panic(err)
	}

	for _, doc := range docs {
		fmt.Printf("%+v\n", doc)
	}
}
