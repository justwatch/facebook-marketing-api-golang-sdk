package fb

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

// fakeSleep records sleep calls without actually sleeping.
type fakeSleep struct {
	total atomic.Int64 // nanoseconds
}

func (f *fakeSleep) sleep(_ context.Context, d time.Duration) {
	f.total.Add(int64(d))
}

func (f *fakeSleep) totalDuration() time.Duration {
	return time.Duration(f.total.Load())
}

// --- IsRateLimited ---

func TestIsRateLimited(t *testing.T) {
	cases := []struct {
		code    uint64
		subcode uint64
		want    bool
	}{
		{4, 0, true},
		{17, 0, true},
		{17, 2446079, true},
		{32, 0, true},
		{613, 0, true},
		{613, 5044001, true},
		{613, 1487742, true},
		{80000, 0, true},
		{80004, 0, true},
		{80014, 0, true},
		// not rate-limit errors
		{100, 33, false}, // not found
		{1, 0, false},    // reduce data
		{200, 0, false},  // permission error
	}

	for _, c := range cases {
		err := &Error{Code: c.code, ErrorSubcode: c.subcode}
		got := IsRateLimited(err)
		if got != c.want {
			t.Errorf("IsRateLimited(code=%d, subcode=%d) = %v, want %v", c.code, c.subcode, got, c.want)
		}
	}

	// non-fb errors
	if IsRateLimited(fmt.Errorf("some error")) {
		t.Error("expected false for non-fb error")
	}
	if IsRateLimited(nil) {
		t.Error("expected false for nil error")
	}
}

// --- Header parsing ---

func TestUpdateFromResponse_AppUsage(t *testing.T) {
	state := newRateLimitState(defaultRateLimitConfig())

	resp := &http.Response{
		Header: http.Header{
			"X-App-Usage": []string{`{"call_count":85,"total_cputime":20,"total_time":10}`},
		},
	}
	state.updateFromResponse(resp)

	state.mu.Lock()
	pct := state.maxUsagePct
	state.mu.Unlock()

	if pct != 85 {
		t.Errorf("expected maxUsagePct=85, got %d", pct)
	}
}

func TestUpdateFromResponse_AdAccountUsage(t *testing.T) {
	state := newRateLimitState(defaultRateLimitConfig())

	resp := &http.Response{
		Header: http.Header{
			"X-Ad-Account-Usage": []string{`{"acc_id_util_pct":92.5,"reset_time_duration":120}`},
		},
	}
	state.updateFromResponse(resp)

	state.mu.Lock()
	pct := state.maxUsagePct
	dur := state.blockDuration
	state.mu.Unlock()

	if pct != 92 {
		t.Errorf("expected maxUsagePct=92, got %d", pct)
	}
	if dur != 120*time.Second {
		t.Errorf("expected blockDuration=120s, got %v", dur)
	}
}

func TestUpdateFromResponse_BUCUsage(t *testing.T) {
	state := newRateLimitState(defaultRateLimitConfig())

	bucHeader := map[string][]bucUsageEntry{
		"66782684": {
			{Type: "ads_management", CallCount: 95, TotalCputime: 20, TotalTime: 20, EstimatedTimeToRegainAccess: 3},
		},
	}
	raw, _ := json.Marshal(bucHeader)
	resp := &http.Response{
		Header: http.Header{
			"X-Business-Use-Case-Usage": []string{string(raw)},
		},
	}
	state.updateFromResponse(resp)

	state.mu.Lock()
	pct := state.maxUsagePct
	dur := state.blockDuration
	state.mu.Unlock()

	if pct != 95 {
		t.Errorf("expected maxUsagePct=95, got %d", pct)
	}
	if dur != 3*time.Minute {
		t.Errorf("expected blockDuration=3m, got %v", dur)
	}
}

func TestUpdateFromResponse_Nil(t *testing.T) {
	state := newRateLimitState(defaultRateLimitConfig())
	// must not panic
	state.updateFromResponse(nil)
}

// --- Proactive delay (waitIfNeeded) ---

func TestWaitIfNeeded_BelowHighWatermark(t *testing.T) {
	fs := &fakeSleep{}
	state := newRateLimitState(defaultRateLimitConfig())
	state.sleep = fs.sleep

	state.mu.Lock()
	state.maxUsagePct = 70
	state.mu.Unlock()

	state.waitIfNeeded(context.Background())

	if fs.totalDuration() != 0 {
		t.Errorf("expected no sleep below highwatermark, got %v", fs.totalDuration())
	}
}

func TestWaitIfNeeded_AboveHighWatermark(t *testing.T) {
	fs := &fakeSleep{}
	state := newRateLimitState(defaultRateLimitConfig())
	state.sleep = fs.sleep

	state.mu.Lock()
	state.maxUsagePct = 85
	state.blockDuration = 0
	state.mu.Unlock()

	state.waitIfNeeded(context.Background())

	// at 85% with watermark=80, blockAt=100 → fraction=0.25 → 25% of 5s = 1.25s
	expected := 1250 * time.Millisecond
	if fs.totalDuration() != expected {
		t.Errorf("expected sleep=%v, got %v", expected, fs.totalDuration())
	}
}

func TestWaitIfNeeded_AtBlockAt(t *testing.T) {
	fs := &fakeSleep{}
	state := newRateLimitState(defaultRateLimitConfig())
	state.sleep = fs.sleep

	state.mu.Lock()
	state.maxUsagePct = 100
	state.blockDuration = 2 * time.Minute
	state.mu.Unlock()

	state.waitIfNeeded(context.Background())

	if fs.totalDuration() != 2*time.Minute {
		t.Errorf("expected sleep=2m, got %v", fs.totalDuration())
	}
}

func TestWaitIfNeeded_AtBlockAt_FallbackDuration(t *testing.T) {
	fs := &fakeSleep{}
	state := newRateLimitState(defaultRateLimitConfig())
	state.sleep = fs.sleep

	state.mu.Lock()
	state.maxUsagePct = 100
	state.blockDuration = 0 // no header info
	state.mu.Unlock()

	state.waitIfNeeded(context.Background())

	// fallback is 1s
	if fs.totalDuration() != time.Second {
		t.Errorf("expected fallback sleep=1s, got %v", fs.totalDuration())
	}
}

func TestWaitIfNeeded_Disabled(t *testing.T) {
	fs := &fakeSleep{}
	cfg := defaultRateLimitConfig()
	cfg.Enabled = false
	state := newRateLimitState(cfg)
	state.sleep = fs.sleep

	state.mu.Lock()
	state.maxUsagePct = 100
	state.blockDuration = 5 * time.Minute
	state.mu.Unlock()

	state.waitIfNeeded(context.Background())

	if fs.totalDuration() != 0 {
		t.Errorf("expected no sleep when disabled, got %v", fs.totalDuration())
	}
}

// --- Transport integration ---

func TestRateLimitTransport_ProactiveDelay(t *testing.T) {
	var requestCount int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		if requestCount == 1 {
			w.Header().Set("X-App-Usage", `{"call_count":90,"total_cputime":10,"total_time":10}`)
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{}`)
	}))
	defer srv.Close()

	fs := &fakeSleep{}
	state := newRateLimitState(defaultRateLimitConfig())
	state.sleep = fs.sleep

	transport := newRateLimitTransport(nil, state, nil)
	client := &http.Client{Transport: transport}

	// First request — sets state
	resp, err := client.Get(srv.URL)
	if err != nil {
		t.Fatalf("first request failed: %v", err)
	}
	resp.Body.Close()

	// Second request — should trigger proactive delay (90% > 80% watermark)
	resp, err = client.Get(srv.URL)
	if err != nil {
		t.Fatalf("second request failed: %v", err)
	}
	resp.Body.Close()

	// 90% with watermark=80, blockAt=100 → fraction=0.5 → 50% of 5s = 2.5s
	expected := 2500 * time.Millisecond
	if fs.totalDuration() != expected {
		t.Errorf("expected proactive sleep=%v, got %v", expected, fs.totalDuration())
	}
}

func TestRateLimitTransport_NoDelayWhenDisabled(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-App-Usage", `{"call_count":100,"total_cputime":100,"total_time":100}`)
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{}`)
	}))
	defer srv.Close()

	fs := &fakeSleep{}
	cfg := defaultRateLimitConfig()
	cfg.Enabled = false
	state := newRateLimitState(cfg)
	state.sleep = fs.sleep

	transport := newRateLimitTransport(nil, state, nil)
	client := &http.Client{Transport: transport}

	for i := 0; i < 3; i++ {
		resp, err := client.Get(srv.URL)
		if err != nil {
			t.Fatalf("request %d failed: %v", i, err)
		}
		resp.Body.Close()
	}

	if fs.totalDuration() != 0 {
		t.Errorf("expected no sleep when disabled, got %v", fs.totalDuration())
	}
}

// --- Retry on rate-limit error ---

func TestRetryTransport_RetriesOnRateLimit(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount < 3 {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprint(w, `{"error":{"message":"too many calls","type":"OAuthException","code":613,"error_subcode":5044001}}`)
			return
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{}`)
	}))
	defer srv.Close()

	fs := &fakeSleep{}
	cfg := defaultRateLimitConfig()
	cfg.Enabled = true
	state := newRateLimitState(cfg)
	state.sleep = fs.sleep
	// simulate a header-reported reset time so the throttle gate has something to use
	state.mu.Lock()
	state.blockDuration = 10 * time.Millisecond
	state.mu.Unlock()

	transport := newRetryTransport(nil, state)
	client := &http.Client{Transport: transport}

	resp, err := client.Get(srv.URL)
	if err != nil {
		t.Fatalf("expected success after retries, got: %v", err)
	}
	resp.Body.Close()

	if callCount != 3 {
		t.Errorf("expected 3 server calls (2 rate-limited + 1 success), got %d", callCount)
	}
}

// --- Shared throttle gate ---

func TestThrottleGate_UsesHeaderDuration(t *testing.T) {
	fs := &fakeSleep{}
	state := newRateLimitState(defaultRateLimitConfig())
	state.sleep = fs.sleep
	state.rnd = func() float64 { return 0 } // no jitter
	base := time.Now()
	state.now = func() time.Time { return base }

	state.mu.Lock()
	state.blockDuration = 2 * time.Minute
	state.mu.Unlock()

	state.registerThrottle()
	state.waitForGate(context.Background())

	if fs.totalDuration() != 2*time.Minute {
		t.Errorf("expected gate wait=2m, got %v", fs.totalDuration())
	}
}

func TestThrottleGate_FallsBackToFloor(t *testing.T) {
	fs := &fakeSleep{}
	state := newRateLimitState(defaultRateLimitConfig())
	state.sleep = fs.sleep
	state.rnd = func() float64 { return 0 } // no jitter
	base := time.Now()
	state.now = func() time.Time { return base }

	// no header reset info → blockDuration stays 0
	state.registerThrottle()
	state.waitForGate(context.Background())

	if fs.totalDuration() != minThrottleWait {
		t.Errorf("expected fallback gate wait=%v, got %v", minThrottleWait, fs.totalDuration())
	}
}

func TestThrottleGate_AppliesJitter(t *testing.T) {
	fs := &fakeSleep{}
	state := newRateLimitState(defaultRateLimitConfig())
	state.sleep = fs.sleep
	state.rnd = func() float64 { return 1.0 } // max jitter
	base := time.Now()
	state.now = func() time.Time { return base }

	state.mu.Lock()
	state.blockDuration = 10 * time.Second
	state.mu.Unlock()

	state.registerThrottle()
	state.waitForGate(context.Background())

	// remaining 10s + jitter (1.0 * 10s / 2 = 5s) = 15s
	if fs.totalDuration() != 15*time.Second {
		t.Errorf("expected gate wait with jitter=15s, got %v", fs.totalDuration())
	}
}

// One throttle response must make every concurrent caller wait, not just the
// request that was throttled.
func TestThrottleGate_CoordinatesConcurrentCallers(t *testing.T) {
	fs := &fakeSleep{}
	state := newRateLimitState(defaultRateLimitConfig())
	state.sleep = fs.sleep
	state.rnd = func() float64 { return 0 } // no jitter
	base := time.Now()
	state.now = func() time.Time { return base }

	state.mu.Lock()
	state.blockDuration = 30 * time.Second
	state.mu.Unlock()

	state.registerThrottle()

	// Two other callers both observe the shared gate and wait.
	state.waitForGate(context.Background())
	state.waitForGate(context.Background())

	if fs.totalDuration() != 60*time.Second {
		t.Errorf("expected two callers to each wait 30s (total 60s), got %v", fs.totalDuration())
	}
}

func TestThrottleGate_NoWaitWhenUnset(t *testing.T) {
	fs := &fakeSleep{}
	state := newRateLimitState(defaultRateLimitConfig())
	state.sleep = fs.sleep

	state.waitForGate(context.Background())

	if fs.totalDuration() != 0 {
		t.Errorf("expected no wait with empty gate, got %v", fs.totalDuration())
	}
}

func TestThrottleGate_DisabledSkipsWait(t *testing.T) {
	fs := &fakeSleep{}
	cfg := defaultRateLimitConfig()
	cfg.Enabled = false
	state := newRateLimitState(cfg)
	state.sleep = fs.sleep
	base := time.Now()
	state.now = func() time.Time { return base }

	state.registerThrottle()
	state.waitForGate(context.Background())

	if fs.totalDuration() != 0 {
		t.Errorf("expected no gate wait when disabled, got %v", fs.totalDuration())
	}
}

// Meta returns a transient code=2 for bulk writes that exceed the account
// write limit. It is not in IsRateLimited, so it must still be retried via the
// is_transient flag.
func TestRetryTransport_RetriesOnTransientCode2(t *testing.T) {
	if testing.Short() {
		t.Skip("retry backoff makes this test slow")
	}

	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount < 2 {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, `{"error":{"message":"An unexpected error has occurred.","type":"OAuthException","code":2,"error_subcode":0,"is_transient":true}}`)
			return
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{}`)
	}))
	defer srv.Close()

	fs := &fakeSleep{}
	state := newRateLimitState(defaultRateLimitConfig())
	state.sleep = fs.sleep

	transport := newRetryTransport(nil, state)
	client := &http.Client{Transport: transport}

	resp, err := client.Get(srv.URL)
	if err != nil {
		t.Fatalf("expected success after retry, got: %v", err)
	}
	resp.Body.Close()

	if callCount != 2 {
		t.Errorf("expected 2 server calls (1 transient + 1 success), got %d", callCount)
	}
	if fs.totalDuration() == 0 {
		t.Error("expected the throttle gate to apply a non-zero wait for code=2")
	}
}

func TestRetryTransport_DoesNotRetryAuthError(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, `{"error":{"message":"Invalid token","type":"OAuthException","code":190}}`)
	}))
	defer srv.Close()

	state := newRateLimitState(defaultRateLimitConfig())
	transport := newRetryTransport(nil, state)
	client := &http.Client{Transport: transport}

	resp, err := client.Get(srv.URL)
	if err != nil {
		t.Fatalf("unexpected transport error: %v", err)
	}
	resp.Body.Close()

	if callCount != 1 {
		t.Errorf("expected 1 call for non-retryable 4xx, got %d", callCount)
	}
}
