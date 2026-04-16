package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/Shivam-1827/payment-engine/services/api-gateway/internal/auth"
	"github.com/Shivam-1827/payment-engine/services/api-gateway/internal/config"
	"github.com/Shivam-1827/payment-engine/services/api-gateway/internal/idempotency"
	"github.com/Shivam-1827/payment-engine/services/api-gateway/internal/queue"
	"github.com/Shivam-1827/payment-engine/services/api-gateway/internal/transport"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/redis/go-redis/v9"
)

func main() {
	// 1. Setup structured JSON logging
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	// 2. Load Config
	cfg := config.Load()

	// 3. Initialize Redis
	rdb := redis.NewClient(&redis.Options{Addr: cfg.RedisAddr})
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		logger.Error("failed to connect to redis", "error", err)
		os.Exit(1)
	}
	defer rdb.Close()

	// 4. Dependency Injection
	idempStore := idempotency.NewRedisStore(rdb)
	authenticator := auth.NewHashedAPIKeyAuthenticator()
	kafkaBrokers := strings.Split(getEnv("KAFKA_BROKERS", "localhost:9092"), ",")
	kafkaProducer := queue.NewKafkaProducer(kafkaBrokers, getEnv("KAFKA_TOPIC", "payment_intents"), logger)
	defer kafkaProducer.Close()
	paymentHandler := transport.NewPaymentHandler(idempStore, kafkaProducer, logger)

	// 5. Router & Global Middleware Setup
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(cfg.WriteTimeout)) // Mitigate Slowloris attacks

	// 6. Mount Protected Routes
	r.Route("/v1/payments", func(r chi.Router) {
		r.Use(transport.RequireAuth(authenticator))
		r.Use(transport.EnforceIdempotencyKey) // 🛡️ Fully enforced here

		r.Post("/", paymentHandler.HandleCreatePayment)
	})

	// 7. Server Configuration
	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      r,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
	}

	// 8. Run Server Asynchronously
	go func() {
		logger.Info("API Gateway starting", "port", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server failed", "error", err)
			os.Exit(1)
		}
	}()

	// 9. Graceful Shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down API gateway gracefully...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("server forced to shutdown", "error", err)
	}
	logger.Info("API gateway exited cleanly")
}

func getEnv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	return value
}
