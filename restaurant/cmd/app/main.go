package main

import (
	"log"
	"net/http"
)

func main() {
	log.Println("restaurant microservice started")

	http.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("hello from restaurant service"))
	})

	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("restaurant server stopped: %v", err)
	}
}
