package semantic

import (
	"code.byted.org/flow/eino/components/document"
	"code.byted.org/flow/eino/components/embedding"
	"code.byted.org/flow/eino/schema"
	"context"
	"math/rand/v2"
	"reflect"
	"testing"
)

type randomEmbedding struct {
	vecLen int
}

func (r *randomEmbedding) EmbedStrings(ctx context.Context, texts []string, opts ...embedding.Option) ([][]float64, error) {
	var ret [][]float64
	for range texts {
		piece := make([]float64, r.vecLen)
		for j := range piece {
			piece[j] = rand.Float64()
		}
		ret = append(ret, piece)
	}
	return ret, nil
}

func TestSemanticSplitter(t *testing.T) {
	type args struct {
		ctx  context.Context
		docs []*schema.Document
		opts []document.TransformerOption
	}
	tests := []struct {
		name      string
		config    *Config
		input     []*schema.Document
		outputLen int
	}{
		{
			name: "success",
			config: &Config{
				Embedding:    &randomEmbedding{vecLen: 5},
				BufferSize:   1,
				MinChunkSize: 9,
				Separators:   []string{"."},
				LenFunc:      nil,
				Percentile:   0.5,
			},
			input: []*schema.Document{{
				Content: "1234567890.1234567890.1234567890.1234567890.1234567890.1234567890",
			}},
			outputLen: 4,
		},
		{
			name: "corner case: text has not exceeded MinChunkSize",
			config: &Config{
				Embedding:    &randomEmbedding{vecLen: 5},
				BufferSize:   1,
				MinChunkSize: 9,
				Separators:   []string{"."},
				LenFunc:      nil,
				Percentile:   0.5,
			},
			input: []*schema.Document{{
				Content: "1.1.1.1",
			}},
			outputLen: 1,
		},
		{
			name: "corner case: percentile is too big",
			config: &Config{
				Embedding:    &randomEmbedding{vecLen: 5},
				BufferSize:   1,
				MinChunkSize: 9,
				Separators:   []string{"."},
				LenFunc:      nil,
				Percentile:   0.9999,
			},
			input: []*schema.Document{{
				Content: "1234567890.1234567890.1234567890.1234567890.1234567890.1234567890",
			}},
			outputLen: 2,
		},
		{
			name: "corner case: percentile is too small",
			config: &Config{
				Embedding:    &randomEmbedding{vecLen: 5},
				BufferSize:   1,
				MinChunkSize: 9,
				Separators:   []string{"."},
				LenFunc:      nil,
				Percentile:   0.00001,
			},
			input: []*schema.Document{{
				Content: "1234567890.1234567890.1234567890.1234567890.1234567890.1234567890",
			}},
			outputLen: 6,
		},
	}
	ctx := context.Background()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, err := NewSplitter(ctx, tt.config)
			if err != nil {
				t.Fatal(err)
			}
			for i := 0; i < 10; i++ {
				got, err := s.Transform(ctx, tt.input)
				if err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(len(got), tt.outputLen) {
					t.Errorf("Transform() got = %v, want %v", got, tt.outputLen)
				}
			}
		})
	}
}
