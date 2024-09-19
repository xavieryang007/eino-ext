package idp

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"code.byted.org/flow/eino/components/document"
	"code.byted.org/flow/eino/compose"
	"code.byted.org/flow/eino/schema"
)

func TestLarkDocxErr(t *testing.T) {
	t.Run("test larkDocx no uri", func(t *testing.T) {
		src := document.Source{}
		ldl := newLarkDocLoader(LarkDocConfig{})

		_, err := ldl.Load(context.Background(), src, nil)
		assert.NotNil(t, err)
	})

	t.Run("test larkDocx error doc type", func(t *testing.T) {
		src := document.Source{URI: "https://bytedance.larkoffice.com/docxx/xxx"}
		ldl := newLarkDocLoader(LarkDocConfig{})
		_, err := ldl.Load(context.Background(), src, nil)
		assert.NotNil(t, err)
	})

	t.Run("test larkDocx error doc type unsupport", func(t *testing.T) {
		src := document.Source{URI: "https://bytedance.larkoffice.com/wiki/xxx"}
		ldl := newLarkDocLoader(LarkDocConfig{})
		_, err := ldl.Load(context.Background(), src, nil)
		assert.NotNil(t, err)
	})

	t.Run("test larkDocx error doc uri", func(t *testing.T) {
		src := document.Source{URI: "https://bytedance.larkoffice.com/docx/xx/xx"}
		ldl := newLarkDocLoader(LarkDocConfig{})
		_, err := ldl.Load(context.Background(), src, nil)
		assert.NotNil(t, err)
	})
}

func TestLarkDocInGraph(t *testing.T) {

	l := NewIDPLoaderSplitter(&Config{})

	t.Run("test larkDocx in graph", func(t *testing.T) {
		graph := compose.NewGraph[document.Source, []*schema.Document](compose.RunTypePregel)

		err := graph.AddLoaderSplitterNode("larkDoc", l)
		assert.Nil(t, err)

		err = graph.AddEdge(compose.START, "larkDoc")
		assert.Nil(t, err)

		err = graph.AddEdge("larkDoc", compose.END)
		assert.Nil(t, err)

		_, err = graph.Compile()
		assert.Nil(t, err)
	})

	t.Run("test larkDocx in chain", func(t *testing.T) {
		chain := compose.NewChain[document.Source, []*schema.Document]()
		chain.AppendLoaderSplitter(l)

		_, err := chain.Compile()
		assert.Nil(t, err)
	})

	t.Run("test larkDocx in parallel", func(t *testing.T) {
		chain := compose.NewChain[document.Source, map[string]any]()
		parallel := compose.NewParallel()
		parallel.AddLoaderSplitter("k1", l)

		chain.AppendParallel(parallel)
		_, err := chain.Compile()
		assert.Nil(t, err)
	})
}
