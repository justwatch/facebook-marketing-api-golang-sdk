package fb

import (
	"context"
	"encoding/json"
	"math/rand"
	"net/http"
	"sync"
	"time"
)

// minThrottleWait is the shared cool-down applied after a throttle response when
// Meta does not report a reset window (e.g. transient code=2 errors). It gives
// the account quiet time to recover instead of retrying with no real pause.
const minThrottleWait = 5 * time.Second

// RateLimitConfig controls the header-based throttling behaviour of the client.
type RateLimitConfig struct {
	// Enabled turns header-based throttling on or off. Default: true.
	Enabled bool
	// HighWatermark is the usage percentage at which proactive linear delays begin. Default: 80.
	HighWatermark int
	// BlockAt is the usage percentage at which requests are held until the reset window passes. Default: 100.
	BlockAt int
	// MaxProactiveDelay caps the linear delay applied between HighWatermark and BlockAt. Default: 5s.
	MaxProactiveDelay time.Duration
}

func defaultRateLimitConfig() RateLimitConfig {
	return RateLimitConfig{
		Enabled:           true,
		HighWatermark:     80,
		BlockAt:           100,
		MaxProactiveDelay: 5 * time.Second,
	}
}

// bucUsageEntry maps to one element inside x-business-use-case-usage.
type bucUsageEntry struct {
	Type                       string `json:"type"`
	CallCount                  int    `json:"call_count"`
	TotalCputime               int    `json:"total_cputime"`
	TotalTime                  int    `json:"total_time"`
	EstimatedTimeToRegainAccess int   `json:"estimated_time_to_regain_access"` // minutes
}

// appUsageHeader maps to x-app-usage.
type appUsageHeader struct {
	CallCount    int `json:"call_count"`
	TotalCputime int `json:"total_cputime"`
	TotalTime    int `json:"total_time"`
}

// adAccountUsageHeader maps to x-ad-account-usage.
type adAccountUsageHeader struct {
	AccIDUtilPct      float64 `json:"acc_id_util_pct"`
	ResetTimeDuration int     `json:"reset_time_duration"` // seconds
}

// rateLimitState holds the latest aggregated usage info from response headers.
// It is shared between rateLimitTransport and retryTransport.
type rateLimitState struct {
	cfg RateLimitConfig

	mu            sync.Mutex
	maxUsagePct   int           // max across all header fields in the last response
	blockDuration time.Duration // time to wait when at BlockAt
	pauseUntil    time.Time     // shared cool-down gate respected by all callers

	sleep func(context.Context, time.Duration) // injectable for tests
	now   func() time.Time                     // injectable for tests
	rnd   func() float64                       // injectable for tests; returns [0,1)
}

func newRateLimitState(cfg RateLimitConfig) *rateLimitState {
	return &rateLimitState{
		cfg:   cfg,
		sleep: sleepWithContext,
		now:   time.Now,
		rnd:   rand.Float64,
	}
}

func sleepWithContext(ctx context.Context, d time.Duration) {
	if d <= 0 {
		return
	}
	select {
	case <-time.After(d):
	case <-ctx.Done():
	}
}

// updateFromResponse parses rate-limit headers from resp and updates state.
// Safe to call with a nil resp.
func (s *rateLimitState) updateFromResponse(resp *http.Response) {
	if resp == nil {
		return
	}

	maxPct := 0
	blockDur := time.Duration(0)

	// x-app-usage
	if raw := resp.Header.Get("x-app-usage"); raw != "" {
		var h appUsageHeader
		if err := json.Unmarshal([]byte(raw), &h); err == nil {
			maxPct = maxInt(maxPct, h.CallCount, h.TotalCputime, h.TotalTime)
		}
	}

	// x-ad-account-usage
	if raw := resp.Header.Get("x-ad-account-usage"); raw != "" {
		var h adAccountUsageHeader
		if err := json.Unmarshal([]byte(raw), &h); err == nil {
			maxPct = maxInt(maxPct, int(h.AccIDUtilPct))
			if h.ResetTimeDuration > 0 {
				candidate := time.Duration(h.ResetTimeDuration) * time.Second
				if candidate > blockDur {
					blockDur = candidate
				}
			}
		}
	}

	// x-business-use-case-usage — map of business-object-id → []entry
	if raw := resp.Header.Get("x-business-use-case-usage"); raw != "" {
		var bucMap map[string][]bucUsageEntry
		if err := json.Unmarshal([]byte(raw), &bucMap); err == nil {
			for _, entries := range bucMap {
				for _, e := range entries {
					maxPct = maxInt(maxPct, e.CallCount, e.TotalCputime, e.TotalTime)
					if e.EstimatedTimeToRegainAccess > 0 {
						candidate := time.Duration(e.EstimatedTimeToRegainAccess) * time.Minute
						if candidate > blockDur {
							blockDur = candidate
						}
					}
				}
			}
		}
	}

	s.mu.Lock()
	s.maxUsagePct = maxPct
	s.blockDuration = blockDur
	s.mu.Unlock()
}

// waitIfNeeded sleeps before sending a request if usage is above HighWatermark.
// Returns early if ctx is cancelled.
func (s *rateLimitState) waitIfNeeded(ctx context.Context) {
	if !s.cfg.Enabled {
		return
	}

	s.mu.Lock()
	pct := s.maxUsagePct
	dur := s.blockDuration
	s.mu.Unlock()

	if pct < s.cfg.HighWatermark {
		return
	}

	var delay time.Duration
	if pct >= s.cfg.BlockAt {
		if dur <= 0 {
			dur = time.Second // minimal fallback if header had no reset time
		}
		delay = dur
	} else {
		// linear scale between HighWatermark and BlockAt
		fraction := float64(pct-s.cfg.HighWatermark) / float64(s.cfg.BlockAt-s.cfg.HighWatermark)
		delay = time.Duration(fraction * float64(s.cfg.MaxProactiveDelay))
	}

	s.sleep(ctx, delay)
}

// registerThrottle opens (or extends) the shared cool-down gate after a throttle
// response. It uses the header-reported reset window when available, otherwise a
// minimum floor, so transient code=2 throttles still get a real pause.
func (s *rateLimitState) registerThrottle() {
	s.mu.Lock()
	defer s.mu.Unlock()

	dur := s.blockDuration
	if dur <= 0 {
		dur = minThrottleWait
	}

	until := s.now().Add(dur)
	if until.After(s.pauseUntil) {
		s.pauseUntil = until
	}
}

// waitForGate blocks until the shared cool-down window has passed, adding
// per-caller jitter so concurrent callers don't all resume at the same instant
// and immediately re-throttle the account.
func (s *rateLimitState) waitForGate(ctx context.Context) {
	if !s.cfg.Enabled {
		return
	}

	s.mu.Lock()
	until := s.pauseUntil
	s.mu.Unlock()

	if until.IsZero() {
		return
	}

	remaining := until.Sub(s.now())
	if remaining <= 0 {
		return
	}

	s.sleep(ctx, remaining+s.jitter(remaining))
}

// jitter returns a random duration in [0, d/2) used to spread out concurrent
// callers resuming from the shared gate.
func (s *rateLimitState) jitter(d time.Duration) time.Duration {
	if d <= 0 {
		return 0
	}

	return time.Duration(s.rnd() * float64(d) / 2)
}

func maxInt(vals ...int) int {
	m := 0
	for _, v := range vals {
		if v > m {
			m = v
		}
	}
	return m
}
