package main

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"

	loader "code.byted.org/flow/eino-ext/components/document/loader/url"
	"code.byted.org/flow/eino/components/document"
)

// you can use any proxy in loader, because you can set your own client.

func main() {
	proxyURL := "http://127.0.0.1:1080"
	u, err := url.Parse(proxyURL)
	if err != nil {
		panic(err)
	}

	ctx := context.Background()
	urlLoader, err := loader.NewLoader(ctx, &loader.LoaderConfig{
		Client: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				Proxy: http.ProxyURL(u),
			},
		},
	})
	if err != nil {
		panic(err)
	}

	docs, err := urlLoader.Load(ctx, document.Source{
		URI: "https://some_private_site.com",
	})
	if err != nil {
		panic(err)
	}
	for _, doc := range docs {
		fmt.Printf("%+v\n", doc)
	}
}
