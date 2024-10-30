package bytedgpt

import (
	"context"
	"net/http"
	"time"

	"code.byted.org/flow/eino/components/embedding"

	"code.byted.org/flow/eino-ext/components/embedding/bytedgpt/internal/transport"
	"code.byted.org/flow/eino-ext/components/embedding/protocols/openai"
)

type EmbeddingConfig struct {
	// if you want to use Azure OpenAI Service, set the next three fields. refs: https://learn.microsoft.com/en-us/azure/ai-services/openai/
	// ByAzure set this field to true when using Azure OpenAI Service, otherwise it does not need to be set.
	ByAzure bool `json:"by_azure"`
	// BaseURL https://{{$YOUR_RESOURCE_NAME}}.openai.azure.com, YOUR_RESOURCE_NAME is the name of your resource that you have created on Azure.
	BaseURL string `json:"base_url"`
	// APIVersion specifies the API version you want to use.
	APIVersion string `json:"api_version"`

	// APIKey is typically OPENAI_API_KEY, but if you have set up Azure, then it is Azure API_KEY.
	APIKey string `json:"api_key"`

	// Timeout specifies the http request timeout.
	Timeout time.Duration `json:"timeout"`

	// The following fields have the same meaning as the fields in the openai embedding API request. Ref: https://platform.openai.com/docs/api-reference/embeddings/create
	Model          string                          `json:"model"`
	EncodingFormat *openai.EmbeddingEncodingFormat `json:"encoding_format,omitempty"`
	Dimensions     *int                            `json:"dimensions,omitempty"`
	User           *string                         `json:"user,omitempty"`
}

var _ embedding.Embedder = (*Embedder)(nil)

type Embedder struct {
	cli *openai.OpenAIClient
}

func NewEmbedder(ctx context.Context, config *EmbeddingConfig) (*Embedder, error) {
	var nConf *openai.OpenAIConfig
	if config != nil {
		nConf = &openai.OpenAIConfig{
			ByAzure:        config.ByAzure,
			BaseURL:        config.BaseURL,
			APIVersion:     config.APIVersion,
			APIKey:         config.APIKey,
			HTTPClient:     &http.Client{Timeout: config.Timeout, Transport: &transport.HeaderTransport{Origin: http.DefaultTransport}},
			Model:          config.Model,
			EncodingFormat: config.EncodingFormat,
			Dimensions:     config.Dimensions,
			User:           config.User,
		}
	}
	cli, err := openai.NewOpenAIClient(ctx, nConf)
	if err != nil {
		return nil, err
	}

	return &Embedder{
		cli: cli,
	}, nil
}

func (e *Embedder) EmbedStrings(ctx context.Context, texts []string, opts ...embedding.Option) (
	embeddings [][]float64, err error) {
	return e.cli.EmbedStrings(ctx, texts, opts...)
}

const typ = "BytedGPT"

func (e *Embedder) GetType() string {
	return typ
}

func (e *Embedder) IsCallbacksEnabled() bool {
	return true
}
