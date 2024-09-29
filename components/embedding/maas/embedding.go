package maas

import (
	"context"
	"fmt"

	"github.com/volcengine/volcengine-go-sdk/service/arkruntime"
	"github.com/volcengine/volcengine-go-sdk/service/arkruntime/model"

	"code.byted.org/flow/eino/callbacks"
	"code.byted.org/flow/eino/components/embedding"
)

var (
	defaultBaseURL = "https://ark.cn-beijing.volces.com/api/v3"
	defaultRegion  = "cn-beijing"
)

type EmbeddingConfig struct {
	// URL of maas endpoint, default "https://ark.cn-beijing.volces.com/api/v3".
	BaseURL string
	// Region of maas endpoint, default "cn-beijing", see more
	Region string

	// one of APIKey or ak/sk must be set for authorization.
	APIKey               string
	AccessKey, SecretKey string

	// endpoint_id of the model you use in ark platform, mostly like `ep-20xxxxxxx-xxxxx`.
	Model string
	// A unique identifier representing your end-user, which will help to monitor and detect abuse. see more at https://github.com/volcengine/volcengine-go-sdk/blob/master/service/arkruntime/model/embeddings.go
	User *string
	// Dimensions The number of dimensions the resulting output embeddings should have, different between models.
	Dimensions *int
}

type Embedder struct {
	client *arkruntime.Client
	conf   *EmbeddingConfig
}

func buildClient(config *EmbeddingConfig) *arkruntime.Client {
	if config.BaseURL == "" {
		config.BaseURL = defaultBaseURL
	}
	if config.Region == "" {
		config.Region = defaultRegion
	}

	if config.APIKey != "" {
		return arkruntime.NewClientWithApiKey(
			config.APIKey,
			arkruntime.WithBaseUrl(config.BaseURL),
			arkruntime.WithRegion(config.Region),
		)
	}

	return arkruntime.NewClientWithAkSk(
		config.AccessKey,
		config.SecretKey,
		arkruntime.WithBaseUrl(config.BaseURL),
		arkruntime.WithRegion(config.Region),
	)
}

func NewEmbedder(ctx context.Context, config *EmbeddingConfig) (*Embedder, error) {

	client := buildClient(config)

	return &Embedder{
		client: client,
		conf:   config,
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

	req := e.genRequest(texts, opts...)
	conf := &embedding.Config{
		Model:          req.Model,
		EncodingFormat: string(req.EncodingFormat),
	}

	if cbmOK {
		ctx = cbm.OnStart(ctx, &embedding.CallbackInput{
			Texts:  texts,
			Config: conf,
		})
	}

	resp, err := e.client.CreateEmbeddings(ctx, &req)
	if err != nil {
		return nil, fmt.Errorf("[MaaS]EmbedStrings error: %v", err)
	}

	var usage *embedding.TokenUsage

	usage = &embedding.TokenUsage{
		PromptTokens:     resp.Usage.PromptTokens,
		CompletionTokens: resp.Usage.CompletionTokens,
		TotalTokens:      resp.Usage.TotalTokens,
	}

	embeddings = make([][]float64, len(resp.Data))
	for i, d := range resp.Data {
		embeddings[i] = toFloat64(d.Embedding)
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
	return getType()
}

func (e *Embedder) IsCallbacksEnabled() bool {
	return true
}

func (e *Embedder) genRequest(texts []string, opts ...embedding.Option) (
	req model.EmbeddingRequestStrings) {
	options := &embedding.Options{
		Model: &e.conf.Model,
	}

	options = embedding.GetCommonOptions(options, opts...)

	req = model.EmbeddingRequestStrings{
		Input:          texts,
		Model:          dereferenceOrZero(options.Model),
		User:           dereferenceOrZero(e.conf.User),
		EncodingFormat: model.EmbeddingEncodingFormatFloat, // only support Float for now?
		Dimensions:     dereferenceOrZero(e.conf.Dimensions),
	}

	return req
}

func toFloat64(in []float32) []float64 {
	out := make([]float64, len(in))
	for i, v := range in {
		out[i] = float64(v)
	}
	return out
}
