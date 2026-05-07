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
	next http.RoundTripper
}

func newRetryTransport(next http.RoundTripper) http.RoundTripper {
	if next == nil {
		next = http.DefaultTransport
	}

	return &retryTransport{
		next: next,
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
		} else if resp.StatusCode >= 500 {
			body, readErr := io.ReadAll(resp.Body)
			resp.Body.Close()
			if readErr != nil {
				return fmt.Errorf("unexpected status %s from facebook, attempt %d", resp.Status, attempt)
			}

			ec := &ErrorContainer{}
			if jsonErr := json.Unmarshal(body, ec); jsonErr == nil &&
				ec.Error != nil &&
				ec.Error.Code == 1 &&
				strings.Contains(ec.Error.Message, "reduce the amount of data") {
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
