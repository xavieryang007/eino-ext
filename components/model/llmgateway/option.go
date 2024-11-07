package llmgateway

import (
	"code.byted.org/flow/eino/components/model"
	"code.byted.org/flow/eino/schema"
	"code.byted.org/overpass/stone_llm_gateway/kitex_gen/stone/llm/gateway"
)

type PreProcessor func([]*schema.Message) ([]*schema.Message, error)

// gatewayOptions Request level advanced configs.
type gatewayOptions struct {
	chatOptions  *string
	userInfo     *gateway.UserInfo
	traffic      map[string]string
	extra        map[string]string
	preProcessor PreProcessor
}

func WithChatOption(opt *string) model.Option {
	return model.WrapImplSpecificOptFn[gatewayOptions](func(o *gatewayOptions) {
		o.chatOptions = opt
	})
}

func WithExtra(extra map[string]string) model.Option {
	return model.WrapImplSpecificOptFn[gatewayOptions](func(o *gatewayOptions) {
		o.extra = extra
	})
}

func WithUserInfo(userInfo *gateway.UserInfo) model.Option {
	return model.WrapImplSpecificOptFn[gatewayOptions](func(o *gatewayOptions) {
		o.userInfo = userInfo
	})
}

func WithTraffic(traffic map[string]string) model.Option {
	return model.WrapImplSpecificOptFn[gatewayOptions](func(o *gatewayOptions) {
		o.traffic = traffic
	})
}

func WithPreProcessor(processor PreProcessor) model.Option {
	return model.WrapImplSpecificOptFn[gatewayOptions](func(o *gatewayOptions) {
		o.preProcessor = processor
	})
}
