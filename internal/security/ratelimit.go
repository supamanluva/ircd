package security

import (
	"sync"
	"time"
)

// RateLimiter implements a token bucket rate limiter
type RateLimiter struct {
	tokens      float64
	maxTokens   float64
	refillRate  float64 // tokens per second
	lastRefill  time.Time
	mu          sync.Mutex
}

// NewRateLimiter creates a new rate limiter
// rate: messages per second
// burst: maximum burst capacity
func NewRateLimiter(rate, burst float64) *RateLimiter {
	return &RateLimiter{
		tokens:     burst,
		maxTokens:  burst,
		refillRate: rate,
		lastRefill: time.Now(),
	}
}

// Allow checks if an action is allowed under the rate limit
func (rl *RateLimiter) Allow() bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	// Refill tokens based on time elapsed
	now := time.Now()
	elapsed := now.Sub(rl.lastRefill).Seconds()
	rl.tokens += elapsed * rl.refillRate

	// Cap at max tokens
	if rl.tokens > rl.maxTokens {
		rl.tokens = rl.maxTokens
	}

	rl.lastRefill = now

	// Check if we have tokens available
	if rl.tokens >= 1.0 {
		rl.tokens -= 1.0
		return true
	}

	return false
}

// Remaining returns the number of tokens remaining
func (rl *RateLimiter) Remaining() float64 {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	return rl.tokens
}

// Reset resets the rate limiter to full capacity
func (rl *RateLimiter) Reset() {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	rl.tokens = rl.maxTokens
	rl.lastRefill = time.Now()
}
