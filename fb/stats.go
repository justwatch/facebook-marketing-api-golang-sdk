package fb

import (
	"context"
	"sync"
	"sync/atomic"
)

type ctxStatsKey struct{}

var statsKey ctxStatsKey

// StatsContainer contains a set of stats.
type StatsContainer struct {
	lock  *sync.RWMutex
	stats map[string]*Stat
}

// NewStatsContainer initializes a new StatsContainer.
func NewStatsContainer() *StatsContainer {
	return &StatsContainer{
		lock:  &sync.RWMutex{},
		stats: map[string]*Stat{},
	}
}

// AddStats adds new stats to the StatsContainer.
func (sc *StatsContainer) AddStats(id string) *Stat {
	s := &Stat{total: 100}

	sc.lock.Lock()
	defer sc.lock.Unlock()
	_, ok := sc.stats[id]
	if ok {
		return nil
	}
	sc.stats[id] = s

	return s
}

// RemoveStats removes the stats from the stats container.
func (sc *StatsContainer) RemoveStats(id string) {
	sc.lock.Lock()
	defer sc.lock.Unlock()
	delete(sc.stats, id)
}

// Stats returns a map of all insights stats.
func (sc *StatsContainer) Stats() map[string]InsightsStatus {
	m := make(map[string]InsightsStatus, len(sc.stats))

	sc.lock.RLock()
	defer sc.lock.RUnlock()
	for k, v := range sc.stats {
		m[k] = v.clone()
	}

	return m
}

// InsightsStatus contains information about the progress of a single insights report.
type InsightsStatus struct {
	IsCreated bool   `json:"is_created"`
	Current   uint64 `json:"current"`
	Total     uint64 `json:"total"`
}

// Stat contains stat information about a list job.
type Stat struct {
	l         sync.RWMutex
	isCreated bool
	current   uint64
	total     uint64
}

// AddToContext adds the stat container to the given context.
func (s *Stat) AddToContext(ctx context.Context) context.Context {
	return context.WithValue(ctx, statsKey, s)
}

// StatFromContext tries to retrieve the stat from the context.
func StatFromContext(ctx context.Context) *Stat {
	s := ctx.Value(statsKey)
	if s == nil {
		return nil
	}
	stat, ok := s.(*Stat)
	if ok {
		return stat
	}

	return nil
}

// SetCreated sets created to true.
func (s *Stat) SetCreated() {
	s.l.Lock()
	defer s.l.Unlock()
	s.isCreated = true
	s.current = 0
	s.total = 0
}

// SetProgress sets the current and total values.
func (s *Stat) SetProgress(current, total uint64) {
	s.l.Lock()
	defer s.l.Unlock()
	s.current = current
	s.total = total
}

// Add adds delta to the current value.
func (s *Stat) Add(delta uint64) {
	atomic.AddUint64(&s.current, delta)
}

func (s *Stat) clone() InsightsStatus {
	s.l.RLock()
	defer s.l.RUnlock()

	return InsightsStatus{
		IsCreated: s.isCreated,
		Current:   s.current,
		Total:     s.total,
	}
}
