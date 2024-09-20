package cozeplugin

import (
	"context"
	"testing"

	"github.com/cloudwego/kitex/client/callopt/streamcall"

	"code.byted.org/kite/kitex/client/callopt"
)

func TestCozePluginCallOption(t *testing.T) {
	options := getPluginOption(
		WithDeviceID(1),
		WithUserID(2),
		WithExtra(map[string]string{"3": "4"}),
		WithInputModifier(func(ctx context.Context, s string) (string, error) {
			return "", nil
		}),
		WithKitexCallOptions(callopt.WithCluster("")),
		WithStreamKitexCallOptions(streamcall.WithURL("")),
	)
	po, ok := options.ImplSpecificOption.(*pluginOption)
	if !ok {
		t.Fatalf("options.ImplSpecificOption isn't *pluginOption")
	}
	if po.UserID != int64(2) ||
		*po.DeviceID != int64(1) ||
		po.Extra["3"] != "4" {
		t.Fatalf("coze plugin call option error")
	}
}
