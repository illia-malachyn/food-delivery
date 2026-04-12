package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/illia-malachyn/microservices/order/application"
	httpinfra "github.com/illia-malachyn/microservices/order/infrastructure/http"
	"github.com/illia-malachyn/microservices/order/infrastructure/persistence"
)

func main() {
	log.Println("order microservice started")
	ctx := context.Background()

	connPool, err := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatal(err)
	}

	postgresOrderRepository := persistence.NewPostgresOrderRepository(connPool)
	//inMemoryOrderRepository := persistence.NewInMemoryOrderRepository()
	orderService := application.NewOrderService(postgresOrderRepository)
	orderHandler := httpinfra.CreateOrderHandler(orderService)

	http.Handle("/orders", orderHandler)
	if err := http.ListenAndServe(":9876", nil); err != nil {
		log.Fatal(err)
	}
}
