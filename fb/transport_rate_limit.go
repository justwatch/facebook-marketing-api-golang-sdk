package fb

import (
	"net/http"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
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
	t.state.waitIfNeeded(r.Context())

	resp, err := t.next.RoundTrip(r)

	t.state.updateFromResponse(resp)

	if resp != nil {
		t.state.mu.Lock()
		pct := t.state.maxUsagePct
		t.state.mu.Unlock()

		if pct > 0 {
			_ = level.Info(t.l).Log(
				"msg", "facebook api usage",
				"usage_pct", pct,
				"block_duration", t.state.blockDurationForRetry(),
			)
		}
	}

	return resp, err
}
