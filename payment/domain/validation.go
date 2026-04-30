package domain

import (
	"regexp"
	"strings"
	"time"
)

var currencyCodeRegex = regexp.MustCompile(`^[A-Z]{3}$`)

func validateNewPaymentInput(orderID string, amount int64, currency string) (string, string, error) {
	trimmedOrderID := strings.TrimSpace(orderID)
	trimmedCurrency := strings.TrimSpace(currency)

	if trimmedOrderID == "" || amount <= 0 || !isValidCurrency(trimmedCurrency) {
		return "", "", ErrValidationFailed
	}

	return trimmedOrderID, trimmedCurrency, nil
}

func validateReconstructedPayment(
	id string,
	orderID string,
	amount int64,
	currency string,
	status PaymentStatus,
	failureReason string,
	createdAt time.Time,
) (string, string, string, string, error) {
	trimmedID := strings.TrimSpace(id)
	trimmedOrderID := strings.TrimSpace(orderID)
	trimmedCurrency := strings.TrimSpace(currency)
	trimmedFailureReason := strings.TrimSpace(failureReason)

	if trimmedID == "" || trimmedOrderID == "" || amount <= 0 || !isValidCurrency(trimmedCurrency) || !status.IsValid() || createdAt.IsZero() {
		return "", "", "", "", ErrValidationFailed
	}
	if status == PaymentStatusFailed && trimmedFailureReason == "" {
		return "", "", "", "", ErrValidationFailed
	}
	if status != PaymentStatusFailed && trimmedFailureReason != "" {
		return "", "", "", "", ErrValidationFailed
	}

	return trimmedID, trimmedOrderID, trimmedCurrency, trimmedFailureReason, nil
}

func validateFailureReason(reason string) (string, error) {
	trimmedReason := strings.TrimSpace(reason)
	if trimmedReason == "" {
		return "", ErrValidationFailed
	}
	return trimmedReason, nil
}

func isValidCurrency(currency string) bool {
	return currencyCodeRegex.MatchString(currency)
}
