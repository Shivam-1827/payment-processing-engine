package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Shivam-1827/payment-engine/services/webhook-service/internal/dispatcher"
	"github.com/Shivam-1827/payment-engine/services/webhook-service/internal/outbox"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	setupSignalHandler(cancel, logger)

	// 1. Connect to Ledger Database
	dbURL := getEnv("DB_URL", "postgres://ledger_admin:supersecretpassword@localhost:5432/payment_engine?sslmode=disable")
	pool, err := outbox.NewPool(ctx, dbURL)
	if err != nil {
		logger.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}

	// 2. Initialize Outbox Repo and Dispatcher
	repo := outbox.NewRepository(pool)
	
	// Mocking the merchant webhook receiver URL (e.g., https://webhook.site/...)
	mockMerchantURL := getEnv("MERCHANT_WEBHOOK_URL", "https://httpbin.org/post")
	
	worker := dispatcher.NewWorker(repo, mockMerchantURL, logger)

	// 3. Start polling every 2 seconds for batches of 50
	go worker.Start(ctx, 2*time.Second, 50)

	<-ctx.Done()
	logger.Info("Webhook service shutting down cleanly...")
	
	// Graceful shutdown with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	
	// Give worker time to finish processing
	select {
	case <-shutdownCtx.Done():
		logger.Warn("graceful shutdown timeout exceeded")
	case <-time.After(1 * time.Second):
		logger.Info("shutdown complete")
	}
	
	pool.Close()
}

func setupSignalHandler(cancel context.CancelFunc, logger *slog.Logger) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		logger.Warn("received termination signal")
		cancel()
	}()
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}