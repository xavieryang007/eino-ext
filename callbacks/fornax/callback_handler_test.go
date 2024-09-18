package fornax

import (
	"testing"

	. "github.com/bytedance/mockey"
	"github.com/smartystreets/goconvey/convey"

	"code.byted.org/flow/eino/callbacks"
	"code.byted.org/flowdevops/fornax_sdk"
)

func TestNewDefaultTelemetryHandler(t *testing.T) {
	PatchConvey("test NewDefaultCallbackHandler", t, func() {
		hb := &callbacks.HandlerBuilder{}
		Mock(newMetricsCallbackHandler).Return(hb).Build()
		Mock(newTraceCallbackHandler).Return(hb).Build()

		handler := NewDefaultCallbackHandler(
			&fornax_sdk.Client{},
			WithCallbackDataParser(defaultDataParser{}))
		convey.So(handler, convey.ShouldNotBeNil)
	})
}
