package server

import (
	"log"
	"net"
	"net/http"
	"strings"

	"go-rate-limiter/internal/limiter"
)

type RateLimitConfig struct {
	GlobalRate     float64
	GlobalCapacity float64
	IPRate         float64
	IPCapacity     float64
	UserRate       float64
	UserCapacity   float64
}

// RateLimitMiddleware 整合了 Local (記憶體) 與 Redis (分散式) 的雙層限流
func RateLimitMiddleware(localLimiter *limiter.LimiterManager, redisLimiter *limiter.RedisLimiter, config RateLimitConfig, serviceName string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// ---------------------------------------------------------
		// 1. 準備資料 (IP & API Key)
		// ---------------------------------------------------------
		ip := extractIP(r)
		apiKey := r.Header.Get("X-API-Key")

		// ---------------------------------------------------------
		// 2. 第一道防線：Local Limiter (記憶體檢查，速度最快)
		// ---------------------------------------------------------

		// 2.1 Local IP Check
		// 使用相同的 Rate/Capacity 配置，或者你可以給 Local 稍微寬鬆一點的值
		if !localLimiter.Allow("ip:"+ip, config.IPRate, config.IPCapacity) {
			w.Header().Set("X-RateLimit-Type", "Local-IP")
			log.Printf("[LIMIT] Local IP limit exceeded: %s", ip)
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}

		// 2.2 Local User Check (如果有 API Key 才檢查)
		if apiKey != "" {
			if !localLimiter.Allow("user:"+apiKey, config.UserRate, config.UserCapacity) {
				w.Header().Set("X-RateLimit-Type", "Local-User")
				log.Printf("[LIMIT] Local User limit exceeded: %s", apiKey)
				w.WriteHeader(http.StatusTooManyRequests)
				return
			}
		}

		// ---------------------------------------------------------
		// 3. 第二道防線：Redis Limiter (分散式檢查，準確度最高)
		// ---------------------------------------------------------

		// 3.1 Global Limit (Redis)
		globalKey := "global:" + serviceName
		allowed, err := redisLimiter.AllowN(ctx, globalKey, config.GlobalRate, config.GlobalCapacity, 1)
		if !allowed {
			if err != nil {
				w.Header().Set("X-RateLimit-Type", "Redis-Error")
				log.Printf("[FAIL-CLOSED] Redis error on Global check: %v", err)
			} else {
				w.Header().Set("X-RateLimit-Type", "Global")
				log.Printf("[LIMIT] Global limit exceeded for %s", serviceName)
			}
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte(`{"error": "Service busy"}`))
			return
		}

		// 3.2 IP Limit (Redis)
		ipKey := "ip:" + ip
		allowed, err = redisLimiter.AllowN(ctx, ipKey, config.IPRate, config.IPCapacity, 1)
		if !allowed {
			if err != nil {
				w.Header().Set("X-RateLimit-Type", "Redis-Error")
				log.Printf("[FAIL-CLOSED] Redis error on IP check: %v", err)
			} else {
				w.Header().Set("X-RateLimit-Type", "IP")
				log.Printf("[LIMIT] IP limit exceeded: %s", ip)
			}
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}

		// 3.3 User Limit (Redis)
		if apiKey == "" {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error": "Missing API Key"}`))
			return
		}

		userKey := "user:" + apiKey
		allowed, err = redisLimiter.AllowN(ctx, userKey, config.UserRate, config.UserCapacity, 1)
		if !allowed {
			if err != nil {
				w.Header().Set("X-RateLimit-Type", "Redis-Error")
				log.Printf("[FAIL-CLOSED] Redis error on User check: %v", err)
			} else {
				w.Header().Set("X-RateLimit-Type", "User")
				log.Printf("[LIMIT] User quota exceeded: %s", apiKey)
			}
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}

		// ---------------------------------------------------------
		// 4. 全部通過，放行
		// ---------------------------------------------------------
		next.ServeHTTP(w, r)
	})
}

func extractIP(r *http.Request) string {
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		return strings.Split(forwarded, ",")[0]
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}
