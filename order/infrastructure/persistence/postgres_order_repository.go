package persistence

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/illia-malachyn/food-delivery/order/application"
	"github.com/illia-malachyn/food-delivery/order/domain"
)

type PostgresOrderRepository struct {
	connPool *pgxpool.Pool
}

var _ application.OrderRepository = (*PostgresOrderRepository)(nil)

func NewPostgresOrderRepository(connPool *pgxpool.Pool) *PostgresOrderRepository {
	return &PostgresOrderRepository{
		connPool: connPool,
	}
}

func (p *PostgresOrderRepository) SaveOrder(ctx context.Context, order *domain.Order, events []application.IntegrationEvent) error {
	return pgx.BeginFunc(ctx, p.connPool, func(tx pgx.Tx) error {
		_, err := tx.Exec(
			ctx,
			`INSERT INTO orders (id, user_id, item_id, quantity)
			 VALUES ($1, $2, $3, $4)`,
			order.ID(), order.UserID(), order.ItemID(), order.Quantity(),
		)
		if err != nil {
			return err
		}

		for _, event := range events {
			payload, marshalErr := json.Marshal(event)
			if marshalErr != nil {
				return fmt.Errorf("cannot marshal integration event %s v%d: %w", event.EventName(), event.EventVersion(), marshalErr)
			}

			_, execErr := tx.Exec(
				ctx,
				`INSERT INTO outbox (
					aggregate_type,
					aggregate_id,
					event_name,
					event_version,
					payload,
					occurred_at
				) VALUES ($1, $2, $3, $4, $5, $6)`,
				event.AggregateType(),
				event.AggregateID(),
				event.EventName(),
				event.EventVersion(),
				payload,
				event.EventOccurredAt(),
			)
			if execErr != nil {
				return fmt.Errorf("cannot insert outbox event %s v%d: %w", event.EventName(), event.EventVersion(), execErr)
			}
		}

		return nil
	})
}

func (p *PostgresOrderRepository) GetOrderById(ctx context.Context, id string) (*domain.Order, error) {
	row := p.connPool.QueryRow(ctx, `SELECT * FROM orders WHERE id = $1`, id)

	var order domain.Order
	if err := row.Scan(&order); err != nil {
		return nil, fmt.Errorf("cannot Scan order: %w", err)
	}

	return &order, nil
}
