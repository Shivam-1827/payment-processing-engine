package outbox

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func NewPool(ctx context.Context, url string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, err
	}
	cfg.MaxConns = 10
	return pgxpool.NewWithConfig(ctx, cfg)
}

type Event struct {
	ID          uuid.UUID
	AggregateID uuid.UUID
	EventType   string
	Payload     []byte
}

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// FetchUnprocessed batch-fetches events and locks them so other workers ignore them
func (r *Repository) FetchUnprocessed(ctx context.Context, batchSize int) ([]Event, error) {
	// SKIP LOCKED is the magic here. If another worker is already processing row 1, 
	// this query will instantly skip it and grab row 2 instead of waiting for a lock.
	query := `
		SELECT id, aggregate_id, event_type, payload
		FROM outbox_events
		WHERE processed = false
		ORDER BY created_at ASC
		LIMIT $1
		FOR UPDATE SKIP LOCKED
	`

	rows, err := r.db.Query(ctx, query, batchSize)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch outbox events: %w", err)
	}
	defer rows.Close()

	var events []Event
	for rows.Next() {
		var e Event
		if err := rows.Scan(&e.ID, &e.AggregateID, &e.EventType, &e.Payload); err != nil {
			return nil, err
		}
		events = append(events, e)
	}

	return events, rows.Err()
}

// MarkProcessed flags the event as complete so it is never sent again
func (r *Repository) MarkProcessed(ctx context.Context, eventID uuid.UUID) error {
	_, err := r.db.Exec(ctx, `UPDATE outbox_events SET processed = true WHERE id = $1`, eventID)
	return err
}