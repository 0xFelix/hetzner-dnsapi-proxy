package ratelimit

import (
	"sync"
	"time"
)

type lockoutEntry struct {
	count       int
	lastAttempt time.Time
	lockedUntil time.Time
}

// Lockout locks out a key for Duration after MaxAttempts failures within
// Window. A successful auth must call Reset to clear the counter.
type Lockout struct {
	mu          sync.Mutex
	entries     map[string]*lockoutEntry
	maxAttempts int
	duration    time.Duration
	window      time.Duration
	now         func() time.Time
}

func NewLockout(maxAttempts int, duration, window time.Duration) *Lockout {
	return &Lockout{
		entries:     make(map[string]*lockoutEntry),
		maxAttempts: maxAttempts,
		duration:    duration,
		window:      window,
		now:         time.Now,
	}
}

func (l *Lockout) IsBlocked(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	e := l.entries[key]
	if e == nil {
		return false
	}
	now := l.now()
	if now.Before(e.lockedUntil) {
		return true
	}
	if e.stale(now, l.window) {
		delete(l.entries, key)
	}
	return false
}

// RecordFailure records a failure for key. Returns true if this call triggered a lockout.
func (l *Lockout) RecordFailure(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	e := l.entries[key]
	if e == nil || e.stale(now, l.window) {
		e = &lockoutEntry{}
		l.entries[key] = e
	}

	e.count++
	e.lastAttempt = now
	if e.count >= l.maxAttempts && e.lockedUntil.IsZero() {
		e.lockedUntil = now.Add(l.duration)
		return true
	}
	return false
}

func (l *Lockout) Reset(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.entries, key)
}

func (e *lockoutEntry) stale(now time.Time, window time.Duration) bool {
	if !e.lockedUntil.IsZero() {
		return !now.Before(e.lockedUntil)
	}
	return now.Sub(e.lastAttempt) >= window
}
