package pdf

import (
	"context"
	"os"
	"testing"

	"code.byted.org/flow/eino/components/document/parser"
	"github.com/stretchr/testify/assert"
)

func TestLoader_Load(t *testing.T) {
	t.Run("TestLoader_Load", func(t *testing.T) {
		ctx := context.Background()

		f, err := os.Open("./testdata/test_pdf.pdf")
		assert.NoError(t, err)

		p, err := NewPDFParser(ctx, nil)
		assert.NoError(t, err)

		docs, err := p.Parse(ctx, f, WithToPages(true), parser.WithExtraMeta(map[string]any{"test": "test"}))
		assert.NoError(t, err)
		assert.Equal(t, 2, len(docs))
		assert.True(t, len(docs[0].Content) > 0)
		assert.Equal(t, map[string]any{"test": "test"}, docs[0].MetaData)
		assert.True(t, len(docs[0].Content) > 0)
		assert.Equal(t, map[string]any{"test": "test"}, docs[1].MetaData)
	})
}
