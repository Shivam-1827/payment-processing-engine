package dispatcher

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/Shivam-1827/payment-engine/services/webhook-service/internal/outbox"
)

type Worker struct {
	repo       *outbox.Repository
	httpClient *http.Client
	logger     *slog.Logger
	targetURL  string // In reality, this comes from a Merchant DB lookup
}

func NewWorker(repo *outbox.Repository, targetURL string, logger *slog.Logger) *Worker {
	return &Worker{
		repo: repo,
		httpClient: &http.Client{
			Timeout: 5 * time.Second, // Never hang forever on a dead merchant server
		},
		targetURL: targetURL,
		logger:    logger,
	}
}

func (w *Worker) Start(ctx context.Context, pollInterval time.Duration, batchSize int) {
	w.logger.Info("Webhook Dispatcher started", "poll_interval", pollInterval)
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			w.logger.Info("Stopping Webhook Dispatcher...")
			return
		case <-ticker.C:
			w.processBatch(ctx, batchSize)
		}
	}
}

func (w *Worker) processBatch(ctx context.Context, batchSize int) {
	// 1. Fetch locked batch
	events, err := w.repo.FetchUnprocessed(ctx, batchSize)
	if err != nil {
		w.logger.Error("failed to fetch outbox events", "error", err)
		return
	}

	if len(events) == 0 {
		return // Nothing to process
	}

	w.logger.Info("processing outbox batch", "count", len(events))

	// 2. Dispatch each event
	for _, event := range events {
		err := w.sendWebhook(ctx, event)
		
		if err != nil {
			w.logger.Warn("webhook delivery failed, will retry next sweep", "event_id", event.ID, "error", err)
			continue // Leave processed = false so it gets picked up again
		}

		// 3. Mark successful delivery
		if err := w.repo.MarkProcessed(ctx, event.ID); err != nil {
			w.logger.Error("failed to mark event as processed", "event_id", event.ID, "error", err)
		} else {
			w.logger.Info("webhook delivered successfully", "event_id", event.ID)
		}
	}
}

func (w *Worker) sendWebhook(ctx context.Context, event outbox.Event) error {
	// Construct the HTTP POST request
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, w.targetURL, bytes.NewReader(event.Payload))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Webhook-Event", event.EventType)

	// Execute request
	resp, err := w.httpClient.Do(req)
	if err != nil {
		return err // Network failure
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("merchant returned non-200 status: %d", resp.StatusCode)
	}

	return nil
}