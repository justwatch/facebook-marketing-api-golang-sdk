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
					t.waitForRetry(r)
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

// waitForRetry pauses before a retry attempt. It prefers the reset duration
// from the last known rate-limit headers; the exponential backoff timer
// handles further spacing between attempts.
func (t *retryTransport) waitForRetry(r *http.Request) {
	if t.state == nil {
		return
	}
	dur := t.state.blockDurationForRetry()
	if dur <= 0 {
		return
	}
	t.state.sleep(r.Context(), dur)
}
