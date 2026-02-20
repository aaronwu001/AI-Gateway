package limiter

import (
	"sync"
	"time"
)

// TokenBucket represents a single rate-limiting bucket.
type TokenBucket struct {
	rate       float64    // Tokens generated per second.
	capacity   float64    // Maximum bucket capacity.
	tokens     float64    // Current remaining tokens.
	lastRefill time.Time  // Timestamp of the last refill.
	mu         sync.Mutex // Ensures concurrency safety.
}

func NewTokenBucket(rate, capacity float64) *TokenBucket {
	return &TokenBucket{
		rate:       rate,
		capacity:   capacity,
		tokens:     capacity, // Starts full.
		lastRefill: time.Now(),
	}
}

// Allow checks whether a request can pass (consumes 1 token).
func (tb *TokenBucket) Allow() bool {
	return tb.AllowN(1)
}

// AllowN checks whether a request can pass with N token cost.
func (tb *TokenBucket) AllowN(n float64) bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	now := time.Now()
	// 1. Calculate elapsed time.
	elapsed := now.Sub(tb.lastRefill).Seconds()

	// 2. Refill tokens (rate * elapsed time).
	tokensToAdd := elapsed * tb.rate
	tb.tokens = tb.tokens + tokensToAdd

	// 3. Cap at bucket capacity.
	if tb.tokens > tb.capacity {
		tb.tokens = tb.capacity
	}

	tb.lastRefill = now

	// 4. Check if enough tokens are available.
	if tb.tokens >= n {
		tb.tokens -= n
		return true
	}

	return false
}
