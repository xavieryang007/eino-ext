package decorator

import (
	"context"
	"fmt"
	"sync"

	"code.byted.org/flow/eino/callbacks"
	"code.byted.org/flow/eino/components/retriever"
	"code.byted.org/flow/eino/schema"
)

type retrieveTask struct {
	name            string
	retriever       retriever.Retriever
	query           string
	retrieveOptions []retriever.Option
	result          []*schema.Document
	err             error
}

func concurrentRetrieveWithCallback(ctx context.Context, tasks []*retrieveTask) {
	wg := sync.WaitGroup{}
	for i := range tasks {
		wg.Add(1)
		go func(ctx context.Context, t *retrieveTask) {
			ctx = ctxWithRetrieverCBM(ctx, t.retriever)
			retrieverCBM, ok := callbacks.ManagerFromCtx(ctx)
			needCallback := !callbacks.IsCallbacksEnabled(t.retriever)

			defer func() {
				if e := recover(); e != nil {
					t.err = fmt.Errorf("retrieve panic, query: %s, error: %v", t.query, e)
					if needCallback && ok {
						ctx = retrieverCBM.OnError(ctx, t.err)
					}
				}
				wg.Done()
			}()

			if needCallback && ok {
				ctx = retrieverCBM.OnStart(ctx, t.query)
			}
			docs, err := t.retriever.Retrieve(ctx, t.query, t.retrieveOptions...)
			if err != nil {
				if needCallback && ok {
					retrieverCBM.OnError(ctx, err)
				}
				t.err = err
				return
			}
			if needCallback && ok {
				retrieverCBM.OnEnd(ctx, docs)
			}
			t.result = docs
		}(ctx, tasks[i])
	}
	wg.Wait()
}
