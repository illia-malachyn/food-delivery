package http

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/illia-malachyn/food-delivery/payment/application"
	"github.com/illia-malachyn/food-delivery/payment/domain"
)

type errorResponse struct {
	Error string `json:"error"`
}

func writeJSON(w http.ResponseWriter, statusCode int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, statusCode int, err error) {
	writeJSON(w, statusCode, errorResponse{Error: err.Error()})
}

func writeDomainError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, application.ErrPaymentNotFound):
		writeError(w, http.StatusNotFound, err)
	case errors.Is(err, application.ErrCreatePaymentDTORequired):
		writeError(w, http.StatusBadRequest, err)
	case errors.Is(err, domain.ErrValidationFailed), errors.Is(err, domain.ErrInvalidStateTransition):
		writeError(w, http.StatusBadRequest, err)
	default:
		writeError(w, http.StatusInternalServerError, err)
	}
}
