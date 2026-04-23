package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/Shivam-1827/payment-engine/services/orchestrator/internal/bank"
	"github.com/Shivam-1827/payment-engine/services/orchestrator/internal/db"
	"github.com/Shivam-1827/payment-engine/services/orchestrator/internal/ledger"
	"github.com/Shivam-1827/payment-engine/services/orchestrator/internal/worker"
)

func main() {
	// 1. Setup Logging
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	setupSignalHandler(cancel, logger)

	// 2. Setup Database (The Vault)
	dbURL := getEnv("DB_URL", "postgres://ledger_admin:supersecretpassword@localhost:5432/payment_engine?sslmode=disable")
	pool, err := db.NewPool(ctx, dbURL)
	if err != nil {
		logger.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()
	logger.Info("✅ Database connection established")

	// 3. Initialize Services
	repo := ledger.NewRepository(pool)
	ledgerSvc := ledger.NewService(repo)
	chaosBank := bank.NewMockChaosBank(logger)

	// 4. Initialize Kafka Worker
	kafkaBrokers := strings.Split(getEnv("KAFKA_BROKERS", "localhost:9092"), ",")
	topic := getEnv("KAFKA_TOPIC_VALIDATED", "payment_validated")
	groupID := getEnv("KAFKA_GROUP_ID", "go_orchestrator_group")

	paymentWorker := worker.NewPaymentWorker(kafkaBrokers, topic, groupID, ledgerSvc, chaosBank, logger)
	defer paymentWorker.Close()

	// 5. Run Worker in a Goroutine
	go func() {
		paymentWorker.Run(ctx)
	}()

	// Block main thread until context is cancelled
	<-ctx.Done()
	
	// Graceful shutdown period
	logger.Info("Shutting down Orchestrator Service, waiting for active transactions to finish...")
	time.Sleep(2 * time.Second) // Allow in-flight DB transactions to complete
	logger.Info("Orchestrator successfully terminated")
}

func setupSignalHandler(cancel context.CancelFunc, logger *slog.Logger) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		logger.Warn("received termination signal, initiating shutdown sequence")
		cancel()
	}()
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}