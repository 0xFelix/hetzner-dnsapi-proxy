package ratelimit

import (
	"sync"
	"time"

	"golang.org/x/time/rate"
)

const (
	limiterSweepInterval = time.Minute
	// limiterDefaultMaxBuckets caps memory use when many unique keys are
	// seen (e.g. under a header-spoofing attack against a trusted proxy).
	// When full, a new key triggers an eager sweep; if that fails to free
	// space, the request is rejected.
	limiterDefaultMaxBuckets = 1 << 16
)

type bucket struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

type Limiter struct {
	mu         sync.Mutex
	buckets    map[string]*bucket
	limit      rate.Limit
	burst      int
	idle       time.Duration
	maxBuckets int
	now        func() time.Time
	lastSweep  time.Time
}

func NewLimiter(ratePerSecond float64, burst int, idle time.Duration) *Limiter {
	return &Limiter{
		buckets:    make(map[string]*bucket),
		limit:      rate.Limit(ratePerSecond),
		burst:      burst,
		idle:       idle,
		maxBuckets: limiterDefaultMaxBuckets,
		now:        time.Now,
	}
}

func (l *Limiter) Allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	if l.shouldSweep(now) {
		l.sweep(now)
	}

	b, ok := l.buckets[key]
	if !ok {
		if len(l.buckets) >= l.maxBuckets {
			l.sweep(now)
			if len(l.buckets) >= l.maxBuckets {
				return false
			}
		}
		b = &bucket{limiter: rate.NewLimiter(l.limit, l.burst)}
		l.buckets[key] = b
	}
	b.lastSeen = now
	return b.limiter.AllowN(now, 1)
}

func (l *Limiter) shouldSweep(now time.Time) bool {
	return now.Sub(l.lastSweep) >= limiterSweepInterval
}

func (l *Limiter) sweep(now time.Time) {
	l.lastSweep = now
	for k, b := range l.buckets {
		if now.Sub(b.lastSeen) > l.idle {
			delete(l.buckets, k)
		}
	}
}
