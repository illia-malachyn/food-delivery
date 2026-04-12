package application

import (
	"context"

	"github.com/illia-malachyn/microservices/order/domain"
)

type OrderRepository interface {
	GetOrderById(ctx context.Context, id string) (*domain.Order, error)
	SaveOrder(ctx context.Context, order *domain.Order) error
}
