package limiter

import (
	"sync"
	"time"
)

// TokenBucket 定義了我們的令牌桶結構
// 這就像是硬體的 Register File，儲存當前的狀態
type TokenBucket struct {
	rate       float64    // 速率 (tokens/sec): 每秒補充幾個代幣
	capacity   float64    // 容量 (burst): 桶子最大能裝多少
	tokens     float64    // 當前餘額: 目前桶子裡剩幾個
	lastRefill time.Time  // 上次更新時間: 用來做 Lazy Refill 計算
	mu         sync.Mutex // 互斥鎖: 保護上述變數不被 "Race Condition" 搞壞
}

// NewTokenBucket 是一個 "Constructor" (工廠函式)
// 它回傳一個指向 TokenBucket 的指標 (*)
func NewTokenBucket(rate, capacity float64) *TokenBucket {
	return &TokenBucket{
		rate:       rate,
		capacity:   capacity,
		tokens:     capacity, // 初始化時，通常桶子是滿的，讓使用者可以直接開始用
		lastRefill: time.Now(),
	}
}

// Allow 是一個便捷方法，預設消耗 1 個 token
func (tb *TokenBucket) Allow() bool {
	return tb.AllowN(1)
}

// AllowN 是核心邏輯：嘗試消耗 n 個 tokens
// 這是 Thread-Safe 的，因為我們用了 Lock
func (tb *TokenBucket) AllowN(cost float64) bool {
	// 1. 上鎖 (Locking)
	// 這就像進廁所把門鎖起來，其他人 (其他的 Goroutine) 必須在外面排隊
	tb.mu.Lock()

	// 2. 解鎖 (Unlocking)
	// defer 關鍵字保證：不管這個函式怎麼結束 (return true/false)，
	// 在離開前一刻，一定會執行 Unlock。避免死鎖 (Deadlock)。
	defer tb.mu.Unlock()

	// 3. Lazy Refill (懶惰加幣演算法)
	now := time.Now()
	// 算出距離上次計算，過了幾秒
	elapsed := now.Sub(tb.lastRefill).Seconds()

	// 根據時間差，算出應該補充多少代幣
	// 公式: 補充量 = 時間差 * 速率
	tokensToAdd := elapsed * tb.rate

	// 如果有需要補充的
	if tokensToAdd > 0 {
		tb.tokens = tb.tokens + tokensToAdd
		// 限制不能超過最大容量 (Capacity)
		if tb.tokens > tb.capacity {
			tb.tokens = tb.capacity
		}
		// 更新 "上次補充時間"
		tb.lastRefill = now
	}

	// 4. 判斷與扣款 (Check and Consume)
	if tb.tokens >= cost {
		tb.tokens -= cost // 扣除代幣
		return true       // 放行 (Allowed)
	}

	// 餘額不足
	return false // 拒絕 (Denied)
}
