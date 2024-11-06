package main

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	loader "code.byted.org/flow/eino-ext/components/document/loader/url"
	"code.byted.org/flow/eino/components/document"
)

// you can build request by yourself, so you can add custom header、cookie、proxy、timeout etc.

func main() {
	ctx := context.Background()

	urlLoader, err := loader.NewLoader(ctx, &loader.LoaderConfig{
		RequestBuilder: func(ctx context.Context, source document.Source, opts ...document.LoaderOption) (*http.Request, error) {
			u, err := url.Parse(source.URI)
			if err != nil {
				return nil, err
			}

			req := &http.Request{
				Method: "GET",
				URL:    u,
			}
			req.Header.Add("auth-token", "xx-token")
			return req, nil
		},
	})
	if err != nil {
		panic(err)
	}

	docs, err := urlLoader.Load(ctx, document.Source{
		URI: "https://some_private_site.com/some_path/some_file",
	})
	if err != nil {
		panic(err)
	}

	for _, doc := range docs {
		fmt.Printf("%+v\n", doc)
	}
}
