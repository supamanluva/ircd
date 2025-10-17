package security

import (
	"testing"
	"time"
)

func TestRateLimiter(t *testing.T) {
	// Create a rate limiter: 2 messages per second, burst of 3
	rl := NewRateLimiter(2.0, 3.0)

	// Should allow burst of 3
	for i := 0; i < 3; i++ {
		if !rl.Allow() {
			t.Errorf("Expected Allow() = true for burst request %d", i+1)
		}
	}

	// 4th request should be denied (exceeded burst)
	if rl.Allow() {
		t.Error("Expected Allow() = false after exceeding burst")
	}

	// Wait for refill (0.5 seconds = 1 token at 2/sec rate)
	time.Sleep(550 * time.Millisecond)

	// Should allow 1 more request
	if !rl.Allow() {
		t.Error("Expected Allow() = true after refill")
	}

	// Should deny immediately after
	if rl.Allow() {
		t.Error("Expected Allow() = false without refill")
	}
}

func TestRateLimiterReset(t *testing.T) {
	rl := NewRateLimiter(1.0, 2.0)

	// Use up tokens
	rl.Allow()
	rl.Allow()

	// Should be denied
	if rl.Allow() {
		t.Error("Expected Allow() = false after using tokens")
	}

	// Reset
	rl.Reset()

	// Should allow again
	if !rl.Allow() {
		t.Error("Expected Allow() = true after reset")
	}
}

func TestRateLimiterRemaining(t *testing.T) {
	rl := NewRateLimiter(5.0, 10.0)

	initial := rl.Remaining()
	if initial != 10.0 {
		t.Errorf("Expected 10.0 remaining tokens, got %f", initial)
	}

	rl.Allow()
	after := rl.Remaining()
	if after != 9.0 {
		t.Errorf("Expected 9.0 remaining tokens, got %f", after)
	}
}

func BenchmarkRateLimiter(b *testing.B) {
	rl := NewRateLimiter(1000.0, 1000.0)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		rl.Allow()
	}
}
