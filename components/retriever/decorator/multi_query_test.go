package decorator

import (
	"code.byted.org/flow/eino/components/model"
	"code.byted.org/flow/eino/components/retriever"
	"code.byted.org/flow/eino/compose"
	"code.byted.org/flow/eino/schema"
	"context"
	"strings"
	"testing"
)

type mockRetriever struct {
}

func (m *mockRetriever) Retrieve(ctx context.Context, query string, opts ...retriever.Option) ([]*schema.Document, error) {
	ret := []*schema.Document{}
	if strings.Contains(query, "1") {
		ret = append(ret, &schema.Document{ID: "1"})
	}
	if strings.Contains(query, "2") {
		ret = append(ret, &schema.Document{ID: "2"})
	}
	if strings.Contains(query, "3") {
		ret = append(ret, &schema.Document{ID: "3"})
	}
	if strings.Contains(query, "4") {
		ret = append(ret, &schema.Document{ID: "4"})
	}
	if strings.Contains(query, "5") {
		ret = append(ret, &schema.Document{ID: "5"})
	}
	return ret, nil
}

type mockModel struct {
}

func (m *mockModel) Generate(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.Message, error) {
	return &schema.Message{
		Content: "12\n23\n34\n14\n23\n45",
	}, nil
}

func (m *mockModel) Stream(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	panic("implement me")
}

func (m *mockModel) BindTools(tools []*schema.ToolInfo) error {
	panic("implement me")
}

func TestMultiQueryRetriever(t *testing.T) {
	ctx := context.Background()

	// use default llm
	mqr, err := NewMultiQueryRetriever(ctx, &MultiQueryConfig{
		RewriteLLM:    &mockModel{},
		OrigRetriever: &mockRetriever{},
	})
	if err != nil {
		t.Fatal(err)
	}
	c := compose.NewChain[string, []*schema.Document]()
	cr, err := c.AppendRetriever(mqr).Compile(ctx)
	if err != nil {
		t.Fatal(err)
	}

	result, err := cr.Invoke(ctx, "query")
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 4 {
		t.Fatal("default llm retrieve result is unexpected")
	}

	// use custom
	mqr, err = NewMultiQueryRetriever(ctx, &MultiQueryConfig{
		RewriteHandler: func(ctx context.Context, query string) ([]string, error) {
			return []string{"1", "3", "5"}, nil
		},
		OrigRetriever: &mockRetriever{},
	})
	if err != nil {
		t.Fatal(err)
	}
	c = compose.NewChain[string, []*schema.Document]()
	cr, err = c.AppendRetriever(mqr).Compile(ctx)
	if err != nil {
		t.Fatal(err)
	}

	result, err = cr.Invoke(ctx, "query")
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 3 {
		t.Fatal("default llm retrieve result is unexpected")
	}
}
