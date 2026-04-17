package application

import (
	"context"

	"github.com/illia-malachyn/food-delivery/order/domain"
)

type OrderService struct {
	orderRepository OrderRepository
}

func NewOrderService(repository OrderRepository) *OrderService {
	return &OrderService{
		repository,
	}
}

func (s *OrderService) Create(ctx context.Context, orderDTO *OrderDTO) error {
	order, err := domain.NewOrder(orderDTO.UserId, orderDTO.ItemId, orderDTO.Quantity)
	if err != nil {
		return err
	}

	if err = order.Place(); err != nil {
		return err
	}

	return s.orderRepository.SaveOrder(ctx, order)
}
