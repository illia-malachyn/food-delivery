package domain

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

var counter int = 0

var ErrValidationFailed = errors.New("validation failed")
var ErrInvalidStateTransition = errors.New("invalid state transition")

type OrderStatus string

const (
	OrderStatusDraft     OrderStatus = "draft"
	OrderStatusPlaced    OrderStatus = "placed"
	OrderStatusConfirmed OrderStatus = "confirmed"
	OrderStatusCancelled OrderStatus = "cancelled"
)

const (
	MaxOrderQuantity      uint = 50
	MinCancellationReason      = 5
)

// Order is an aggregate root representing an order
type Order struct {
	id       string
	userId   string
	itemID   string
	quantity uint
	status   OrderStatus
	events   []DomainEvent
}

func NewOrder(userId string, itemId string, quantity uint) (*Order, error) {
	normalizedUserID := strings.TrimSpace(userId)
	normalizedItemID := strings.TrimSpace(itemId)

	if normalizedUserID == "" {
		return nil, ErrValidationFailed
	}
	if normalizedItemID == "" {
		return nil, ErrValidationFailed
	}
	if quantity == 0 || quantity > MaxOrderQuantity {
		return nil, ErrValidationFailed
	}

	counter = counter + 1
	return &Order{
		id:       fmt.Sprintf("id-%d", counter),
		userId:   normalizedUserID,
		itemID:   normalizedItemID,
		quantity: quantity,
		status:   OrderStatusDraft,
	}, nil
}

func ReconstructOrder(id, userId, itemID string, quantity uint, status OrderStatus) (*Order, error) {
	normalizedID := strings.TrimSpace(id)
	normalizedUserID := strings.TrimSpace(userId)
	normalizedItemID := strings.TrimSpace(itemID)

	if normalizedID == "" || normalizedUserID == "" || normalizedItemID == "" || quantity == 0 || quantity > MaxOrderQuantity || !status.IsValid() {
		return nil, ErrValidationFailed
	}

	return &Order{
		id:       normalizedID,
		userId:   normalizedUserID,
		itemID:   normalizedItemID,
		quantity: quantity,
		status:   status,
	}, nil
}

func (o *Order) Place() error {
	if o.status != OrderStatusDraft {
		return ErrInvalidStateTransition
	}

	o.status = OrderStatusPlaced
	o.events = append(o.events, OrderPlacedEvent{
		EventBase: EventBase{
			OrderID:    o.id,
			OccurredAt: time.Now(),
		},
		UserID:   o.userId,
		ItemID:   o.itemID,
		Quantity: o.quantity,
	})

	return nil
}

func (o *Order) Confirm() error {
	if o.status != OrderStatusPlaced {
		return ErrInvalidStateTransition
	}

	o.status = OrderStatusConfirmed
	o.events = append(o.events, OrderConfirmedEvent{
		EventBase: EventBase{
			OrderID:    o.id,
			OccurredAt: time.Now(),
		},
	})

	return nil
}

func (o *Order) Cancel(reason string) error {
	normalizedReason := strings.TrimSpace(reason)
	if len(normalizedReason) < MinCancellationReason {
		return ErrValidationFailed
	}

	if o.status != OrderStatusPlaced {
		return ErrInvalidStateTransition
	}

	o.status = OrderStatusCancelled
	o.events = append(o.events, OrderCancelledEvent{
		EventBase: EventBase{
			OrderID:    o.id,
			OccurredAt: time.Now(),
		},
		Reason: normalizedReason,
	})

	return nil
}

func (o *Order) FlushEvents() []DomainEvent {
	events := o.events
	o.events = nil
	return events
}

func (o *Order) ID() string {
	return o.id
}

func (o *Order) UserID() string {
	return o.userId
}

func (o *Order) ItemID() string {
	return o.itemID
}

func (o *Order) Quantity() uint {
	return o.quantity
}

func (o *Order) Status() OrderStatus {
	return o.status
}

func (s OrderStatus) IsValid() bool {
	switch s {
	case OrderStatusDraft, OrderStatusPlaced, OrderStatusConfirmed, OrderStatusCancelled:
		return true
	default:
		return false
	}
}
