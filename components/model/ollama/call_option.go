package ollama

import (
	"code.byted.org/flow/eino/components/model"
)

type options struct {
	Seed *int
}

func WithSeed(seed int) model.Option {
	return model.WrapImplSpecificOptFn(func(o *options) {
		o.Seed = &seed
	})
}
