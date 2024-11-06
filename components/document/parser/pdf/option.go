package pdf

import "code.byted.org/flow/eino/components/document/parser"

type options struct {
	toPages *bool
}

// WithToPages is a parser option that specifies whether to parse the PDF into pages.
func WithToPages(toPages bool) parser.Option {
	return parser.WrapImplSpecificOptFn(func(opts *options) {
		opts.toPages = &toPages
	})
}
