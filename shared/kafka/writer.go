package kafka

import (
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"
)

func NewWriter(brokers []string, topic string) (*kafka.Writer, error) {
	if len(brokers) == 0 {
		return nil, fmt.Errorf("kafka brokers are required")
	}
	if topic == "" {
		return nil, fmt.Errorf("kafka topic is required")
	}

	return &kafka.Writer{
		Addr:         kafka.TCP(brokers...),
		Topic:        topic,
		RequiredAcks: kafka.RequireOne,
		Balancer:     &kafka.Hash{},
		BatchTimeout: 10 * time.Millisecond,
	}, nil
}
