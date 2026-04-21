package main

import (
	"log"
	"net/http"
)

func main() {
	log.Println("payment microservice started")

	http.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("hello from payment service"))
	})

	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("payment server stopped: %v", err)
	}
}
