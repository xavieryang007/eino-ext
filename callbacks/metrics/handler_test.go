package metrics

import (
	"context"
	"testing"

	"github.com/bytedance/mockey"

	"code.byted.org/flow/eino/callbacks"
	"code.byted.org/flow/eino/compose"
	"code.byted.org/flow/eino/schema"
	"code.byted.org/gopkg/metrics/v4"
)

func TestFakeMetricsHandler(t *testing.T) {
	nopCli := metrics.NewNoopClient()
	defer mockey.Mock(metrics.NewClient).Return(nopCli, nil).Build().UnPatch()

	h, err := NewMetricsHandler()
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	info := &callbacks.RunInfo{
		Name:      "name",
		Type:      "type",
		Component: compose.ComponentOfGraph,
	}
	h.OnStart(ctx, info, nil)
	h.OnEnd(ctx, info, nil)
	h.OnError(ctx, info, nil)

	inputReader, inputWriter := schema.Pipe[callbacks.CallbackInput](1)
	inputWriter.Send("", nil)
	inputWriter.Close()
	h.OnStartWithStreamInput(ctx, info, inputReader)
	outputReader, outputWriter := schema.Pipe[callbacks.CallbackOutput](1)
	outputWriter.Send("", nil)
	outputWriter.Close()
	h.OnEndWithStreamOutput(ctx, info, outputReader)
}
