package ratelimit

import (
	"sync"
	"time"
)

const (
	lockoutSweepInterval = time.Minute
	// lockoutDefaultMaxEntries caps memory use when many unique keys are
	// seen. When full, a new key triggers an eager sweep; if that fails
	// to free space, an arbitrary entry is evicted.
	lockoutDefaultMaxEntries = 1 << 16
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
	maxEntries  int
	now         func() time.Time
	lastSweep   time.Time
}

func NewLockout(maxAttempts int, duration, window time.Duration) *Lockout {
	return &Lockout{
		entries:     make(map[string]*lockoutEntry),
		maxAttempts: maxAttempts,
		duration:    duration,
		window:      window,
		maxEntries:  lockoutDefaultMaxEntries,
		now:         time.Now,
	}
}

func (l *Lockout) IsBlocked(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	if l.shouldSweep(now) {
		l.sweep(now)
	}

	e := l.entries[key]
	if e == nil {
		return false
	}
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
	if l.shouldSweep(now) {
		l.sweep(now)
	}

	e := l.entries[key]
	if e == nil || e.stale(now, l.window) {
		if e == nil && len(l.entries) >= l.maxEntries {
			l.sweep(now)
			if len(l.entries) >= l.maxEntries {
				l.evictOne()
			}
		}
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

func (l *Lockout) shouldSweep(now time.Time) bool {
	return now.Sub(l.lastSweep) >= lockoutSweepInterval
}

func (l *Lockout) sweep(now time.Time) {
	l.lastSweep = now
	for k, e := range l.entries {
		if e.stale(now, l.window) {
			delete(l.entries, k)
		}
	}
}

func (l *Lockout) evictOne() {
	for k := range l.entries {
		delete(l.entries, k)
		return
	}
}

func (e *lockoutEntry) stale(now time.Time, window time.Duration) bool {
	if !e.lockedUntil.IsZero() {
		return !now.Before(e.lockedUntil)
	}
	return now.Sub(e.lastAttempt) >= window
}
