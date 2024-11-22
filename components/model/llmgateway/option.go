package llmgateway

import (
	"code.byted.org/flow/eino/components/model"
	"code.byted.org/flow/eino/schema"
	"code.byted.org/gopkg/lang/conv"
	"code.byted.org/overpass/stone_llm_gateway/kitex_gen/stone/llm/gateway"

	"code.byted.org/flow/eino-ext/components/model/llmgateway/internal/utils"
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
func WithMetaId(metaId int64) model.Option {
	return model.WrapImplSpecificOptFn[gatewayOptions](func(o *gatewayOptions) {
		if o.extra != nil {
			o.extra[utils.MetaId] = conv.StringDefault(metaId, "")
		} else {
			o.extra = make(map[string]string)
			o.extra[utils.MetaId] = conv.StringDefault(metaId, "")
		}
	})
}

func WithExtra(extra map[string]string) model.Option {
	return model.WrapImplSpecificOptFn[gatewayOptions](func(o *gatewayOptions) {
		if o.extra != nil {
			copyMap(o.extra, extra)
		} else {
			o.extra = extra
		}
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
