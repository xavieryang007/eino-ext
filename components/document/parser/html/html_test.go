package html

import (
	"context"
	"os"
	"testing"

	"code.byted.org/flow/eino/components/document/parser"
	"github.com/stretchr/testify/assert"
)

func TestHTMLParser(t *testing.T) {

	ctx := context.Background()
	p, err := NewParser(ctx, &Config{})
	assert.Nil(t, err)

	t.Run("Test normal Parser", func(t *testing.T) {
		f, err := os.Open("testdata/normal.html")
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()

		docs, err := p.Parse(ctx, f, parser.WithExtraMeta(map[string]any{"key": "value"}), parser.WithURI("http://localhost/testdata/normal.html"))
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, 1, len(docs))
		assert.Equal(t, "Test Document", docs[0].MetaData[MetaKeyTitle])
		assert.Equal(t, "Test Document\n\n\n    hello world!\n    content in xid", docs[0].Content)
		assert.Equal(t, "en", docs[0].MetaData[MetaKeyLang])
		assert.Equal(t, "UTF-8", docs[0].MetaData[MetaKeyCharset])
		assert.Equal(t, "http://localhost/testdata/normal.html", docs[0].MetaData[MetaKeySource])
		assert.Equal(t, "value", docs[0].MetaData["key"])
	})

	t.Run("test only text", func(t *testing.T) {
		f, err := os.Open("testdata/text.html")
		assert.NoError(t, err)
		defer f.Close()

		docs, err := p.Parse(context.Background(), f)
		assert.NoError(t, err)

		assert.Equal(t, 1, len(docs))
		assert.Equal(t, "hello world", docs[0].Content)
	})

	t.Run("test with selector", func(t *testing.T) {
		sel := "#xid"
		conf := &Config{
			Selector: &sel,
		}

		p, err := NewParser(context.Background(), conf)
		assert.NoError(t, err)
		f, err := os.Open("./testdata/normal.html")
		assert.NoError(t, err)
		defer f.Close()

		docs, err := p.Parse(context.Background(), f)
		assert.NoError(t, err)

		assert.Equal(t, 1, len(docs))
		assert.Equal(t, "content in xid", docs[0].Content)
	})

}
