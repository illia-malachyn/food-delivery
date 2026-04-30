package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type HTTPPaymentProvider struct {
	baseURL string
	client  *http.Client
}

func NewHTTPPaymentProvider(baseURL string, client *http.Client) *HTTPPaymentProvider {
	if client == nil {
		client = http.DefaultClient
	}
	return &HTTPPaymentProvider{
		baseURL: strings.TrimRight(baseURL, "/"),
		client:  client,
	}
}

func (p *HTTPPaymentProvider) Capture(ctx context.Context, paymentID string, amount int64, currency string) error {
	payload := map[string]any{
		"payment_id": paymentID,
		"amount":     amount,
		"currency":   currency,
	}
	return p.post(ctx, "/capture", payload)
}

func (p *HTTPPaymentProvider) Refund(ctx context.Context, paymentID string) error {
	payload := map[string]any{"payment_id": paymentID}
	return p.post(ctx, "/refund", payload)
}

func (p *HTTPPaymentProvider) post(ctx context.Context, path string, payload any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payment provider request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create payment provider request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("call payment provider: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		responseBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("payment provider returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(responseBody)))
	}

	return nil
}
