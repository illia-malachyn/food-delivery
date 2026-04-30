package http

import (
	"context"
	"errors"

	"github.com/illia-malachyn/food-delivery/payment/domain"
)

type repositoryStub struct {
	getByIDFn func(ctx context.Context, id string) (*domain.Payment, error)
	saveFn    func(ctx context.Context, payment *domain.Payment, events []domain.DomainEvent) error
}

func (s repositoryStub) GetByID(ctx context.Context, id string) (*domain.Payment, error) {
	if s.getByIDFn == nil {
		return nil, errors.New("getByID not implemented")
	}
	return s.getByIDFn(ctx, id)
}

func (s repositoryStub) Save(ctx context.Context, payment *domain.Payment, events []domain.DomainEvent) error {
	if s.saveFn == nil {
		return errors.New("save not implemented")
	}
	return s.saveFn(ctx, payment, events)
}
