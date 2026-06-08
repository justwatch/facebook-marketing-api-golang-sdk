package fb

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

// TestRetryTransport_ReplaysBodyOnRetry guards against a regression where the
// retry transport reused the same *http.Request without restoring its body.
// The first attempt consumes the body, so retried writes used to be sent with
// an empty body, which Meta reports as a misleading "name field is required".
func TestRetryTransport_ReplaysBodyOnRetry(t *testing.T) {
	if testing.Short() {
		t.Skip("retry backoff makes this test slow")
	}

	const wantBody = `{"name":"my-adset"}`
	var seen []string

	fake := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		body, _ := io.ReadAll(req.Body)
		seen = append(seen, string(body))

		// Fail the first attempt with a retryable 5xx, succeed afterwards.
		if len(seen) == 1 {
			return &http.Response{
				StatusCode: http.StatusInternalServerError,
				Status:     "500 Internal Server Error",
				Body:       io.NopCloser(strings.NewReader("server error")),
				Header:     make(http.Header),
			}, nil
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{}`)),
			Header:     make(http.Header),
		}, nil
	})

	rt := newRetryTransport(fake, nil)

	req, err := http.NewRequest(http.MethodPost, "https://graph.facebook.com/v24.0/act_1/adsets", bytes.NewReader([]byte(wantBody)))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}

	resp, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip returned error: %v", err)
	}
	resp.Body.Close()

	if len(seen) < 2 {
		t.Fatalf("expected at least 2 attempts, got %d", len(seen))
	}
	for i, body := range seen {
		if body != wantBody {
			t.Errorf("attempt %d sent body %q, want %q", i+1, body, wantBody)
		}
	}
}
