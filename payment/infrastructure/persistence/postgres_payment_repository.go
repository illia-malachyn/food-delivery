package persistence

import (
	"context"
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

func (r *PostgresPaymentRepository) Save(ctx context.Context, payment *domain.Payment, _ []domain.DomainEvent) error {
	_, err := r.connPool.Exec(
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
