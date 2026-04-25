package httpapi

import (
	"net/http"

	"github.com/illia-malachyn/food-delivery/auth/internal/httpapi/middleware"
	"github.com/illia-malachyn/food-delivery/auth/internal/security"
)

func NewRouter(handler *Handler, tokenManager security.TokenManager) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", handler.HandleHealth)
	mux.HandleFunc("POST /auth/register", handler.HandleRegister)
	mux.HandleFunc("POST /auth/login", handler.HandleLogin)
	mux.HandleFunc("POST /auth/refresh", handler.HandleRefresh)
	mux.HandleFunc("POST /auth/logout", handler.HandleLogout)
	mux.Handle("GET /auth/me", middleware.Chain(
		http.HandlerFunc(handler.HandleMe),
		middleware.RequireAuth(tokenManager),
	))

	return middleware.Chain(
		mux,
		middleware.Recovery(),
		middleware.Logging(),
		middleware.Tracing(),
		middleware.Metrics(),
	)
}
