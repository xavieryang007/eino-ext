package transport

import (
	"net/http"

	"code.byted.org/gopkg/ctxvalues"
)

type HeaderTransport struct {
	Origin http.RoundTripper
}

func (h *HeaderTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	if request.Header == nil {
		request.Header = make(http.Header)
	}

	if logID, ok := ctxvalues.LogID(request.Context()); ok {
		request.Header.Set("X-TT-LOGID", logID)
	}

	return h.Origin.RoundTrip(request)
}
