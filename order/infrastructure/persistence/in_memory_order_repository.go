package persistence

import (
	"context"
	"errors"
	"sync"

	"github.com/illia-malachyn/microservices/order/application"
	"github.com/illia-malachyn/microservices/order/domain"
)

var ErrOrderNotFound = errors.New("order not found")

type InMemoryOrderRepository struct {
	orders map[string]*domain.Order
	mu     sync.Mutex
}

var _ application.OrderRepository = (*InMemoryOrderRepository)(nil)

func NewInMemoryOrderRepository() *InMemoryOrderRepository {
	return &InMemoryOrderRepository{
		orders: make(map[string]*domain.Order),
	}
}

func (r *InMemoryOrderRepository) SaveOrder(_ context.Context, order *domain.Order) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.orders[order.ID()] = order

	return nil
}

func (r *InMemoryOrderRepository) GetOrderById(_ context.Context, id string) (*domain.Order, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	order, ok := r.orders[id]
	if !ok {
		return nil, ErrOrderNotFound
	}

	return order, nil
}
