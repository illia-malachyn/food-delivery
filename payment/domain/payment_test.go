package domain_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/illia-malachyn/food-delivery/payment/domain"
)

func TestNewPayment_ValidatesInput(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		orderID  string
		amount   int64
		currency string
	}{
		{name: "empty order id", orderID: "", amount: 100, currency: "USD"},
		{name: "zero amount", orderID: "order-1", amount: 0, currency: "USD"},
		{name: "negative amount", orderID: "order-1", amount: -10, currency: "USD"},
		{name: "invalid currency lower", orderID: "order-1", amount: 100, currency: "usd"},
		{name: "invalid currency short", orderID: "order-1", amount: 100, currency: "US"},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			payment, err := domain.NewPayment(tc.orderID, tc.amount, tc.currency)
			require.ErrorIs(t, err, domain.ErrValidationFailed)
			assert.Nil(t, payment)
		})
	}
}

func TestNewPayment_CreatesPendingWithInitiatedEvent(t *testing.T) {
	t.Parallel()

	payment, err := domain.NewPayment(" order-1 ", 1599, "USD")
	require.NoError(t, err)

	_, err = uuid.Parse(payment.ID())
	require.NoError(t, err)
	assert.Equal(t, "order-1", payment.OrderID())
	assert.Equal(t, domain.PaymentStatusPending, payment.Status())

	events := payment.FlushEvents()
	require.Len(t, events, 1)
	assert.Equal(t, "PaymentInitiated", events[0].EventName())
}

func TestPayment_Transitions(t *testing.T) {
	t.Parallel()

	t.Run("pending to paid", func(t *testing.T) {
		payment, err := domain.NewPayment("order-1", 1000, "USD")
		require.NoError(t, err)

		err = payment.MarkPaid()
		require.NoError(t, err)
		assert.Equal(t, domain.PaymentStatusPaid, payment.Status())

		events := payment.FlushEvents()
		require.Len(t, events, 2)
		assert.Equal(t, "PaymentPaid", events[1].EventName())
	})

	t.Run("pending to failed", func(t *testing.T) {
		payment, err := domain.NewPayment("order-1", 1000, "USD")
		require.NoError(t, err)

		err = payment.MarkFailed("declined")
		require.NoError(t, err)
		assert.Equal(t, domain.PaymentStatusFailed, payment.Status())
		assert.Equal(t, "declined", payment.FailureReason())
	})

	t.Run("paid to refunded", func(t *testing.T) {
		payment, err := domain.NewPayment("order-1", 1000, "USD")
		require.NoError(t, err)
		require.NoError(t, payment.MarkPaid())

		err = payment.Refund()
		require.NoError(t, err)
		assert.Equal(t, domain.PaymentStatusRefunded, payment.Status())
	})

	t.Run("reject invalid transitions", func(t *testing.T) {
		payment, err := domain.NewPayment("order-1", 1000, "USD")
		require.NoError(t, err)

		err = payment.Refund()
		require.ErrorIs(t, err, domain.ErrInvalidStateTransition)

		require.NoError(t, payment.MarkFailed("declined"))

		err = payment.MarkPaid()
		require.ErrorIs(t, err, domain.ErrInvalidStateTransition)
	})
}

func TestReconstructPayment_ValidatesRules(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()

	payment, err := domain.ReconstructPayment(
		"id-1",
		"order-1",
		1000,
		"USD",
		domain.PaymentStatusFailed,
		"declined",
		now,
	)
	require.NoError(t, err)
	assert.Equal(t, domain.PaymentStatusFailed, payment.Status())

	payment, err = domain.ReconstructPayment(
		"id-1",
		"order-1",
		1000,
		"USD",
		domain.PaymentStatusFailed,
		"",
		now,
	)
	require.ErrorIs(t, err, domain.ErrValidationFailed)
	assert.Nil(t, payment)
}
