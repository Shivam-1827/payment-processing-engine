package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/Shivam-1827/payment-engine/services/orchestrator/internal/db"
	"github.com/Shivam-1827/payment-engine/services/orchestrator/internal/ledger"
)

func main(){

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	setupSignalHandler(cancel)

	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		dbURL = "postgres://ledger_admin:supersecretpassword@localhost:5432/payment_engine?sslmode=disable"
	}

	log.Println("Connecting to the database...")
	pool, err := db.NewPool(ctx, dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	defer pool.Close()
	log.Println("Database connection established")

	repo := ledger.NewRepository(pool)
	_ = ledger.NewService(repo)

	log.Println("Orchestrator service is running. Waiting for events...")

	<-ctx.Done()
	log.Println("Shutting down Orchestrator Service gracefully");
}

func setupSignalHandler(cancel context.CancelFunc){
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func ()  {
		<-c
		log.Println("\n Received termination signal. Initiating shotdown...")
		cancel()
	}()
}