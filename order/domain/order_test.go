package domain_test

import (
	"testing"

	"github.com/illia-malachyn/food-delivery/order/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewOrder_ValidatesInput(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		userID   string
		itemID   string
		quantity uint
	}{
		{name: "empty user", userID: "", itemID: "item-1", quantity: 1},
		{name: "blank user", userID: "   ", itemID: "item-1", quantity: 1},
		{name: "empty item", userID: "user-1", itemID: "", quantity: 1},
		{name: "blank item", userID: "user-1", itemID: "   ", quantity: 1},
		{name: "zero quantity", userID: "user-1", itemID: "item-1", quantity: 0},
		{name: "too big quantity", userID: "user-1", itemID: "item-1", quantity: domain.MaxOrderQuantity + 1},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			order, err := domain.NewOrder(tc.userID, tc.itemID, tc.quantity)
			require.Error(t, err)
			assert.Nil(t, order)
		})
	}
}

func TestOrderPlace_TransitionsToPlacedAndEmitsEvent(t *testing.T) {
	t.Parallel()

	order, err := domain.NewOrder("user-1", "item-1", 2)
	require.NoError(t, err)

	assert.Equal(t, domain.OrderStatusDraft, order.Status())

	err = order.Place()
	require.NoError(t, err)

	assert.Equal(t, domain.OrderStatusPlaced, order.Status())

	events := order.FlushEvents()
	require.Len(t, events, 1)

	placedEvent, ok := events[0].(domain.OrderPlacedEvent)
	require.True(t, ok)
	assert.Equal(t, "user-1", placedEvent.UserID)
	assert.Equal(t, "item-1", placedEvent.ItemID)
	assert.Equal(t, uint(2), placedEvent.Quantity)
}

func TestNewOrder_NormalizesIDs(t *testing.T) {
	t.Parallel()

	order, err := domain.NewOrder(" user-1 ", " item-1 ", 1)
	require.NoError(t, err)

	assert.Equal(t, "user-1", order.UserID())
	assert.Equal(t, "item-1", order.ItemID())
}

func TestOrderPlace_RejectsDuplicatePlace(t *testing.T) {
	t.Parallel()

	order, err := domain.NewOrder("user-1", "item-1", 1)
	require.NoError(t, err)
	err = order.Place()
	require.NoError(t, err)

	assert.ErrorIs(t, order.Place(), domain.ErrInvalidStateTransition)
}

func TestOrderConfirm_OnlyAllowedFromPlaced(t *testing.T) {
	t.Parallel()

	order, err := domain.NewOrder("user-1", "item-1", 1)
	require.NoError(t, err)

	assert.ErrorIs(t, order.Confirm(), domain.ErrInvalidStateTransition)

	err = order.Place()
	require.NoError(t, err)
	err = order.Confirm()
	require.NoError(t, err)
	assert.Equal(t, domain.OrderStatusConfirmed, order.Status())

	events := order.FlushEvents()
	require.Len(t, events, 2)
	_, ok := events[1].(domain.OrderConfirmedEvent)
	assert.True(t, ok)
}

func TestOrderCancel_Rules(t *testing.T) {
	t.Parallel()

	order, err := domain.NewOrder("user-1", "item-1", 1)
	require.NoError(t, err)

	assert.ErrorIs(t, order.Cancel("customer requested"), domain.ErrInvalidStateTransition)

	err = order.Place()
	require.NoError(t, err)

	assert.ErrorIs(t, order.Cancel("x"), domain.ErrValidationFailed)

	err = order.Cancel("  payment failed  ")
	require.NoError(t, err)
	assert.Equal(t, domain.OrderStatusCancelled, order.Status())

	events := order.FlushEvents()
	require.Len(t, events, 2)

	cancelledEvent, ok := events[1].(domain.OrderCancelledEvent)
	require.True(t, ok)
	assert.Equal(t, "payment failed", cancelledEvent.Reason)
}

func TestOrderCancel_RejectsConfirmedOrder(t *testing.T) {
	t.Parallel()

	order, err := domain.NewOrder("user-1", "item-1", 1)
	require.NoError(t, err)
	err = order.Place()
	require.NoError(t, err)
	err = order.Confirm()
	require.NoError(t, err)

	assert.ErrorIs(t, order.Cancel("customer changed mind"), domain.ErrInvalidStateTransition)
}

func TestReconstructOrder_ValidatesInput(t *testing.T) {
	t.Parallel()

	order, err := domain.ReconstructOrder("id-1", "user-1", "item-1", 2, domain.OrderStatusPlaced)
	require.NoError(t, err)
	assert.Equal(t, domain.OrderStatusPlaced, order.Status())

	invalidOrder, invalidErr := domain.ReconstructOrder("id-1", "user-1", "item-1", 2, domain.OrderStatus("unknown"))
	require.Error(t, invalidErr)
	assert.Nil(t, invalidOrder)

	invalidQuantityOrder, invalidQuantityErr := domain.ReconstructOrder("id-1", "user-1", "item-1", domain.MaxOrderQuantity+1, domain.OrderStatusPlaced)
	require.Error(t, invalidQuantityErr)
	assert.Nil(t, invalidQuantityOrder)
}
