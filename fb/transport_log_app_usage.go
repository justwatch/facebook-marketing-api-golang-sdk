package fb

import (
	"encoding/json"
	"net/http"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
)

type xAppUsage struct {
	CallCount    int `json:"call_count"`
	TotalCputime int `json:"total_cputime"`
	TotalTime    int `json:"total_time"`
}

type logAppUsageTransport struct {
	l    log.Logger
	next http.RoundTripper
}

func newLogAppUsageTransport(l log.Logger, next http.RoundTripper) http.RoundTripper {
	if l == nil {
		l = log.NewNopLogger()
	}
	if next == nil {
		next = http.DefaultTransport
	}

	return &logAppUsageTransport{
		l:    l,
		next: next,
	}
}

func (t *logAppUsageTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	resp, err := t.next.RoundTrip(r)

	if resp != nil {
		xau := resp.Header.Get("x-app-usage")
		if xau == "" {
			return resp, err
		}

		xAppUsage := &xAppUsage{}
		err := json.Unmarshal([]byte(xau), xAppUsage)
		if err != nil {
			_ = level.Error(t.l).Log("msg", "encountered an error when unmarshalling into *xAppUsage", "x-app-usage", xau, "err", err)
		}

		if xAppUsage.CallCount > 0 || xAppUsage.TotalCputime > 0 || xAppUsage.TotalTime > 0 {
			_ = level.Info(t.l).Log("msg", "got x-app-usage", "call_count", xAppUsage.CallCount, "total_cputime", xAppUsage.TotalCputime, "total_time", xAppUsage.TotalTime)
		}
	}

	return resp, err
}
