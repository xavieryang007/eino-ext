package main

import (
	"context"
	"fmt"
	"net/http"

	"code.byted.org/flow/eino-ext/components/document/loader/url"
	"code.byted.org/flow/eino/components/document"
)

func main() {

	staticDir := "../testdata"
	// server
	fileServer := http.FileServer(http.Dir(staticDir))
	http.Handle("/", fileServer)

	addr := "127.0.0.1:18001"

	go func() { // nolint: byted_goroutine_recover
		fmt.Println("Serving directory on http://127.0.0.1:18001")
		if err := http.ListenAndServe(addr, nil); err != nil {
			fmt.Println("Server failed to start:", err)
		}
	}()

	ctx := context.Background()
	loader, err := url.NewLoader(ctx, &url.LoaderConfig{})
	if err != nil {
		panic(err)
	}

	docs, err := loader.Load(ctx, document.Source{
		URI: fmt.Sprintf("http://%s/test.html", addr),
	})
	if err != nil {
		panic(err)
	}

	for _, doc := range docs {
		fmt.Printf("%+v\n", doc)
	}
}
