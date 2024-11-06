package url

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"code.byted.org/flow/eino-ext/components/document/parser/html"
	"code.byted.org/flow/eino/callbacks"
	"code.byted.org/flow/eino/components/document"
	"code.byted.org/flow/eino/components/document/parser"
	"code.byted.org/flow/eino/schema"
)

type MockParser struct {
	mock func(io.Reader) ([]*schema.Document, error)
}

func (p *MockParser) Parse(ctx context.Context, reader io.Reader, opts ...parser.Option) (doc []*schema.Document, err error) {
	return p.mock(reader)
}

func TestLoad(t *testing.T) {
	staticDir := "./testdata"
	fileServer := http.FileServer(http.Dir(staticDir))
	addr := "127.0.0.1:18001"

	go func() {
		if err := http.ListenAndServe(addr, fileServer); err != nil {
			fmt.Println("Server failed to start:", err)
		}
	}()

	time.Sleep(1 * time.Second)
	ctx := context.Background()
	ctx = callbacks.CtxWithManager(ctx, &callbacks.Manager{})

	t.Run("html loader", func(t *testing.T) {
		loader, err := NewLoader(ctx, &LoaderConfig{})
		assert.Nil(t, err)

		url := fmt.Sprintf("http://%s/test.html", addr)
		docs, err := loader.Load(ctx, document.Source{URI: url})
		assert.Nil(t, err)

		assert.Equal(t, 1, len(docs))
		assert.Equal(t, url, docs[0].MetaData[html.MetaKeySource])
		assert.Equal(t, "Test html in url loader", docs[0].MetaData[html.MetaKeyTitle])
	})

	t.Run("md loader", func(t *testing.T) {
		p := &MockParser{
			mock: func(reader io.Reader) ([]*schema.Document, error) {
				data, err := io.ReadAll(reader)
				assert.Nil(t, err)

				return []*schema.Document{
					{
						Content: string(data),
					},
				}, nil
			},
		}

		loader, err := NewLoader(ctx, &LoaderConfig{
			Parser: p,
		})
		assert.Nil(t, err)

		url := fmt.Sprintf("http://%s/test.md", addr)
		docs, err := loader.Load(ctx, document.Source{
			URI: url,
		})
		assert.Nil(t, err)

		assert.Equal(t, 1, len(docs))
		assert.Equal(t, "# Title\nhello world", docs[0].Content)
	})

	t.Run("custom request builder and custom client", func(t *testing.T) {
		loader, err := NewLoader(ctx, &LoaderConfig{
			RequestBuilder: func(ctx context.Context, src document.Source, opts ...document.LoaderOption) (*http.Request, error) {
				url := "file:///test.html"
				return http.NewRequest("GET", url, nil)
			},
			Client: &http.Client{
				Timeout:   5 * time.Second,
				Transport: http.NewFileTransport(http.Dir("./testdata")),
			},
		})
		assert.Nil(t, err)

		url := fmt.Sprintf("http://%s/test.xx", addr)
		docs, err := loader.Load(ctx, document.Source{
			URI: url,
		})
		assert.Nil(t, err)

		assert.Equal(t, 1, len(docs))
		assert.Equal(t, "Test html in url loader", docs[0].MetaData[html.MetaKeyTitle])
	})
}
