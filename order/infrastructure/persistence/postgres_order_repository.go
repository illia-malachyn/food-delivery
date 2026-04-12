package persistence

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/illia-malachyn/microservices/order/application"
	"github.com/illia-malachyn/microservices/order/domain"
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

func (p *PostgresOrderRepository) SaveOrder(ctx context.Context, order *domain.Order) error {
	return pgx.BeginFunc(ctx, p.connPool, func(tx pgx.Tx) error {
		_, err := tx.Exec(
			ctx,
			`INSERT INTO orders (id, user_id, item_id, quantity)
			 VALUES ($1, $2, $3, $4)`,
			order.ID(), order.UserID(), order.ItemID(), order.Quantity(),
		)
		return err
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
