package main

import (
	"context"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"

	"go-rate-limiter/internal/config"
	"go-rate-limiter/internal/limiter"
	"go-rate-limiter/internal/server"
)

// 輔助函式：優先讀取環境變數，若無則回傳 YAML 的數值
func envOrYaml(envKey string, yamlVal float64) float64 {
	if val, exists := os.LookupEnv(envKey); exists {
		if f, err := strconv.ParseFloat(val, 64); err == nil {
			return f
		}
	}
	return yamlVal
}

func main() {
	cfg, err := config.LoadConfig("config.yaml")
	if err != nil {
		log.Fatalf("❌ 無法載入設定檔: %v", err)
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       0,
	})

	ctx := context.Background()
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Printf("⚠️ Redis 未啟動或連線失敗")
	}

	// 這裡讀取環境變數 REDIS_FAILURE_MODE 以支援測試腳本切換 Fail-Open/Closed
	failureMode := cfg.Redis.FailureMode
	if envMode, exists := os.LookupEnv("REDIS_FAILURE_MODE"); exists {
		failureMode = envMode
	}

	// 初始化時帶入最終決定的 failureMode (open 或 closed)
	redisLimiter := limiter.NewRedisLimiter(rdb, failureMode)

	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})

	for _, svc := range cfg.Services {
		s := svc
		targetURL, _ := url.Parse(s.TargetURL)
		proxy := httputil.NewSingleHostReverseProxy(targetURL)

		limitCfg := server.RateLimitConfig{
			GlobalRate:     envOrYaml("LIMIT_GLOBAL_RATE", s.RateLimit.GlobalRate),
			GlobalCapacity: envOrYaml("LIMIT_GLOBAL_CAP", s.RateLimit.GlobalCapacity),
			IPRate:         envOrYaml("LIMIT_IP_RATE", s.RateLimit.IPRate),
			IPCapacity:     envOrYaml("LIMIT_IP_CAP", s.RateLimit.IPCapacity),
			UserRate:       envOrYaml("LIMIT_USER_RATE", s.RateLimit.UserRate),
			UserCapacity:   envOrYaml("LIMIT_USER_CAP", s.RateLimit.UserCapacity),
		}

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Host = targetURL.Host
			proxy.ServeHTTP(w, r)
		})

		// ✨ 關鍵修改點：傳入 s.Name 作為第三個參數
		// 這樣 middleware.go 才能建立 "global:gpt4-service" 這種獨立的 Key
		mux.Handle(s.Path, server.RateLimitMiddleware(redisLimiter, limitCfg, s.Name, handler))

		log.Printf("🚀 路由就緒: %s -> %s (Name: %s)", s.Path, s.TargetURL, s.Name)
	}

	addr := ":" + strconv.Itoa(cfg.Server.Port)
	srv := &http.Server{Addr: addr, Handler: mux}

	go func() {
		log.Printf("🌐 AI Gateway 啟動於 %s", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server 啟動失敗: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	_ = rdb.Close()
	ctxShutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	srv.Shutdown(ctxShutdown)
	log.Println("✅ 伺服器已退出")
}
