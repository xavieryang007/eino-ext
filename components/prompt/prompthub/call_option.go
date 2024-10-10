package prompthub

import (
	"code.byted.org/flow/eino/components/prompt"
)

type options struct {
	UserID   *string
	DeviceID *string
	KV       map[string]any
}

func WithUserID(userID string) prompt.Option {
	return prompt.WrapImplSpecificOptFn(func(o *options) {
		o.UserID = &userID
	})
}

func WithDeviceID(deviceID string) prompt.Option {
	return prompt.WrapImplSpecificOptFn(func(o *options) {
		o.DeviceID = &deviceID
	})
}

func WithKV(kv map[string]any) prompt.Option {
	return prompt.WrapImplSpecificOptFn(func(o *options) {
		o.KV = kv
	})
}
