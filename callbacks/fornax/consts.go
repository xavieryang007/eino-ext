package fornax

import (
	"time"

	"code.byted.org/flow/eino/callbacks"
	"code.byted.org/flow/eino/components"
	"code.byted.org/flow/eino/compose"
)

type traceStreamInputAsyncKey struct{}

type streamInputAsyncVal chan struct{}

type metricsVariablesKey struct{}

type metricsVariablesValue struct {
	graphName     string
	startTime     time.Time
	callbackInput callbacks.CallbackInput
}

type metricsGraphNameKey struct{}

const (
	customSpanTagKeyName      = "eino_name"
	customSpanTagKeyType      = "eino_type"
	customSpanTagKeyComponent = "eino_component"
)

var infraComponents = map[components.Component]struct{}{
	compose.ComponentOfGraph:      {},
	compose.ComponentOfStateGraph: {},
	compose.ComponentOfChain:      {},
}
