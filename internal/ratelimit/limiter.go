package ratelimit

import (
	"sync"
	"time"
)

type KeyType string

const (
	KeyIP     KeyType = "ip"
	KeyIPPath KeyType = "ip_path"
)

type Limiter struct {
	mu      sync.Mutex
	buckets map[string]*bucket
}

type bucket struct {
	okens   float64
	last    time.Time
	burst   float64
	perSec  float64
}

func NewLimiter() *Limiter {
	return &Limiter{buckets: make(map[string]*bucket)}
}

// Allow returns true if the request is allowed, false if rate limited.
func (l *Limiter) Allow(key string, rps float64, burst int, now time.Time) bool {
	if key == "" {
		return true
	}
	if rps <= 0 || burst <= 0 {
		return true
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	b, ok := l.buckets[key]
	if !ok {
		b = &bucket{
			tokens:  float64(burst),
			last:    now,
			burst:   float64(burst),
			perSec:  rps,
		}
		l.buckets[key] = b
	}

	if b.perSec != rps || b.burst != float64(burst) {
		b.perSec = rps
		b.burst = float64(burst)
		if b.tokens > b.burst {
			b.tokens = b.burst
		}
	}

	elapsed := now.Sub(b.last).Seconds()
	if elapsed < 0 {
		elapsed = 0
	}
	b.tokens += elapsed * b.perSec
	if b.tokens > b.burst {
		b.tokens = b.burst
	}
	b.last = now

	if b.tokens < 1 {
		return false
	}

	b.tokens -= 1
	return true
}
