package infrastructure

import "context"

type PaymentEvent struct {
	OrderID       string `json:"order_id"`
	Amount        int64  `json:"amount"`
	CustomerEmail string `json:"customer_email"`
	Status        string `json:"status"`
	MessageID     string `json:"message_id"`
}

type MessageBroker interface {
	PublishPaymentCompleted(ctx context.Context, event PaymentEvent) error
	Close() error
}
