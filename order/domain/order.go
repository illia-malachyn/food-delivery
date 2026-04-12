package domain

import (
	"errors"
	"fmt"
	"time"
)

var counter int = 0

var ErrValidationFailed = errors.New("validation failed")

// Order is an aggregate root representing an order
type Order struct {
	id       string
	userId   string
	itemID   string
	quantity uint
	events   []domainEvent
}

func NewOrder(userId string, itemId string, quantity uint) (*Order, error) {
	if userId == "" {
		return nil, ErrValidationFailed
	}
	if itemId == "" {
		return nil, ErrValidationFailed
	}

	counter = counter + 1
	return &Order{
		id:       fmt.Sprintf("id-%d", counter),
		userId:   userId,
		itemID:   itemId,
		quantity: quantity,
	}, nil
}

func ReconstructOrder(id, userId, itemID string, quantity uint) *Order {
	return &Order{
		id:       id,
		userId:   userId,
		itemID:   itemID,
		quantity: quantity,
	}
}

func (o *Order) Place() error {
	o.events = append(o.events, OrderPlacedEvent{
		EventBase: EventBase{
			OrderID:    o.id,
			OccurredAt: time.Now(),
		},
	})

	return nil
}

func (o *Order) FlushEvents() []domainEvent {
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
