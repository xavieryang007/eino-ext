package decorator

import (
	"context"
	"reflect"
	"testing"

	"code.byted.org/flow/eino/components/retriever"
	"code.byted.org/flow/eino/schema"
)

func TestRouterRetriever(t *testing.T) {
	ctx := context.Background()
	r, err := NewRouterRetriever(ctx, &RouterConfig{
		Retrievers: map[string]retriever.Retriever{
			"1": &mockRetriever{},
			"2": &mockRetriever{},
			"3": &mockRetriever{},
		},
		Router: func(ctx context.Context, query string) ([]string, error) {
			return []string{"2", "3"}, nil
		},
		FusionFunc: func(ctx context.Context, result map[string][]*schema.Document) ([]*schema.Document, error) {
			var ret []*schema.Document
			for _, v := range result {
				ret = append(ret, v...)
			}
			return ret, nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	result, err := r.Retrieve(ctx, "3")
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 2 {
		t.Fatal("expected 2 results")
	}
}

func TestRRF(t *testing.T) {
	doc1 := &schema.Document{ID: "1"}
	doc2 := &schema.Document{ID: "2"}
	doc3 := &schema.Document{ID: "3"}
	doc4 := &schema.Document{ID: "4"}
	doc5 := &schema.Document{ID: "5"}

	input := map[string][]*schema.Document{
		"1": {doc1, doc2, doc3, doc4, doc5},
		"2": {doc2, doc3, doc4, doc5, doc1},
		"3": {doc3, doc4, doc5, doc1, doc2},
	}

	result, err := rrf(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(result, []*schema.Document{doc3, doc2, doc4, doc1, doc5}) {
		t.Fatal("rrf fail")
	}
}
