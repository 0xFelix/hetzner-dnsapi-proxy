package ratelimit

import (
	"sync"
	"time"

	"golang.org/x/time/rate"
)

const limiterSweepInterval = time.Minute

type bucket struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

type Limiter struct {
	mu        sync.Mutex
	buckets   map[string]*bucket
	limit     rate.Limit
	burst     int
	idle      time.Duration
	now       func() time.Time
	lastSweep time.Time
}

func NewLimiter(ratePerSecond float64, burst int, idle time.Duration) *Limiter {
	return &Limiter{
		buckets: make(map[string]*bucket),
		limit:   rate.Limit(ratePerSecond),
		burst:   burst,
		idle:    idle,
		now:     time.Now,
	}
}

func (l *Limiter) Allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	l.maybeSweep(now)

	b, ok := l.buckets[key]
	if !ok {
		b = &bucket{limiter: rate.NewLimiter(l.limit, l.burst)}
		l.buckets[key] = b
	}
	b.lastSeen = now
	return b.limiter.AllowN(now, 1)
}

func (l *Limiter) maybeSweep(now time.Time) {
	if now.Sub(l.lastSweep) < limiterSweepInterval {
		return
	}
	l.lastSweep = now
	for k, b := range l.buckets {
		if now.Sub(b.lastSeen) > l.idle {
			delete(l.buckets, k)
		}
	}
}
