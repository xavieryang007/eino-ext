package idp

import "code.byted.org/flow/eino/components/document"

type option struct {
	// LarkDocAccessKey is the `user_access_token` of the user who has permission to access the document by feishu openapi.
	// mostly, you should manage it in your app.
	LarkDocAccessKey string
}

func WithLarkDocAccessKey(accessKey string) document.LoaderSplitterOption {
	return func(opts *document.LoaderSplitterOptions) {
		if o, ok := opts.ImplSpecificOption.(*option); ok {
			o.LarkDocAccessKey = accessKey
		}

		return
	}
}
