package limiter

import (
	"sync"
)

// RateLimiterManager 負責管理所有使用者的 TokenBucket
type RateLimiterManager struct {
	// key: 使用者 ID 或 IP
	// value: 指向該使用者專屬 TokenBucket 的指標
	buckets map[string]*TokenBucket

	// globalRate & globalCapacity: 全域預設的限流設定 (例如每秒 10 次)
	globalRate     float64
	globalCapacity float64

	// RWMutex (讀寫鎖): 比一般的 Mutex 更高效
	// 它允許多個 "讀取者" 同時查 Map (高併發讀取)
	// 但如果有 "寫入者" (新增使用者)，所有人都要停下來等
	mu sync.RWMutex
}

// NewRateLimiterManager 建立一個管理器
func NewRateLimiterManager(rate, capacity float64) *RateLimiterManager {
	return &RateLimiterManager{
		buckets:        make(map[string]*TokenBucket),
		globalRate:     rate,
		globalCapacity: capacity,
	}
}

// GetLimiter 根據使用者 ID 取得他的 Rate Limiter
// 如果該使用者不存在，則動態建立一個
func (m *RateLimiterManager) GetLimiter(id string) *TokenBucket {
	// 1. 快速路徑 (Fast Path): 只用 "讀鎖" 檢查是否存在
	// RLock 允許成千上萬個 Goroutine 同時進來讀，不會卡住
	m.mu.RLock()
	bucket, exists := m.buckets[id]
	m.mu.RUnlock() // 讀完立刻解鎖

	// 如果已經存在，直接回傳 (這是 99% 的情況，速度極快)
	if exists {
		return bucket
	}

	// 2. 慢速路徑 (Slow Path): 需要 "寫鎖" 來建立新使用者
	// 這裡必須用 Lock (互斥鎖)，因為我們要修改 map
	m.mu.Lock()
	defer m.mu.Unlock()

	// Double Check: 在我們拿鎖的等待期間，可能別的 Goroutine 已經幫這個人建好了
	// 所以要再檢查一次，避免重複建立
	bucket, exists = m.buckets[id]
	if exists {
		return bucket
	}

	// 真的不存在，建立新的
	newBucket := NewTokenBucket(m.globalRate, m.globalCapacity)
	m.buckets[id] = newBucket

	return newBucket
}
