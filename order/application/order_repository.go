package application

import (
	"context"

	"github.com/illia-malachyn/food-delivery/order/domain"
)

type OrderRepository interface {
	// TODO: these methods should accept db's transaction context as well
	GetOrderById(ctx context.Context, id string) (*domain.Order, error)
	SaveOrder(ctx context.Context, order *domain.Order) error
}
