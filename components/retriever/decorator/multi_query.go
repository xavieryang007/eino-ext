package decorator

import (
	"context"
	"fmt"
	"strings"

	"code.byted.org/flow/eino/callbacks"
	"code.byted.org/flow/eino/components"
	"code.byted.org/flow/eino/components/model"
	"code.byted.org/flow/eino/components/prompt"
	"code.byted.org/flow/eino/components/retriever"
	"code.byted.org/flow/eino/compose"
	"code.byted.org/flow/eino/schema"
)

const (
	defaultRewritePrompt = `You are an helpful assistant.
	Your task is to generate 3 different versions of the given question to retrieve relevant documents from a vector store. 
    By generating multiple perspectives on the user question, your goal is to help the user overcome some of the limitations of distance-based similarity search.
	Provide these alternative questions separated by newlines. 
	Original question: {{question}}`
	defaultQueryVariable = "question"
	defaultMaxQueriesNum = 5
)

var deduplicateFusion = func(ctx context.Context, docs [][]*schema.Document) ([]*schema.Document, error) {
	m := map[string]bool{}
	var ret []*schema.Document
	for i := range docs {
		for j := range docs[i] {
			if _, ok := m[docs[i][j].ID]; !ok {
				m[docs[i][j].ID] = true
				ret = append(ret, docs[i][j])
			}
		}
	}
	return ret, nil
}

// NewMultiQueryRetriever https://bytedance.larkoffice.com/wiki/G8T2w5bYuigJ4LkMi1ycw6VznAh#A4PqdcJmpoveWcxv8NPc70TanLb
func NewMultiQueryRetriever(ctx context.Context, config *MultiQueryConfig) (retriever.Retriever, error) {
	var err error

	// config validate
	if config.OrigRetriever == nil {
		return nil, fmt.Errorf("OrigRetriever is required")
	}
	if config.RewriteHandler == nil && config.RewriteLLM == nil {
		return nil, fmt.Errorf("at least one of RewriteHandler and RewriteLLM must not be empty")
	}

	// construct rewrite chain
	rewriteChain := compose.NewChain[string, []string]()
	if config.RewriteHandler != nil {
		rewriteChain.AppendLambda(compose.InvokableLambda(config.RewriteHandler), compose.WithNodeName("CustomQueryRewriter"))
	} else {
		tpl := config.RewriteTemplate
		variable := config.QueryVar
		parser := config.LLMOutputParser
		if tpl == nil {
			tpl = prompt.FromMessages(schema.Jinja2, schema.UserMessage(defaultRewritePrompt))
			variable = defaultQueryVariable
		}
		if parser == nil {
			parser = func(ctx context.Context, message *schema.Message) ([]string, error) {
				return strings.Split(message.Content, "\n"), nil
			}
		}

		rewriteChain.
			AppendLambda(compose.InvokableLambda(func(ctx context.Context, input string) (output map[string]any, err error) {
				return map[string]any{variable: input}, nil
			}), compose.WithNodeName("Converter")).
			AppendChatTemplate(tpl).
			AppendChatModel(config.RewriteLLM).
			AppendLambda(compose.InvokableLambda(parser), compose.WithNodeName("OutputParser"))
	}
	rewriteRunner, err := rewriteChain.Compile(ctx, compose.WithGraphName("QueryRewrite"))
	if err != nil {
		return nil, err
	}

	maxQueriesNum := config.MaxQueriesNum
	if maxQueriesNum == 0 {
		maxQueriesNum = defaultMaxQueriesNum
	}

	fusionFunc := config.FusionFunc
	if fusionFunc == nil {
		fusionFunc = deduplicateFusion
	}

	return &multiQueryRetriever{
		queryRunner:   rewriteRunner,
		maxQueriesNum: maxQueriesNum,
		origRetriever: config.OrigRetriever,
		fusionFunc:    fusionFunc,
	}, nil
}

type MultiQueryConfig struct {
	// Rewrite
	// 1. set the following fields to use llm to generate multi queries
	// 	a. chat model, required
	RewriteLLM model.ChatModel
	//	b. prompt llm to generate multi queries, we provide default template so you can leave this field blank
	RewriteTemplate prompt.ChatTemplate
	//	c. origin query variable of your custom template, it can be empty if you use default template
	QueryVar string
	//	d. parser llm output to queries, split content using "\n" by default
	LLMOutputParser func(context.Context, *schema.Message) ([]string, error)
	// 2. set RewriteHandler to provide custom query generation logic, possibly without a ChatModel. If this field is set, it takes precedence over other configurations above
	RewriteHandler func(ctx context.Context, query string) ([]string, error)
	// limit max queries num that Rewrite generates, and excess queries will be truncated, 5 by default
	MaxQueriesNum int

	// Origin Retriever
	OrigRetriever retriever.Retriever

	// fusion docs recalled from multi retrievers, remove dup based on document id by default
	FusionFunc func(ctx context.Context, docs [][]*schema.Document) ([]*schema.Document, error)
}

type multiQueryRetriever struct {
	queryRunner   compose.Runnable[string, []string]
	maxQueriesNum int
	origRetriever retriever.Retriever
	fusionFunc    func(ctx context.Context, docs [][]*schema.Document) ([]*schema.Document, error)
}

func (m *multiQueryRetriever) Retrieve(ctx context.Context, query string, opts ...retriever.Option) ([]*schema.Document, error) {
	// generate queries
	queries, err := m.queryRunner.Invoke(ctx, query)
	if err != nil {
		return nil, err
	}
	if len(queries) > m.maxQueriesNum {
		queries = queries[:m.maxQueriesNum]
	}

	// retrieve
	tasks := make([]*retrieveTask, len(queries))
	for i := range queries {
		tasks[i] = &retrieveTask{retriever: m.origRetriever, query: queries[i]}
	}
	concurrentRetrieveWithCallback(ctx, tasks)
	result := make([][]*schema.Document, len(queries))
	for i, task := range tasks {
		if task.err != nil {
			return nil, task.err
		}
		result[i] = task.result
	}

	// fusion
	ctx = ctxWithFusionCBM(ctx)
	fusionCBM, ok := callbacks.ManagerFromCtx(ctx)
	if ok {
		ctx = fusionCBM.OnStart(ctx, result)
	}
	fusionDocs, err := m.fusionFunc(ctx, result)
	if err != nil {
		if ok {
			fusionCBM.OnError(ctx, err)
		}
		return nil, err
	}
	if ok {
		fusionCBM.OnEnd(ctx, fusionDocs)
	}
	return fusionDocs, nil
}

func (m *multiQueryRetriever) GetType() string {
	return "MultiQuery"
}

func ctxWithRetrieverCBM(ctx context.Context, r retriever.Retriever) context.Context {
	cbm, ok := callbacks.ManagerFromCtx(ctx)
	if !ok {
		return ctx
	}

	runInfo := &callbacks.RunInfo{
		Component: components.ComponentOfRetriever,
	}

	if embType, ok := components.GetType(r); ok {
		runInfo.Type = embType
	}

	runInfo.Name = runInfo.Type + string(runInfo.Component)

	return callbacks.CtxWithManager(ctx, cbm.WithRunInfo(runInfo))
}

func ctxWithFusionCBM(ctx context.Context) context.Context {
	cbm, ok := callbacks.ManagerFromCtx(ctx)
	if !ok {
		return ctx
	}

	runInfo := &callbacks.RunInfo{
		Component: compose.ComponentOfLambda,
		Type:      "FusionFunc",
	}

	runInfo.Name = runInfo.Type + string(runInfo.Component)

	return callbacks.CtxWithManager(ctx, cbm.WithRunInfo(runInfo))
}
