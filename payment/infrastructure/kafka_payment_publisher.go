package infrastructure

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"

	sharedkafka "github.com/illia-malachyn/food-delivery/shared/kafka"
)

type paymentIntegrationEvent interface {
	EventName() string
}

type KafkaPaymentEventPublisher struct {
	writer *kafka.Writer
}

func NewKafkaPaymentEventPublisher(brokers []string, topic string) (*KafkaPaymentEventPublisher, error) {
	writer, err := sharedkafka.NewWriter(brokers, topic)
	if err != nil {
		return nil, err
	}
	return &KafkaPaymentEventPublisher{writer: writer}, nil
}

func (p *KafkaPaymentEventPublisher) Publish(ctx context.Context, event paymentIntegrationEvent) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal payment integration event %s: %w", event.EventName(), err)
	}

	message := kafka.Message{
		Value: payload,
		Time:  time.Now().UTC(),
		Headers: []kafka.Header{
			{Key: "event_name", Value: []byte(event.EventName())},
		},
	}

	if err := p.writer.WriteMessages(ctx, message); err != nil {
		return fmt.Errorf("write payment integration event %s: %w", event.EventName(), err)
	}

	return nil
}

func (p *KafkaPaymentEventPublisher) Close() error {
	if p == nil || p.writer == nil {
		return nil
	}
	return p.writer.Close()
}
