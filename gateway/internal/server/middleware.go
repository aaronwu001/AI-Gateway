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

func RateLimitMiddleware(redisLimiter *limiter.RedisLimiter, config RateLimitConfig, serviceName string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// --- Global Limit ---
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

		// --- IP Limit ---
		ip := extractIP(r)
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

		// --- User Limit ---
		apiKey := r.Header.Get("X-API-Key")
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
