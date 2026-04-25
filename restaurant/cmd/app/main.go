package main

import (
	"log"
	"net/http"

	httpinfra "github.com/illia-malachyn/food-delivery/restaurant/infrastructure/http"
)

func main() {
	log.Println("restaurant microservice started")

	if err := http.ListenAndServe(":8080", httpinfra.NewRouter()); err != nil {
		log.Fatalf("restaurant server stopped: %v", err)
	}
}
