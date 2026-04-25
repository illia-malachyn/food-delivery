package main

import (
	"log"
	"net/http"

	httpinfra "github.com/illia-malachyn/food-delivery/payment/infrastructure/http"
)

func main() {
	log.Println("payment microservice started")

	if err := http.ListenAndServe(":8080", httpinfra.NewRouter()); err != nil {
		log.Fatalf("payment server stopped: %v", err)
	}
}
