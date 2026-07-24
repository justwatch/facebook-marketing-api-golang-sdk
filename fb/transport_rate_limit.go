package fb

import (
	"net/http"

	"github.com/go-kit/log"
)

type rateLimitTransport struct {
	l     log.Logger
	state *rateLimitState
	next  http.RoundTripper
}

func newRateLimitTransport(l log.Logger, state *rateLimitState, next http.RoundTripper) http.RoundTripper {
	if l == nil {
		l = log.NewNopLogger()
	}
	if next == nil {
		next = http.DefaultTransport
	}

	return &rateLimitTransport{
		l:     l,
		state: state,
		next:  next,
	}
}

func (t *rateLimitTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	t.state.waitForGate(r.Context())
	t.state.waitIfNeeded(r.Context())

	resp, err := t.next.RoundTrip(r)

	t.state.updateFromResponse(resp)

	return resp, err
}
