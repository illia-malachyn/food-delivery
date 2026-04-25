package main

import (
	"log"
	"net/http"

	httpinfra "github.com/illia-malachyn/food-delivery/delivery/infrastructure/http"
)

func main() {
	log.Println("delivery microservice started")

	if err := http.ListenAndServe(":8080", httpinfra.NewRouter()); err != nil {
		log.Fatalf("delivery server stopped: %v", err)
	}
}
