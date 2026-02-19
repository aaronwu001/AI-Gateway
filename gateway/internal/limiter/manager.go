package limiter

import (
	"sync"
)

// LimiterManager 管理多個 TokenBucket (例如針對不同 IP 或 API Key)
type LimiterManager struct {
	buckets map[string]*TokenBucket
	mu      sync.RWMutex
}

func NewLimiterManager() *LimiterManager {
	lm := &LimiterManager{
		buckets: make(map[string]*TokenBucket),
	}
	// 啟動一個背景協程，每分鐘清理一次過期的桶子 (這裡簡化處理，暫不實作複雜的 TTL)
	return lm
}

// GetLimiter 獲取或創建一個指定 key 的限流器
func (lm *LimiterManager) GetLimiter(key string, rate, capacity float64) *TokenBucket {
	// 1. 快速讀取鎖 (Read Lock)
	lm.mu.RLock()
	bucket, exists := lm.buckets[key]
	lm.mu.RUnlock()

	if exists {
		return bucket
	}

	// 2. 寫入鎖 (Write Lock)
	lm.mu.Lock()
	defer lm.mu.Unlock()

	// 再次檢查 (Double Check Locking)，防止在鎖切換瞬間被建立
	if bucket, exists = lm.buckets[key]; exists {
		return bucket
	}

	// 創建新的
	newBucket := NewTokenBucket(rate, capacity)
	lm.buckets[key] = newBucket
	return newBucket
}

// Allow 直接檢查某個 Key 是否通過
func (lm *LimiterManager) Allow(key string, rate, capacity float64) bool {
	bucket := lm.GetLimiter(key, rate, capacity)
	return bucket.Allow()
}
