package cozeplugin

import (
	"context"
	"testing"

	"code.byted.org/flow/eino/components/tool"
	"github.com/cloudwego/kitex/client/callopt/streamcall"

	"code.byted.org/kite/kitex/client/callopt"
)

func TestCozePluginCallOption(t *testing.T) {
	po := tool.GetImplSpecificOptions(
		&pluginOptions{},
		WithDeviceID(1),
		WithUserID(2),
		WithExtra(map[string]string{"3": "4"}),
		WithInputModifier(func(ctx context.Context, s string) (string, error) {
			return "", nil
		}),
		WithKitexCallOptions(callopt.WithCluster("")),
		WithStreamKitexCallOptions(streamcall.WithURL("")),
	)
	if po.UserID != int64(2) ||
		*po.DeviceID != int64(1) ||
		po.Extra["3"] != "4" {
		t.Fatalf("coze plugin call option error")
	}
}
