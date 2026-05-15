package service

import "sync"

type RequestTracker interface {
	Acquire(ruleID string)
	Release(ruleID string)
	ActiveCount(ruleID string) int64
}

func NewRequestTracker() RequestTracker {
	return &requestTracker{counts: make(map[string]int64)}
}

type requestTracker struct {
	mu     sync.Mutex
	counts map[string]int64
}

func (t *requestTracker) Acquire(ruleID string) {
	t.mu.Lock()
	t.counts[ruleID]++
	t.mu.Unlock()
}

func (t *requestTracker) Release(ruleID string) {
	t.mu.Lock()
	t.counts[ruleID]--
	if t.counts[ruleID] <= 0 {
		delete(t.counts, ruleID)
	}
	t.mu.Unlock()
}

func (t *requestTracker) ActiveCount(ruleID string) int64 {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.counts[ruleID]
}
