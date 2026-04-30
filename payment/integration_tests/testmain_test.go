//go:build integration

package payment_test

import (
	"context"
	"log"
	"os"
	"testing"

	kafkamodule "github.com/testcontainers/testcontainers-go/modules/kafka"
)

var integrationKafkaContainer *kafkamodule.KafkaContainer
var integrationKafkaBrokers []string

func TestMain(m *testing.M) {
	ctx := context.Background()

	container, err := kafkamodule.RunContainer(ctx)
	if err != nil {
		log.Printf("failed to start kafka container: %v", err)
		os.Exit(1)
	}
	integrationKafkaContainer = container

	brokers, err := container.Brokers(ctx)
	if err != nil || len(brokers) == 0 {
		log.Printf("failed to get kafka brokers: %v", err)
		_ = container.Terminate(context.Background())
		os.Exit(1)
	}
	integrationKafkaBrokers = brokers

	exitCode := m.Run()

	if terminateErr := container.Terminate(context.Background()); terminateErr != nil {
		log.Printf("failed to terminate kafka container: %v", terminateErr)
	}

	os.Exit(exitCode)
}
