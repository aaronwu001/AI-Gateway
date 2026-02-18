package limiter

import (
	"context"
	_ "embed"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

//go:embed request_rate.lua
var requestRateScript string

type RedisLimiter struct {
	client      *redis.Client
	failureMode string
}

func NewRedisLimiter(client *redis.Client, failureMode string) *RedisLimiter {
	return &RedisLimiter{
		client:      client,
		failureMode: failureMode,
	}
}

func (r *RedisLimiter) AllowN(ctx context.Context, key string, rate, capacity float64, cost int) (bool, error) {
	tokensKey := fmt.Sprintf("rate_limit:%s:tokens", key)
	timestampKey := fmt.Sprintf("rate_limit:%s:ts", key)

	cmd := r.client.Eval(ctx, requestRateScript, []string{tokensKey, timestampKey},
		rate, capacity, time.Now().Unix(), cost)

	allowed, err := cmd.Bool()
	if err != nil {
		// Log 改為純英文
		log.Printf("[REDIS ERROR] Eval failed: %v", err)

		if r.failureMode == "closed" {
			log.Printf("[FAIL-CLOSED] Request blocked due to Redis failure")
			return false, err
		}

		log.Printf("[FAIL-OPEN] Request allowed despite Redis failure")
		return true, nil
	}

	return allowed, nil
}
