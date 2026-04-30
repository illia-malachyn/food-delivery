package main

import (
	"log"
	"net/http"

	httpinfra "github.com/illia-malachyn/food-delivery/restaurant/infrastructure/http"
	sharedconfig "github.com/illia-malachyn/food-delivery/shared/config"
	sharedmiddleware "github.com/illia-malachyn/food-delivery/shared/http/middleware"
	sharedjwt "github.com/illia-malachyn/food-delivery/shared/security/jwt"
)

func main() {
	log.Println("restaurant microservice started")

	jwtVerifier, err := sharedjwt.NewVerifier(
		sharedconfig.JWTPublicKeyFromEnv("restaurant"),
		sharedconfig.JWTIssuerFromEnv("restaurant"),
	)
	if err != nil {
		log.Fatalf("cannot initialize JWT verifier: %v", err)
	}

	if err := http.ListenAndServe(":8080", httpinfra.NewRouter(sharedmiddleware.RequireJWT(jwtVerifier))); err != nil {
		log.Fatalf("restaurant server stopped: %v", err)
	}
}
