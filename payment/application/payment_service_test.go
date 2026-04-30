package application_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/illia-malachyn/food-delivery/payment/application"
	"github.com/illia-malachyn/food-delivery/payment/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestPaymentService_Create(t *testing.T) {
	t.Parallel()

	t.Run("creates payment and saves initiated event", func(t *testing.T) {
		var savedPayment *domain.Payment
		var savedEvents []domain.DomainEvent

		svc := application.NewPaymentService(repositoryStub{
			saveFn: func(_ context.Context, payment *domain.Payment, events []domain.DomainEvent) error {
				savedPayment = payment
				savedEvents = events
				return nil
			},
		})

		paymentID, err := svc.Create(context.Background(), &application.CreatePaymentDTO{
			OrderID:  "order-1",
			Amount:   1200,
			Currency: "USD",
		})
		require.NoError(t, err)

		_, err = uuid.Parse(paymentID)
		require.NoError(t, err)
		require.NotNil(t, savedPayment)
		assert.Equal(t, "order-1", savedPayment.OrderID())
		assert.Equal(t, domain.PaymentStatusPending, savedPayment.Status())
		require.Len(t, savedEvents, 1)
		assert.Equal(t, "PaymentInitiated", savedEvents[0].EventName())
	})

	t.Run("rejects nil dto", func(t *testing.T) {
		svc := application.NewPaymentService(repositoryStub{})

		paymentID, err := svc.Create(context.Background(), nil)
		require.ErrorIs(t, err, application.ErrCreatePaymentDTORequired)
		assert.Empty(t, paymentID)
	})
}

func TestPaymentService_MarkPaid(t *testing.T) {
	t.Parallel()

	payment, err := domain.NewPayment("order-1", 1200, "USD")
	require.NoError(t, err)
	paymentID := payment.ID()
	_ = payment.FlushEvents()

	var savedEvents []domain.DomainEvent
	svc := application.NewPaymentService(repositoryStub{
		getByIDFn: func(_ context.Context, id string) (*domain.Payment, error) {
			assert.Equal(t, paymentID, id)
			return payment, nil
		},
		saveFn: func(_ context.Context, _ *domain.Payment, events []domain.DomainEvent) error {
			savedEvents = events
			return nil
		},
	})

	err = svc.MarkPaid(context.Background(), paymentID)
	require.NoError(t, err)
	assert.Equal(t, domain.PaymentStatusPaid, payment.Status())
	require.Len(t, savedEvents, 1)
	assert.Equal(t, "PaymentPaid", savedEvents[0].EventName())
}

func TestPaymentService_MarkFailed(t *testing.T) {
	t.Parallel()

	payment, err := domain.NewPayment("order-1", 1200, "USD")
	require.NoError(t, err)
	paymentID := payment.ID()
	_ = payment.FlushEvents()

	var savedEvents []domain.DomainEvent
	svc := application.NewPaymentService(repositoryStub{
		getByIDFn: func(_ context.Context, id string) (*domain.Payment, error) {
			assert.Equal(t, paymentID, id)
			return payment, nil
		},
		saveFn: func(_ context.Context, _ *domain.Payment, events []domain.DomainEvent) error {
			savedEvents = events
			return nil
		},
	})

	err = svc.MarkFailed(context.Background(), paymentID, "declined")
	require.NoError(t, err)
	assert.Equal(t, domain.PaymentStatusFailed, payment.Status())
	assert.Equal(t, "declined", payment.FailureReason())
	require.Len(t, savedEvents, 1)
	assert.Equal(t, "PaymentFailed", savedEvents[0].EventName())
}

func TestPaymentService_Refund(t *testing.T) {
	t.Parallel()

	payment, err := domain.NewPayment("order-1", 1200, "USD")
	require.NoError(t, err)
	require.NoError(t, payment.MarkPaid())
	_ = payment.FlushEvents()

	paymentID := payment.ID()
	var savedEvents []domain.DomainEvent

	svc := application.NewPaymentService(repositoryStub{
		getByIDFn: func(_ context.Context, id string) (*domain.Payment, error) {
			assert.Equal(t, paymentID, id)
			return payment, nil
		},
		saveFn: func(_ context.Context, _ *domain.Payment, events []domain.DomainEvent) error {
			savedEvents = events
			return nil
		},
	})

	err = svc.Refund(context.Background(), paymentID)
	require.NoError(t, err)
	assert.Equal(t, domain.PaymentStatusRefunded, payment.Status())
	require.Len(t, savedEvents, 1)
	assert.Equal(t, "PaymentRefunded", savedEvents[0].EventName())
}

func TestPaymentService_PropagatesRepositoryErrors(t *testing.T) {
	t.Parallel()

	repoErr := errors.New("repo error")
	svc := application.NewPaymentService(repositoryStub{
		getByIDFn: func(_ context.Context, _ string) (*domain.Payment, error) {
			return nil, repoErr
		},
	})

	err := svc.MarkPaid(context.Background(), "missing")
	require.ErrorIs(t, err, repoErr)
}
