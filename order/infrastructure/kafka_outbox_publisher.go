package infrastructure

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/segmentio/kafka-go"
)

// KafkaOutboxPublisher sends outbox events to Kafka.
type KafkaOutboxPublisher struct {
	writer *kafka.Writer
}

func NewKafkaOutboxPublisher(brokers []string, topic string) (*KafkaOutboxPublisher, error) {
	if len(brokers) == 0 {
		return nil, fmt.Errorf("kafka brokers are required")
	}

	if topic == "" {
		return nil, fmt.Errorf("kafka topic is required")
	}

	return &KafkaOutboxPublisher{
		writer: &kafka.Writer{
			Addr:         kafka.TCP(brokers...),
			Topic:        topic,
			RequiredAcks: kafka.RequireOne,
			Balancer:     &kafka.Hash{},
			BatchTimeout: 10 * time.Millisecond,
		},
	}, nil
}

func (p *KafkaOutboxPublisher) Publish(ctx context.Context, event OutboxEvent) error {
	message := kafka.Message{
		Key:   []byte(event.AggregateID),
		Value: event.Payload,
		Time:  event.OccurredAt,
		Headers: []kafka.Header{
			{Key: "event_name", Value: []byte(event.EventName)},
			{Key: "event_version", Value: []byte(strconv.Itoa(event.EventVersion))},
			{Key: "aggregate_type", Value: []byte(event.AggregateType)},
			{Key: "aggregate_id", Value: []byte(event.AggregateID)},
			{Key: "occurred_at", Value: []byte(event.OccurredAt.UTC().Format(time.RFC3339Nano))},
		},
	}

	if err := p.writer.WriteMessages(ctx, message); err != nil {
		return fmt.Errorf("write kafka message: %w", err)
	}

	return nil
}

func (p *KafkaOutboxPublisher) Close() error {
	if p == nil || p.writer == nil {
		return nil
	}
	return p.writer.Close()
}
