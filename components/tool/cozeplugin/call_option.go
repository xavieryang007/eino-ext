package cozeplugin

import (
	"context"

	"github.com/cloudwego/kitex/client/callopt"
	"github.com/cloudwego/kitex/client/callopt/streamcall"

	"code.byted.org/flow/eino/components/tool"
)

type pluginOptions struct {
	UserID        int64
	DeviceID      *int64
	Extra         map[string]string
	InputModifier func(context.Context, string) (string, error)

	callOpts       []callopt.Option
	streamCallOpts []streamcall.Option
}

// WithUserID request coze plugin with user id.
func WithUserID(userID int64) tool.Option {
	return tool.WrapImplSpecificOptFn(func(o *pluginOptions) {
		o.UserID = userID
	})
}

// WithDeviceID request coze plugin with device id.
func WithDeviceID(deviceID int64) tool.Option {
	return tool.WrapImplSpecificOptFn(func(o *pluginOptions) {
		o.DeviceID = &deviceID
	})
}

// WithExtra request coze plugin with extra.
func WithExtra(extra map[string]string) tool.Option {
	return tool.WrapImplSpecificOptFn(func(o *pluginOptions) {
		o.Extra = extra
	})
}

// WithInputModifier if you want to modify tool input before request, you can set InputModifier.
func WithInputModifier(inputModifier func(context.Context, string) (string, error)) tool.Option {
	return tool.WrapImplSpecificOptFn(func(o *pluginOptions) {
		o.InputModifier = inputModifier
	})
}

// WithKitexCallOptions call options set here will be passed through to the kitex request for executing coze plugin, with higher priority than the options set in InitCallOpts.
func WithKitexCallOptions(opts ...callopt.Option) tool.Option {
	return tool.WrapImplSpecificOptFn(func(o *pluginOptions) {
		o.callOpts = append(o.callOpts, opts...)
	})
}

// WithStreamKitexCallOptions stream call options set here will be passed through to the stream kitex request for executing coze plugin, with higher priority than the options set in InitStreamCallOpts.
func WithStreamKitexCallOptions(opts ...streamcall.Option) tool.Option {
	return tool.WrapImplSpecificOptFn(func(o *pluginOptions) {
		o.streamCallOpts = append(o.streamCallOpts, opts...)
	})
}
