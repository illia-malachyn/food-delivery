package main

import (
	"log"
	"net/http"

	httpinfra "github.com/illia-malachyn/food-delivery/delivery/infrastructure/http"
	sharedconfig "github.com/illia-malachyn/food-delivery/shared/config"
	sharedmiddleware "github.com/illia-malachyn/food-delivery/shared/http/middleware"
	sharedjwt "github.com/illia-malachyn/food-delivery/shared/security/jwt"
)

func main() {
	log.Println("delivery microservice started")

	jwtVerifier, err := sharedjwt.NewVerifier(
		sharedconfig.JWTPublicKeyFromEnv("delivery"),
		sharedconfig.JWTIssuerFromEnv("delivery"),
	)
	if err != nil {
		log.Fatalf("cannot initialize JWT verifier: %v", err)
	}

	if err := http.ListenAndServe(":8080", httpinfra.NewRouter(sharedmiddleware.RequireJWT(jwtVerifier))); err != nil {
		log.Fatalf("delivery server stopped: %v", err)
	}
}
