package application

type EventUpcaster interface {
	Upcast(events []IntegrationEvent) []IntegrationEvent
}

type IntegrationEventUpcaster struct{}

func NewIntegrationEventUpcaster() *IntegrationEventUpcaster {
	return &IntegrationEventUpcaster{}
}

func (u *IntegrationEventUpcaster) Upcast(events []IntegrationEvent) []IntegrationEvent {
	upcasted := make([]IntegrationEvent, 0, len(events))

	for _, event := range events {
		switch e := event.(type) {
		case OrderPlacedEvent:
			upcasted = append(upcasted, OrderPlacedEventV2{
				Version:    2,
				OrderID:    e.OrderID,
				CustomerID: e.UserID,
				ItemID:     e.ItemID,
				Quantity:   e.Quantity,
				OccurredAt: e.OccurredAt,
				Source:     "order-service",
			})
		default:
			upcasted = append(upcasted, event)
		}
	}

	return upcasted
}
