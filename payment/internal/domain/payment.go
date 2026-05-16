package domain

import "context"

type PaymentStatus string

const (
	PaymentStatusAuthorized PaymentStatus = "Authorized"
	PaymentStatusDeclined   PaymentStatus = "Declined"
)

type Payment struct {
	ID            string        `json:"id"`
	OrderID       string        `json:"order_id"`
	TransactionID string        `json:"transaction_id,omitempty"`
	Amount        int64         `json:"amount"`
	Status        PaymentStatus `json:"status"`
}

type PaymentRepository interface {
	Create(ctx context.Context, payment *Payment) error
	GetByOrderID(ctx context.Context, orderID string) (*Payment, error)
	ListByStatus(ctx context.Context, status string) ([]*Payment, error)
}
