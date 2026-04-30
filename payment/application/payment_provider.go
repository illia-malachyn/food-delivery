package application

import "context"

type PaymentProvider interface {
	Capture(ctx context.Context, paymentID string, amount int64, currency string) error
	Refund(ctx context.Context, paymentID string) error
}

type noopPaymentProvider struct{}

func NewNoopPaymentProvider() PaymentProvider {
	return noopPaymentProvider{}
}

func (noopPaymentProvider) Capture(context.Context, string, int64, string) error {
	return nil
}

func (noopPaymentProvider) Refund(context.Context, string) error {
	return nil
}
