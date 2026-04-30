package application

import (
	"context"
	"errors"

	"github.com/illia-malachyn/food-delivery/payment/domain"
)

var ErrCreatePaymentDTORequired = errors.New("create payment dto is required")
var ErrPaymentRequired = errors.New("payment is required")
var ErrPaymentNotFound = errors.New("payment not found")

type PaymentService struct {
	repository PaymentRepository
	provider   PaymentProvider
}

func NewPaymentService(repository PaymentRepository, providers ...PaymentProvider) *PaymentService {
	var provider PaymentProvider
	if len(providers) > 0 {
		provider = providers[0]
	}
	if provider == nil {
		provider = NewNoopPaymentProvider()
	}
	return &PaymentService{repository: repository, provider: provider}
}

func (s *PaymentService) Create(ctx context.Context, dto *CreatePaymentDTO) (string, error) {
	if dto == nil {
		return "", ErrCreatePaymentDTORequired
	}

	payment, err := domain.NewPayment(dto.OrderID, dto.Amount, dto.Currency)
	if err != nil {
		return "", err
	}

	if err := s.persist(ctx, payment); err != nil {
		return "", err
	}

	return payment.ID(), nil
}

func (s *PaymentService) MarkPaid(ctx context.Context, paymentID string) error {
	payment, err := s.repository.GetByID(ctx, paymentID)
	if err != nil {
		return err
	}

	if err := payment.MarkPaid(); err != nil {
		return err
	}

	if err := s.provider.Capture(ctx, payment.ID(), payment.Amount(), payment.Currency()); err != nil {
		return err
	}

	return s.persist(ctx, payment)
}

func (s *PaymentService) MarkFailed(ctx context.Context, paymentID string, reason string) error {
	payment, err := s.repository.GetByID(ctx, paymentID)
	if err != nil {
		return err
	}

	if err := payment.MarkFailed(reason); err != nil {
		return err
	}

	return s.persist(ctx, payment)
}

func (s *PaymentService) Refund(ctx context.Context, paymentID string) error {
	payment, err := s.repository.GetByID(ctx, paymentID)
	if err != nil {
		return err
	}

	if err := payment.Refund(); err != nil {
		return err
	}

	if err := s.provider.Refund(ctx, payment.ID()); err != nil {
		return err
	}

	return s.persist(ctx, payment)
}

func (s *PaymentService) persist(ctx context.Context, payment *domain.Payment) error {
	if payment == nil {
		return ErrPaymentRequired
	}

	events := payment.FlushEvents()
	return s.repository.Save(ctx, payment, events)
}
