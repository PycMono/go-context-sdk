package tracing

import (
	"net/http"

	"github.com/PycMono/go-context-sdk/bizctx"
)

type requestIDTransport struct {
	base http.RoundTripper
}

// NewRequestIDTransport injects the biz context request ID into outbound requests.
func NewRequestIDTransport(base http.RoundTripper) http.RoundTripper {
	if base == nil {
		base = http.DefaultTransport
	}
	return &requestIDTransport{base: base}
}

func (t *requestIDTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	requestID := bizctx.GetRequestID(request.Context())
	if requestID == "" {
		return t.base.RoundTrip(request)
	}

	cloned := request.Clone(request.Context())
	cloned.Header = request.Header.Clone()
	if cloned.Header == nil {
		cloned.Header = make(http.Header)
	}
	cloned.Header.Set(HeaderRequestID, requestID)
	return t.base.RoundTrip(cloned)
}
