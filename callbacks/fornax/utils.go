package fornax

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strconv"

	"github.com/bytedance/sonic"

	"code.byted.org/flow/eino/components"
	"code.byted.org/flow/flow-telemetry-common/go/obtag"
)

func getErrorTags(_ context.Context, err error) spanTags {
	return make(spanTags).
		set(obtag.Error, err.Error()).
		set(obtag.StatusCode, obtag.VErrDefault)
}

type spanTags map[string]any

func (t spanTags) setTags(kv map[string]any) spanTags {
	for k, v := range kv {
		t.set(k, v)
	}

	return t
}

func (t spanTags) set(key string, value any) spanTags {
	if t == nil || value == nil {
		return t
	}

	if _, found := t[key]; found {
		return t
	}

	switch k := reflect.TypeOf(value).Kind(); k {
	case reflect.Array,
		reflect.Interface,
		reflect.Map,
		reflect.Pointer,
		reflect.Slice,
		reflect.Struct:
		value = toJson(value, false)
	default:

	}

	t[key] = value

	return t
}

func (t spanTags) setIfNotZero(key string, val any) {
	if val == nil {
		return
	}

	rv := reflect.ValueOf(val)
	if rv.IsValid() && rv.IsZero() {
		return
	}

	t.set(key, val)
}

func (t spanTags) setFromExtraIfNotZero(key string, extra map[string]any) {
	if extra == nil {
		return
	}

	t.setIfNotZero(key, extra[key])
}

func setMetricsVariablesValue(ctx context.Context, val *metricsVariablesValue) context.Context {
	if val == nil {
		return ctx
	}

	return context.WithValue(ctx, metricsVariablesKey{}, val)
}

func getMetricsVariablesValue(ctx context.Context) (*metricsVariablesValue, bool) {
	val, ok := ctx.Value(metricsVariablesKey{}).(*metricsVariablesValue)
	return val, ok
}

func setTraceVariablesValue(ctx context.Context, val *traceVariablesValue) context.Context {
	if val == nil {
		return ctx
	}

	return context.WithValue(ctx, traceVariablesKey{}, val)
}

func getTraceVariablesValue(ctx context.Context) (*traceVariablesValue, bool) {
	val, ok := ctx.Value(traceVariablesKey{}).(*traceVariablesValue)
	return val, ok
}

func setMetricsGraphName(ctx context.Context, name string) context.Context {
	if name == "" {
		return ctx
	}

	return context.WithValue(ctx, metricsGraphNameKey{}, name)
}

func getMetricsGraphName(ctx context.Context) string {
	val, _ := ctx.Value(metricsGraphNameKey{}).(string)
	return val
}

func isInfraComponent(component components.Component) bool {
	_, found := infraComponents[component]
	return found
}

func toJson(v any, bStream bool) string {
	if v == nil {
		return fmt.Sprintf("%s", errors.New("try to marshal nil error"))
	}
	if bStream {
		v = map[string]any{"stream": v}
	}
	b, err := sonic.MarshalString(v)
	if err != nil {
		return fmt.Sprintf("%s", err.Error())
	}
	return b
}

func itoa(i int64) string {
	return strconv.FormatInt(i, 10)
}
