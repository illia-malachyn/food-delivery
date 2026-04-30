package infrastructure

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/segmentio/kafka-go"

	"github.com/illia-malachyn/food-delivery/payment/application"
	"github.com/illia-malachyn/food-delivery/payment/domain"
	"github.com/illia-malachyn/food-delivery/shared/resilience"
)

type OrderPlacedEvent struct {
	Version  int       `json:"version"`
	OrderID  string    `json:"order_id"`
	Quantity uint      `json:"quantity"`
	Occurred time.Time `json:"occurred_at"`
}

type OrderCancelledEvent struct {
	Version  int       `json:"version"`
	OrderID  string    `json:"order_id"`
	Reason   string    `json:"reason"`
	Occurred time.Time `json:"occurred_at"`
}

type OrderEventsConsumer struct {
	reader          *kafka.Reader
	paymentService  *application.PaymentService
	repository      application.PaymentRepository
	defaultAmount   int64
	defaultCurrency string
	retryPolicy     resilience.RetryPolicy
}

type OrderEventsConsumerOption func(*OrderEventsConsumer)

func WithRetryPolicy(policy resilience.RetryPolicy) OrderEventsConsumerOption {
	return func(c *OrderEventsConsumer) {
		c.retryPolicy = resilience.NewRetryPolicy(policy)
	}
}

func NewOrderEventsConsumer(
	brokers []string,
	topic string,
	groupID string,
	paymentService *application.PaymentService,
	repository application.PaymentRepository,
	defaultAmount int64,
	defaultCurrency string,
	options ...OrderEventsConsumerOption,
) *OrderEventsConsumer {
	if groupID == "" {
		groupID = "payment-service"
	}
	if defaultAmount <= 0 {
		defaultAmount = 1000
	}
	if strings.TrimSpace(defaultCurrency) == "" {
		defaultCurrency = "USD"
	}

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  brokers,
		Topic:    topic,
		GroupID:  groupID,
		MinBytes: 1,
		MaxBytes: 10e6,
	})

	consumer := &OrderEventsConsumer{
		reader:          reader,
		paymentService:  paymentService,
		repository:      repository,
		defaultAmount:   defaultAmount,
		defaultCurrency: defaultCurrency,
		retryPolicy:     resilience.NewRetryPolicy(resilience.RetryPolicy{}),
	}
	for _, option := range options {
		option(consumer)
	}

	return consumer
}

func (c *OrderEventsConsumer) Run(ctx context.Context) {
	for {
		message, err := c.reader.FetchMessage(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return
			}
			log.Printf("payment consumer fetch failed: %v", err)
			continue
		}

		if err := c.HandleMessageWithRetry(ctx, message); err != nil {
			log.Printf("payment consumer handle failed: %v", err)
		}

		if err := c.reader.CommitMessages(ctx, message); err != nil {
			log.Printf("payment consumer commit failed: %v", err)
		}
	}
}

func (c *OrderEventsConsumer) HandleMessageWithRetry(ctx context.Context, message kafka.Message) error {
	return c.retryPolicy.Do(ctx, func(ctx context.Context) error {
		return c.HandleMessage(ctx, message)
	})
}

func (c *OrderEventsConsumer) HandleMessage(ctx context.Context, message kafka.Message) error {
	eventName := headerValue(message.Headers, "event_name")
	switch eventName {
	case "OrderPlaced":
		return c.handleOrderPlaced(ctx, message.Value)
	case "OrderCancelled":
		return c.handleOrderCancelled(ctx, message.Value)
	default:
		// Unknown events are ignored to keep consumer forward-compatible.
		return nil
	}
}

func (c *OrderEventsConsumer) Close() error {
	if c == nil || c.reader == nil {
		return nil
	}
	return c.reader.Close()
}

func (c *OrderEventsConsumer) handleOrderPlaced(ctx context.Context, payload []byte) error {
	var event OrderPlacedEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		return fmt.Errorf("unmarshal OrderPlaced: %w", err)
	}

	payment, err := c.repository.GetByOrderID(ctx, event.OrderID)
	if err != nil {
		if !errors.Is(err, application.ErrPaymentNotFound) {
			return err
		}

		paymentID, createErr := c.paymentService.Create(ctx, &application.CreatePaymentDTO{
			OrderID:  event.OrderID,
			Amount:   c.defaultAmount,
			Currency: c.defaultCurrency,
		})
		if createErr != nil {
			return createErr
		}

		if err := c.paymentService.MarkPaid(ctx, paymentID); err != nil {
			return err
		}

		payment, err = c.repository.GetByID(ctx, paymentID)
		if err != nil {
			return err
		}
	}

	if payment.Status() == domain.PaymentStatusPending {
		if err := c.paymentService.MarkPaid(ctx, payment.ID()); err != nil {
			return err
		}
		payment, err = c.repository.GetByID(ctx, payment.ID())
		if err != nil {
			return err
		}
	}

	if payment.Status() == domain.PaymentStatusFailed {
		return nil
	}

	return nil
}

func (c *OrderEventsConsumer) handleOrderCancelled(ctx context.Context, payload []byte) error {
	var event OrderCancelledEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		return fmt.Errorf("unmarshal OrderCancelled: %w", err)
	}

	payment, err := c.repository.GetByOrderID(ctx, event.OrderID)
	if err != nil {
		if errors.Is(err, application.ErrPaymentNotFound) {
			return nil
		}
		return err
	}

	switch payment.Status() {
	case domain.PaymentStatusPaid:
		if err := c.paymentService.Refund(ctx, payment.ID()); err != nil {
			return err
		}
		return nil
	case domain.PaymentStatusPending:
		if err := c.paymentService.MarkFailed(ctx, payment.ID(), "order_cancelled"); err != nil {
			return err
		}
		return nil
	default:
		return nil
	}
}

func headerValue(headers []kafka.Header, key string) string {
	for _, header := range headers {
		if strings.EqualFold(header.Key, key) {
			return string(header.Value)
		}
	}
	return ""
}
