package limiter

import (
	"sync"
	"time"
)

// TokenBucket 代表單一個限流桶
type TokenBucket struct {
	rate       float64    // 每秒產生多少令牌
	capacity   float64    // 桶子最大容量
	tokens     float64    // 目前剩下的令牌
	lastRefill time.Time  // 上次補水的時間
	mu         sync.Mutex // 確保併發安全
}

func NewTokenBucket(rate, capacity float64) *TokenBucket {
	return &TokenBucket{
		rate:       rate,
		capacity:   capacity,
		tokens:     capacity, // 一開始是滿的
		lastRefill: time.Now(),
	}
}

// Allow 檢查是否允許通過 (消耗 1 個 token)
func (tb *TokenBucket) Allow() bool {
	return tb.AllowN(1)
}

// AllowN 檢查是否允許通過 N 個 token
func (tb *TokenBucket) AllowN(n float64) bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	now := time.Now()
	// 1. 計算經過時間
	elapsed := now.Sub(tb.lastRefill).Seconds()

	// 2. 補充令牌 (Rate * 時間)
	tokensToAdd := elapsed * tb.rate
	tb.tokens = tb.tokens + tokensToAdd

	// 3. 不能超過容量
	if tb.tokens > tb.capacity {
		tb.tokens = tb.capacity
	}

	tb.lastRefill = now

	// 4. 檢查是否足夠
	if tb.tokens >= n {
		tb.tokens -= n
		return true
	}

	return false
}
