package markdown

import (
	"code.byted.org/flow/eino/schema"
	"context"
	"reflect"
	"testing"
)

func TestMarkdownHeaderSplitter(t *testing.T) {
	tests := []struct {
		name   string
		config *HeaderConfig
		input  []*schema.Document
		want   []*schema.Document
	}{
		{
			name: "success",
			config: &HeaderConfig{
				Headers: map[string]string{
					"#":   "Header1",
					"##":  "Header2",
					"###": "Header3",
				},
				TrimHeaders: true,
			},
			input: []*schema.Document{{
				ID:       "id",
				Content:  "# Header1\n\n ```code1\ncode2\ncode3\n```\n ## Header2\n\nContent1\n\n ### Header3 \n\n Content2 \n\n ## Header4\n\n Content3",
				MetaData: map[string]interface{}{},
			}},
			want: []*schema.Document{{
				ID:      "id",
				Content: "```code1\ncode2\ncode3\n```",
				MetaData: map[string]interface{}{
					"Header1": "Header1",
				},
			}, {
				ID:      "id",
				Content: "Content1",
				MetaData: map[string]interface{}{
					"Header1": "Header1",
					"Header2": "Header2",
				},
			}, {
				ID:      "id",
				Content: "Content2",
				MetaData: map[string]interface{}{
					"Header1": "Header1",
					"Header2": "Header2",
					"Header3": "Header3",
				},
			}, {
				ID:      "id",
				Content: "Content3",
				MetaData: map[string]interface{}{
					"Header1": "Header1",
					"Header2": "Header4",
				},
			}},
		},
	}
	ctx := context.Background()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			splitter, err := NewHeaderSplitter(ctx, tt.config)
			if err != nil {
				t.Fatal(err)
			}
			ret, err := splitter.Transform(ctx, tt.input)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(ret, tt.want) {
				t.Errorf("NewHeaderSplitter() got = %v, want %v", ret, tt.want)
			}
		})
	}
}
