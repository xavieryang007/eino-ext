package decorator

import (
	"context"
	"fmt"
	"sort"

	"code.byted.org/flow/eino/callbacks"
	"code.byted.org/flow/eino/components/retriever"
	"code.byted.org/flow/eino/compose"
	"code.byted.org/flow/eino/schema"
)

var rrf = func(ctx context.Context, result map[string][]*schema.Document) ([]*schema.Document, error) {
	if len(result) < 1 {
		return nil, fmt.Errorf("no documents")
	}
	if len(result) == 1 {
		for _, docs := range result {
			return docs, nil
		}
	}

	docRankMap := make(map[string]float64)
	docMap := make(map[string]*schema.Document)
	for _, v := range result {
		for i := range v {
			docMap[v[i].ID] = v[i]
			if _, ok := docRankMap[v[i].ID]; !ok {
				docRankMap[v[i].ID] = 1.0 / float64(i+60)
			} else {
				docRankMap[v[i].ID] += 1.0 / float64(i+60)
			}
		}
	}
	docList := make([]*schema.Document, 0, len(docMap))
	for id := range docMap {
		docList = append(docList, docMap[id])
	}

	sort.Slice(docList, func(i, j int) bool {
		return docRankMap[docList[i].ID] > docRankMap[docList[j].ID]
	})

	return docList, nil
}

// NewRouterRetriever https://bytedance.larkoffice.com/wiki/G8T2w5bYuigJ4LkMi1ycw6VznAh#LXz5dRzy0oxJJixa4OKcPZvznpg
func NewRouterRetriever(ctx context.Context, config *RouterConfig) (retriever.Retriever, error) {
	if len(config.Retrievers) == 0 {
		return nil, fmt.Errorf("retrievers is empty")
	}

	router := config.Router
	if router == nil {
		var retrieverSet []string
		for k := range config.Retrievers {
			retrieverSet = append(retrieverSet, k)
		}
		router = func(ctx context.Context, query string) ([]string, error) {
			return retrieverSet, nil
		}
	}

	fusion := config.FusionFunc
	if fusion == nil {
		fusion = rrf
	}

	return &routerRetriever{
		retrievers: config.Retrievers,
		router:     config.Router,
		fusionFunc: fusion,
	}, nil
}

type RouterConfig struct {
	Retrievers map[string]retriever.Retriever
	Router     func(ctx context.Context, query string) ([]string, error)
	FusionFunc func(ctx context.Context, result map[string][]*schema.Document) ([]*schema.Document, error)
}

type routerRetriever struct {
	retrievers map[string]retriever.Retriever
	router     func(ctx context.Context, query string) ([]string, error)
	fusionFunc func(ctx context.Context, result map[string][]*schema.Document) ([]*schema.Document, error)
}

func (e *routerRetriever) Retrieve(ctx context.Context, query string, opts ...retriever.Option) ([]*schema.Document, error) {
	routeCtx := ctxWithRouterCBM(ctx)
	cbm, ok := callbacks.ManagerFromCtx(routeCtx)
	if ok {
		routeCtx = cbm.OnStart(routeCtx, query)
	}
	retrieverNames, err := e.router(routeCtx, query)
	if err != nil {
		if ok {
			cbm.OnError(routeCtx, err)
		}
		return nil, err
	}
	if len(retrieverNames) == 0 {
		err = fmt.Errorf("no retriever has been selected")
		if ok {
			cbm.OnError(routeCtx, err)
		}
		return nil, err
	}
	if ok {
		cbm.OnEnd(routeCtx, retrieverNames)
	}

	// retrieve
	tasks := make([]*retrieveTask, len(retrieverNames))
	for i := range retrieverNames {
		r, ok := e.retrievers[retrieverNames[i]]
		if !ok {
			return nil, fmt.Errorf("router output[%s] has not registered", retrieverNames[i])
		}
		tasks[i] = &retrieveTask{
			name:            retrieverNames[i],
			retriever:       r,
			query:           query,
			retrieveOptions: opts,
		}
	}
	concurrentRetrieveWithCallback(ctx, tasks)
	result := make(map[string][]*schema.Document)
	for i := range tasks {
		if tasks[i].err != nil {
			return nil, tasks[i].err
		}
		result[tasks[i].name] = tasks[i].result
	}

	// fusion
	fusionCtx := ctxWithFusionCBM(ctx)
	fusionCBM, ok := callbacks.ManagerFromCtx(fusionCtx)
	if ok {
		fusionCtx = fusionCBM.OnStart(fusionCtx, result)
	}
	fusionDocs, err := e.fusionFunc(fusionCtx, result)
	if err != nil {
		if ok {
			fusionCBM.OnError(fusionCtx, err)
		}
		return nil, err
	}
	if ok {
		fusionCBM.OnEnd(fusionCtx, fusionDocs)
	}
	return fusionDocs, nil
}

func (e *routerRetriever) GetType() string { return "Router" }

func ctxWithRouterCBM(ctx context.Context) context.Context {
	cbm, ok := callbacks.ManagerFromCtx(ctx)
	if !ok {
		return ctx
	}

	runInfo := &callbacks.RunInfo{
		Component: compose.ComponentOfLambda,
		Type:      "Router",
	}

	runInfo.Name = runInfo.Type + string(runInfo.Component)

	return callbacks.CtxWithManager(ctx, cbm.WithRunInfo(runInfo))
}
