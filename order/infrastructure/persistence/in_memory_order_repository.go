package persistence

import (
	"context"
	"errors"
	"sync"

	"github.com/illia-malachyn/food-delivery/order/application"
	"github.com/illia-malachyn/food-delivery/order/domain"
)

var ErrOrderNotFound = errors.New("order not found")

type InMemoryOrderRepository struct {
	orders map[string]*domain.Order
	events []application.IntegrationEvent
	mu     sync.Mutex
}

var _ application.OrderRepository = (*InMemoryOrderRepository)(nil)

func NewInMemoryOrderRepository() *InMemoryOrderRepository {
	return &InMemoryOrderRepository{
		orders: make(map[string]*domain.Order),
		events: make([]application.IntegrationEvent, 0),
	}
}

func (r *InMemoryOrderRepository) SaveOrder(_ context.Context, order *domain.Order, events []application.IntegrationEvent) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.orders[order.ID()] = order
	r.events = append(r.events, events...)

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
