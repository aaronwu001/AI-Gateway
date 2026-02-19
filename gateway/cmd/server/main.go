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

// Helper: Prioritize environment variable, fallback to YAML value if not set
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
		log.Fatalf("❌ Failed to load config: %v", err)
	}

	// 1. Initialize Redis Client
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       0,
	})

	ctx := context.Background()
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Printf("⚠️ Redis not reachable (handling via FailureMode)")
	}

	// 2. Set Failure Mode (Support Env Override)
	failureMode := cfg.Redis.FailureMode
	if envMode, exists := os.LookupEnv("REDIS_FAILURE_MODE"); exists {
		failureMode = envMode
	}

	// 3. Initialize Limiters
	// Distributed (Redis)
	redisLimiter := limiter.NewRedisLimiter(rdb, failureMode)

	// Local (Memory) - Start local limiter manager
	localLimiter := limiter.NewLimiterManager()

	// 4. Setup Router
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})

	for _, svc := range cfg.Services {
		s := svc // Capture loop variable
		targetURL, _ := url.Parse(s.TargetURL)

		// Create Reverse Proxy
		proxy := httputil.NewSingleHostReverseProxy(targetURL)

		// Optimization: Ensure Host Header is correct for forwarding
		originalDirector := proxy.Director
		proxy.Director = func(req *http.Request) {
			originalDirector(req)
			req.Host = targetURL.Host
		}

		// Configure Rate Limits (Env Override)
		limitCfg := server.RateLimitConfig{
			GlobalRate:     envOrYaml("LIMIT_GLOBAL_RATE", s.RateLimit.GlobalRate),
			GlobalCapacity: envOrYaml("LIMIT_GLOBAL_CAP", s.RateLimit.GlobalCapacity),
			IPRate:         envOrYaml("LIMIT_IP_RATE", s.RateLimit.IPRate),
			IPCapacity:     envOrYaml("LIMIT_IP_CAP", s.RateLimit.IPCapacity),
			UserRate:       envOrYaml("LIMIT_USER_RATE", s.RateLimit.UserRate),
			UserCapacity:   envOrYaml("LIMIT_USER_CAP", s.RateLimit.UserCapacity),
		}

		// ✨ Chain Middleware
		// Flow: Request -> Middleware (Local -> Redis) -> Proxy -> Backend
		handler := server.RateLimitMiddleware(localLimiter, redisLimiter, limitCfg, s.Name, proxy)

		// Register Route
		mux.Handle(s.Path, handler)

		log.Printf("🚀 Route ready: %s -> %s (Service: %s)", s.Path, s.TargetURL, s.Name)
	}

	// 5. Start Server
	addr := ":" + strconv.Itoa(cfg.Server.Port)
	srv := &http.Server{Addr: addr, Handler: mux}

	go func() {
		log.Printf("🌐 AI Gateway started at %s", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("❌ Server startup failed: %v", err)
		}
	}()

	// 6. Graceful Shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	_ = rdb.Close()
	ctxShutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	srv.Shutdown(ctxShutdown)
	log.Println("✅ Server exited successfully")
}
