package httpapi

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/illia-malachyn/food-delivery/auth/internal/auth"
	"github.com/illia-malachyn/food-delivery/auth/internal/config"
	"github.com/illia-malachyn/food-delivery/auth/internal/httpapi/cookies"
	"github.com/illia-malachyn/food-delivery/auth/internal/httpapi/httputil"
	"github.com/illia-malachyn/food-delivery/auth/internal/httpapi/middleware"
)

type Handler struct {
	authService    *auth.Service
	cookieConfig   cookies.Config
	refreshTTL     time.Duration
	requestTimeout time.Duration
}

type credentialsRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type tokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int64  `json:"expires_in"`
}

func NewHandler(
	authService *auth.Service,
	cookieConfig config.CookieConfig,
	refreshTTL time.Duration,
	requestTimeout time.Duration,
) *Handler {
	return &Handler{
		authService: authService,
		cookieConfig: cookies.Config{
			Name:     cookieConfig.Name,
			Domain:   cookieConfig.Domain,
			Path:     cookieConfig.Path,
			Secure:   cookieConfig.Secure,
			HTTPOnly: cookieConfig.HTTPOnly,
			SameSite: cookieConfig.SameSite,
		},
		refreshTTL:     refreshTTL,
		requestTimeout: requestTimeout,
	}
}

func (h *Handler) HandleHealth(w http.ResponseWriter, _ *http.Request) {
	httputil.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) HandleRegister(w http.ResponseWriter, r *http.Request) {
	var req credentialsRequest
	if err := httputil.DecodeJSON(r, &req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), h.requestTimeout)
	defer cancel()

	result, err := h.authService.Register(ctx, req.Email, req.Password)
	if err != nil {
		h.writeAuthError(w, err)
		return
	}

	cookies.SetRefreshToken(w, h.cookieConfig, result.RefreshToken, h.refreshTTL)
	httputil.WriteJSON(w, http.StatusCreated, tokenResponseFromResult(result))
}

func (h *Handler) HandleLogin(w http.ResponseWriter, r *http.Request) {
	var req credentialsRequest
	if err := httputil.DecodeJSON(r, &req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), h.requestTimeout)
	defer cancel()

	result, err := h.authService.Login(ctx, req.Email, req.Password)
	if err != nil {
		h.writeAuthError(w, err)
		return
	}

	cookies.SetRefreshToken(w, h.cookieConfig, result.RefreshToken, h.refreshTTL)
	httputil.WriteJSON(w, http.StatusOK, tokenResponseFromResult(result))
}

func (h *Handler) HandleRefresh(w http.ResponseWriter, r *http.Request) {
	refreshToken, err := cookies.ReadRefreshToken(r, h.cookieConfig)
	if err != nil {
		httputil.WriteError(w, http.StatusUnauthorized, "refresh cookie is missing")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), h.requestTimeout)
	defer cancel()

	result, err := h.authService.Refresh(ctx, refreshToken)
	if err != nil {
		h.writeAuthError(w, err)
		return
	}

	cookies.SetRefreshToken(w, h.cookieConfig, result.RefreshToken, h.refreshTTL)
	httputil.WriteJSON(w, http.StatusOK, tokenResponseFromResult(result))
}

func (h *Handler) HandleLogout(w http.ResponseWriter, r *http.Request) {
	refreshToken, err := cookies.ReadRefreshToken(r, h.cookieConfig)
	if err != nil {
		cookies.ClearRefreshToken(w, h.cookieConfig)
		w.WriteHeader(http.StatusNoContent)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), h.requestTimeout)
	defer cancel()

	if err := h.authService.Logout(ctx, refreshToken); err != nil && !errors.Is(err, auth.ErrInvalidRefreshToken) {
		h.writeAuthError(w, err)
		return
	}

	cookies.ClearRefreshToken(w, h.cookieConfig)
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) HandleMe(w http.ResponseWriter, r *http.Request) {
	principal, ok := middleware.PrincipalFromContext(r.Context())
	if !ok {
		httputil.WriteError(w, http.StatusUnauthorized, "missing auth principal")
		return
	}

	httputil.WriteJSON(w, http.StatusOK, map[string]any{
		"user_id": principal.UserID,
		"email":   principal.Email,
	})
}

func (h *Handler) writeAuthError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, auth.ErrInvalidEmail), errors.Is(err, auth.ErrInvalidPassword):
		httputil.WriteError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, auth.ErrEmailAlreadyExists):
		httputil.WriteError(w, http.StatusConflict, err.Error())
	case errors.Is(err, auth.ErrInvalidCredentials):
		httputil.WriteError(w, http.StatusUnauthorized, err.Error())
	case errors.Is(err, auth.ErrInvalidRefreshToken), errors.Is(err, auth.ErrRefreshTokenRevoked):
		httputil.WriteError(w, http.StatusUnauthorized, err.Error())
	default:
		httputil.WriteError(w, http.StatusInternalServerError, "internal server error")
	}
}

func tokenResponseFromResult(result auth.AuthResult) tokenResponse {
	return tokenResponse{
		AccessToken: result.AccessToken,
		TokenType:   result.TokenType,
		ExpiresIn:   result.ExpiresIn,
	}
}
