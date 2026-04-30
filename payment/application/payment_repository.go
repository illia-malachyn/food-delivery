package application

import (
	"context"

	"github.com/illia-malachyn/food-delivery/payment/domain"
)

type PaymentRepository interface {
	GetByID(ctx context.Context, id string) (*domain.Payment, error)
	GetByOrderID(ctx context.Context, orderID string) (*domain.Payment, error)
	Save(ctx context.Context, payment *domain.Payment, events []domain.DomainEvent) error
}
