package persistence

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/illia-malachyn/food-delivery/payment/application"
	"github.com/illia-malachyn/food-delivery/payment/domain"
)

type PostgresPaymentRepository struct {
	connPool *pgxpool.Pool
}

var _ application.PaymentRepository = (*PostgresPaymentRepository)(nil)

func NewPostgresPaymentRepository(connPool *pgxpool.Pool) *PostgresPaymentRepository {
	return &PostgresPaymentRepository{connPool: connPool}
}

func (r *PostgresPaymentRepository) GetByID(ctx context.Context, id string) (*domain.Payment, error) {
	return r.getOne(ctx, `SELECT id, order_id, amount::bigint, currency, status, COALESCE(failure_reason, ''), created_at
		FROM payments
		WHERE id = $1`, id)
}

func (r *PostgresPaymentRepository) GetByOrderID(ctx context.Context, orderID string) (*domain.Payment, error) {
	return r.getOne(ctx, `SELECT id, order_id, amount::bigint, currency, status, COALESCE(failure_reason, ''), created_at
		FROM payments
		WHERE order_id = $1
		ORDER BY created_at DESC
		LIMIT 1`, orderID)
}

func (r *PostgresPaymentRepository) getOne(ctx context.Context, query string, arg string) (*domain.Payment, error) {
	row := r.connPool.QueryRow(ctx, query, arg)

	var paymentID string
	var orderID string
	var amount int64
	var currency string
	var status string
	var failureReason string
	var createdAt time.Time

	if err := row.Scan(&paymentID, &orderID, &amount, &currency, &status, &failureReason, &createdAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, application.ErrPaymentNotFound
		}
		return nil, fmt.Errorf("cannot scan payment: %w", err)
	}

	payment, err := domain.ReconstructPayment(
		paymentID,
		orderID,
		amount,
		currency,
		domain.PaymentStatus(status),
		failureReason,
		createdAt,
	)
	if err != nil {
		return nil, fmt.Errorf("cannot reconstruct payment: %w", err)
	}

	return payment, nil
}

func (r *PostgresPaymentRepository) Save(ctx context.Context, payment *domain.Payment, events []domain.DomainEvent) error {
	return pgx.BeginFunc(ctx, r.connPool, func(tx pgx.Tx) error {
		if err := upsertPayment(ctx, tx, payment); err != nil {
			return err
		}

		return appendOutboxEvents(ctx, tx, payment, events)
	})
}

func upsertPayment(ctx context.Context, tx pgx.Tx, payment *domain.Payment) error {
	_, err := tx.Exec(
		ctx,
		`INSERT INTO payments (id, order_id, amount, currency, status, failure_reason)
		 VALUES ($1, $2, $3, $4, $5, NULLIF($6, ''))
		 ON CONFLICT (id) DO UPDATE SET
			order_id = EXCLUDED.order_id,
			amount = EXCLUDED.amount,
			currency = EXCLUDED.currency,
			status = EXCLUDED.status,
			failure_reason = EXCLUDED.failure_reason,
			paid_at = CASE
				WHEN EXCLUDED.status = 'paid' AND payments.paid_at IS NULL THEN NOW()
				ELSE payments.paid_at
			END,
			updated_at = NOW()`,
		payment.ID(),
		payment.OrderID(),
		payment.Amount(),
		payment.Currency(),
		string(payment.Status()),
		payment.FailureReason(),
	)
	if err != nil {
		return fmt.Errorf("cannot save payment: %w", err)
	}

	return nil
}

func appendOutboxEvents(ctx context.Context, tx pgx.Tx, payment *domain.Payment, events []domain.DomainEvent) error {
	for _, event := range events {
		record, ok, err := mapDomainEventToOutboxRecord(payment, event)
		if err != nil {
			return err
		}
		if !ok {
			continue
		}

		payload, marshalErr := json.Marshal(record.payload)
		if marshalErr != nil {
			return fmt.Errorf("cannot marshal payment outbox payload for %s: %w", record.eventName, marshalErr)
		}

		_, execErr := tx.Exec(
			ctx,
			`INSERT INTO payment_outbox (
				aggregate_type,
				aggregate_id,
				event_name,
				event_version,
				payload,
				occurred_at
			) VALUES ($1, $2, $3, $4, $5, $6)`,
			record.aggregateType,
			record.aggregateID,
			record.eventName,
			record.eventVersion,
			payload,
			record.occurredAt,
		)
		if execErr != nil {
			return fmt.Errorf("cannot insert payment outbox event %s v%d: %w", record.eventName, record.eventVersion, execErr)
		}
	}

	return nil
}
