package openai

import (
	"context"
	"net/http"
	"time"

	"github.com/sashabaranov/go-openai"

	"code.byted.org/flow/eino-ext/components/embedding/openai/internal/transport"
	"code.byted.org/flow/eino/callbacks"
	"code.byted.org/flow/eino/components/embedding"
)

type EmbeddingEncodingFormat string

const (
	EmbeddingEncodingFormatFloat  EmbeddingEncodingFormat = "float"
	EmbeddingEncodingFormatBase64 EmbeddingEncodingFormat = "base64"
)

type EmbeddingConfig struct {
	BaseURL string `json:"base_url"`
	APIKey  string `json:"api_key"`
	ByAzure bool   `json:"by_azure"`

	Model string `json:"model"`
	User  string `json:"user"`
	// EmbeddingEncodingFormat is the format of the embedding data.
	// Currently, only "float" and "base64" are supported, however, "base64" is not officially documented.
	// If not specified OpenAI will use "float".
	EncodingFormat EmbeddingEncodingFormat `json:"encoding_format,omitempty"`
	// Dimensions The number of dimensions the resulting output embeddings should have.
	// Only supported in text-embedding-3 and later models.
	Dimensions int `json:"dimensions,omitempty"`

	Timeout time.Duration `json:"timeout"`
}

var _ embedding.Embedder = (*Embedder)(nil)

type Embedder struct {
	cli    *openai.Client
	config *EmbeddingConfig
}

func NewEmbedder(ctx context.Context, config *EmbeddingConfig) (*Embedder, error) {
	if config == nil {
		config = &EmbeddingConfig{Model: string(openai.AdaEmbeddingV2)}
	}

	var clientConf openai.ClientConfig

	if config.ByAzure {
		clientConf = openai.DefaultAzureConfig(config.APIKey, config.BaseURL)
	} else {
		clientConf = openai.DefaultConfig(config.APIKey)
	}

	clientConf.HTTPClient = &http.Client{
		Timeout:   config.Timeout,
		Transport: &transport.HeaderTransport{Origin: http.DefaultTransport},
	}

	return &Embedder{
		cli:    openai.NewClientWithConfig(clientConf),
		config: config,
	}, nil
}

func (e *Embedder) EmbedStrings(ctx context.Context, texts []string, opts ...embedding.Option) (
	embeddings [][]float64, err error) {

	var (
		cbm, cbmOK = callbacks.ManagerFromCtx(ctx)
	)

	defer func() {
		if err != nil && cbmOK {
			_ = cbm.OnError(ctx, err)
		}
	}()

	req := &openai.EmbeddingRequest{
		Input:          texts,
		Model:          openai.EmbeddingModel(e.config.Model),
		User:           e.config.User,
		EncodingFormat: openai.EmbeddingEncodingFormat(e.config.EncodingFormat),
		Dimensions:     e.config.Dimensions,
	}

	runtimeModifyReq(req, opts...)

	conf := &embedding.Config{
		Model:          string(req.Model),
		EncodingFormat: string(req.EncodingFormat),
	}

	ctx = cbm.OnStart(ctx, &embedding.CallbackInput{
		Texts:  texts,
		Config: conf,
	})

	resp, err := e.cli.CreateEmbeddings(ctx, *req)
	if err != nil {
		return nil, err
	}

	embeddings = make([][]float64, len(resp.Data))
	for i, d := range resp.Data {
		res := make([]float64, len(d.Embedding))
		for j, emb := range d.Embedding {
			res[j] = float64(emb)
		}
		embeddings[i] = res
	}

	usage := &embedding.TokenUsage{
		PromptTokens:     resp.Usage.PromptTokens,
		CompletionTokens: resp.Usage.CompletionTokens,
		TotalTokens:      resp.Usage.TotalTokens,
	}

	if cbmOK {
		_ = cbm.OnEnd(ctx, &embedding.CallbackOutput{
			Embeddings: embeddings,
			Config:     conf,
			TokenUsage: usage,
		})
	}

	return embeddings, nil
}

func (e *Embedder) GetType() string {
	return typ
}

func (e *Embedder) IsCallbacksEnabled() bool {
	return true
}

func runtimeModifyReq(req *openai.EmbeddingRequest, opts ...embedding.Option) {
	if req == nil {
		return
	}
	if len(opts) == 0 {
		return
	}
	o := embedding.GetEmbeddingOptions(opts...)

	if o.Model != nil {
		req.Model = openai.EmbeddingModel(*o.Model)
	}
}
