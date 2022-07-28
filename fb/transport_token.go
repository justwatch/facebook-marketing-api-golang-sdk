package fb

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"fmt"
	"net/http"
	"net/url"
)

type tokenTransport struct {
	token, clientKey string
	next             http.RoundTripper
}

func newTokenTransport(token, clientKey string, next http.RoundTripper) http.RoundTripper {
	if next == nil {
		next = http.DefaultTransport
	}

	return &tokenTransport{
		token:     token,
		clientKey: clientKey,
		next:      next,
	}
}

func (t *tokenTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	ctx := r.Context()
	u, err := url.Parse(r.URL.String())
	if err != nil {
		return nil, err
	}
	q := u.Query()
	q.Set("access_token", t.getAccessToken(ctx))
	q.Set("appsecret_proof", t.getAppSecretProof(ctx))
	u.RawQuery = q.Encode()

	rNew := *r
	rNew.URL = u

	return t.next.RoundTrip(&rNew)
}

// SetPageAccessToken adds token to the context to be used for making requests.
func SetPageAccessToken(ctx context.Context, token string) context.Context {
	if token == "" {
		return ctx
	}

	return context.WithValue(ctx, tk, token)
}

func (t *tokenTransport) getAccessToken(ctx context.Context) string {
	token, ok := ctx.Value(tk).(string)
	if ok && token != "" {
		return token
	}

	return t.token
}

type tokenKey struct{}

var tk tokenKey

func (t *tokenTransport) getAppSecretProof(ctx context.Context) string {
	h := hmac.New(sha256.New, []byte(t.clientKey))
	h.Write([]byte(t.getAccessToken(ctx)))

	return fmt.Sprintf("%x", h.Sum(nil))
}
