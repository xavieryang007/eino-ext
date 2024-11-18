package llmgateway

import (
	"code.byted.org/flow/eino/components/model"
	"code.byted.org/flow/eino/schema"
	"code.byted.org/overpass/stone_llm_gateway/kitex_gen/stone/llm/gateway"
	"github.com/bytedance/sonic"
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
func WithModel2MetaMap(modelId int64, metaId int64) model.Option {
	return model.WrapImplSpecificOptFn[gatewayOptions](func(o *gatewayOptions) {

		metaMap := make(map[int64]int64)
		metaMap[modelId] = metaId
		if o.extra != nil {
			data, err := sonic.MarshalString(metaMap)
			if err == nil {
				o.extra["model_meta_map"] = data
			}

		}
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
