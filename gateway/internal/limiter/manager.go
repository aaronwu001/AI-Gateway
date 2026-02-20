package limiter

import (
	"sync"
)

// LimiterManager manages multiple TokenBuckets (e.g., for different IPs or API keys).
type LimiterManager struct {
	buckets map[string]*TokenBucket
	mu      sync.RWMutex
}

func NewLimiterManager() *LimiterManager {
	lm := &LimiterManager{
		buckets: make(map[string]*TokenBucket),
	}
	// Start a background goroutine to clean expired buckets every minute
	// (simplified here; complex TTL handling is not implemented yet).
	return lm
}

// GetLimiter gets or creates a limiter for the specified key.
func (lm *LimiterManager) GetLimiter(key string, rate, capacity float64) *TokenBucket {
	// 1. Fast read lock.
	lm.mu.RLock()
	bucket, exists := lm.buckets[key]
	lm.mu.RUnlock()

	if exists {
		return bucket
	}

	// 2. Write lock.
	lm.mu.Lock()
	defer lm.mu.Unlock()

	// Double-check to prevent duplicate creation during lock switch.
	if bucket, exists = lm.buckets[key]; exists {
		return bucket
	}

	// Create a new bucket.
	newBucket := NewTokenBucket(rate, capacity)
	lm.buckets[key] = newBucket
	return newBucket
}

// Allow checks whether the given key is allowed.
func (lm *LimiterManager) Allow(key string, rate, capacity float64) bool {
	bucket := lm.GetLimiter(key, rate, capacity)
	return bucket.Allow()
}
