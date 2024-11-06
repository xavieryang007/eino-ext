package file

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"code.byted.org/flow/eino/components/document"
)

func TestFileLoader_Load(t *testing.T) {
	t.Run("TestFileLoader_Load", func(t *testing.T) {
		ctx := context.Background()
		loader, err := NewFileLoader(ctx, &FileLoaderConfig{
			UseNameAsID: true,
		})
		assert.NoError(t, err)

		docs, err := loader.Load(ctx, document.Source{
			URI: "./testdata/test.md",
		})
		assert.NoError(t, err)
		assert.Equal(t, 1, len(docs))
		assert.Equal(t, "test.md", docs[0].ID)
		assert.Equal(t, docs[0].Content, `# Title

- Bullet 1
- Bullet 2`)
		assert.Equal(t, 3, len(docs[0].MetaData))
		assert.Equal(t, "test.md", docs[0].MetaData[MetaKeyFileName])
		assert.Equal(t, ".md", docs[0].MetaData[MetaKeyExtension])
		assert.Equal(t, "./testdata/test.md", docs[0].MetaData[MetaKeySource])
	})
}
