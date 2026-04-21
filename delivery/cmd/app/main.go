package main

import (
	"log"
	"net/http"
)

func main() {
	log.Println("delivery microservice started")

	http.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("hello from delivery service"))
	})

	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("delivery server stopped: %v", err)
	}
}
