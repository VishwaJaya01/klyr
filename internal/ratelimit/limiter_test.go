package ratelimit

import (
	"testing"
	"time"
)

func TestLimiterAllow(t *testing.T) {
	l := NewLimiter()
	now := time.Now()

	if !l.Allow("ip:1", 1, 2, now) {
		t.Fatalf("expected first request allowed")
	}
	if !l.Allow("ip:1", 1, 2, now) {
		t.Fatalf("expected second request allowed")
	}
	if l.Allow("ip:1", 1, 2, now) {
		t.Fatalf("expected third request limited")
	}

	later := now.Add(1500 * time.Millisecond)
	if !l.Allow("ip:1", 1, 2, later) {
		t.Fatalf("expected refill to allow after time")
	}
}

func TestLimiterDifferentKeys(t *testing.T) {
	l := NewLimiter()
	now := time.Now()

	if !l.Allow("ip:1", 1, 1, now) {
		t.Fatalf("expected first key allowed")
	}
	if !l.Allow("ip:2", 1, 1, now) {
		t.Fatalf("expected second key allowed")
	}
}
