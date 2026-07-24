package fb

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/cenk/backoff"
)

type retryTransport struct {
	next  http.RoundTripper
	state *rateLimitState // may be nil; used for header-informed retry waits
}

func newRetryTransport(next http.RoundTripper, state *rateLimitState) http.RoundTripper {
	if next == nil {
		next = http.DefaultTransport
	}

	return &retryTransport{
		next:  next,
		state: state,
	}
}

func (t *retryTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	bo := backoff.NewExponentialBackOff()
	bo.InitialInterval = 6 * time.Second
	bo.MaxElapsedTime = 10 * time.Minute
	var resp *http.Response
	var attempt int
	err := backoff.Retry(func() error {
		attempt++
		var e error

		// The same *http.Request is reused across retry attempts and its Body is
		// consumed by the first RoundTrip. Without restoring it, retried requests
		// send an empty body, which Meta reports as a misleading "name field is
		// required" error. Reset the body from GetBody before every attempt.
		if r.GetBody != nil {
			body, getBodyErr := r.GetBody()
			if getBodyErr != nil {
				return backoff.Permanent(getBodyErr)
			}
			r.Body = body
		}

		resp, e = t.next.RoundTrip(r) // nolint:bodyclose // not a correct linter detection

		if e != nil {
			return e
		}

		// 5xx and 4xx: read body to decide whether to retry.
		// 200 responses may also contain JSON errors, but those are surfaced
		// by the client's handleResponse and handled by the caller.
		if resp.StatusCode >= 400 {
			body, readErr := io.ReadAll(resp.Body)
			resp.Body.Close()
			if readErr != nil {
				return fmt.Errorf("unexpected status %s from facebook, attempt %d", resp.Status, attempt)
			}

			ec := &ErrorContainer{}
			if jsonErr := json.Unmarshal(body, ec); jsonErr == nil && ec.Error != nil {
				// "reduce the amount of data" is not retryable — pass through to caller.
				if ec.Error.Code == 1 && strings.Contains(ec.Error.Message, "reduce the amount of data") {
					resp.Body = io.NopCloser(bytes.NewReader(body))
					return nil
				}

				// Rate-limited or transient: wait for the header-indicated reset window
				// before retrying so we don't worsen the throttle score.
				if IsRateLimited(ec.Error) || ec.Error.IsTransient {
					resp.Body = io.NopCloser(bytes.NewReader(body))
					t.throttle(r)
					return fmt.Errorf("rate limited by facebook (code=%d subcode=%d), attempt %d", ec.Error.Code, ec.Error.ErrorSubcode, attempt)
				}
			}

			// Non-retryable 4xx (e.g. auth errors, bad requests): pass through.
			if resp.StatusCode < 500 {
				resp.Body = io.NopCloser(bytes.NewReader(body))
				return nil
			}

			return fmt.Errorf("unexpected status %s from facebook, attempt %d", resp.Status, attempt)
		}

		return nil
	}, backoff.WithContext(bo, r.Context()))
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// throttle opens the shared cool-down gate after a throttle response and waits
// it out (with jitter), so concurrent callers back off together instead of
// retrying in lockstep. The exponential backoff timer handles further spacing
// between attempts. No-op when no shared state is configured.
func (t *retryTransport) throttle(r *http.Request) {
	if t.state == nil {
		return
	}
	t.state.registerThrottle()
	t.state.waitForGate(r.Context())
}
