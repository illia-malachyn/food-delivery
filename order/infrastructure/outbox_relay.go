package infrastructure

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// OutboxRelay polls unpublished events from the outbox table and publishes them to Kafka.
type OutboxRelay struct {
	connPool     *pgxpool.Pool
	publisher    *KafkaOutboxPublisher
	batchSize    int
	pollInterval time.Duration
}

func NewOutboxRelay(connPool *pgxpool.Pool, publisher *KafkaOutboxPublisher, batchSize int, pollInterval time.Duration) *OutboxRelay {
	if batchSize <= 0 {
		batchSize = 100
	}
	if pollInterval <= 0 {
		pollInterval = 2 * time.Second
	}

	return &OutboxRelay{
		connPool:     connPool,
		publisher:    publisher,
		batchSize:    batchSize,
		pollInterval: pollInterval,
	}
}

func (r *OutboxRelay) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		publishedCount, err := r.publishPendingBatch(ctx)
		if err != nil {
			log.Printf("outbox relay batch failed: %v", err)
		}

		if publishedCount >= r.batchSize {
			continue
		}

		select {
		case <-ctx.Done():
			return
		case <-time.After(r.pollInterval):
		}
	}
}

func (r *OutboxRelay) publishPendingBatch(ctx context.Context) (int, error) {
	publishedCount := 0

	err := pgx.BeginFunc(ctx, r.connPool, func(tx pgx.Tx) error {
		events, err := r.lockPendingEvents(ctx, tx)
		if err != nil {
			return err
		}

		for _, event := range events {
			if err := r.publisher.Publish(ctx, event); err != nil {
				log.Printf("outbox publish failed for event id=%s name=%s v=%d: %v", event.ID, event.EventName, event.EventVersion, err)
				if updateErr := r.increaseRetryCount(ctx, tx, event.ID); updateErr != nil {
					return updateErr
				}
				continue
			}

			if err := r.markPublished(ctx, tx, event.ID); err != nil {
				return err
			}

			publishedCount++
		}

		return nil
	})
	if err != nil {
		return 0, err
	}

	return publishedCount, nil
}

func (r *OutboxRelay) lockPendingEvents(ctx context.Context, tx pgx.Tx) ([]OutboxEvent, error) {
	rows, err := tx.Query(
		ctx,
		`SELECT id, aggregate_type, aggregate_id, event_name, event_version, payload, occurred_at
		 FROM outbox
		 WHERE published_at IS NULL
		 ORDER BY created_at
		 LIMIT $1
		 FOR UPDATE SKIP LOCKED`,
		r.batchSize,
	)
	if err != nil {
		return nil, fmt.Errorf("select outbox events: %w", err)
	}
	defer rows.Close()

	events := make([]OutboxEvent, 0, r.batchSize)
	for rows.Next() {
		var event OutboxEvent
		if scanErr := rows.Scan(
			&event.ID,
			&event.AggregateType,
			&event.AggregateID,
			&event.EventName,
			&event.EventVersion,
			&event.Payload,
			&event.OccurredAt,
		); scanErr != nil {
			return nil, fmt.Errorf("scan outbox event: %w", scanErr)
		}
		events = append(events, event)
	}

	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, fmt.Errorf("iterate outbox events: %w", rowsErr)
	}

	return events, nil
}

func (r *OutboxRelay) markPublished(ctx context.Context, tx pgx.Tx, eventID string) error {
	if _, err := tx.Exec(ctx, `UPDATE outbox SET published_at = NOW() WHERE id = $1`, eventID); err != nil {
		return fmt.Errorf("mark outbox event %s as published: %w", eventID, err)
	}
	return nil
}

func (r *OutboxRelay) increaseRetryCount(ctx context.Context, tx pgx.Tx, eventID string) error {
	if _, err := tx.Exec(ctx, `UPDATE outbox SET retry_count = retry_count + 1 WHERE id = $1`, eventID); err != nil {
		return fmt.Errorf("increment retry_count for outbox event %s: %w", eventID, err)
	}
	return nil
}
