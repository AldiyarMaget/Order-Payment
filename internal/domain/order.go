package domain

import (
	"context"
	"time"
)

type OrderStatus string

const (
	StatusPending   OrderStatus = "Pending"
	StatusPaid      OrderStatus = "Paid"
	StatusFailed    OrderStatus = "Failed"
	StatusCancelled OrderStatus = "Cancelled"
)

type Order struct {
	ID             string      `json:"id"`
	CustomerID     string      `json:"customer_id"`
	ItemName       string      `json:"item_name"`
	Amount         int64       `json:"amount"`
	Status         OrderStatus `json:"status"`
	CreatedAt      time.Time   `json:"created_at"`
	IdempotencyKey string      `json:"idempotency_key"`
}

type OrderRepository interface {
	Create(ctx context.Context, order *Order) error
	GetByID(ctx context.Context, id string) (*Order, error)
	GetByIdempotencyKey(ctx context.Context, key string) (*Order, error)
	UpdateStatus(ctx context.Context, id string, status OrderStatus) error
	GetByAmountRange(ctx context.Context, minAmount, maxAmount int64) ([]Order, error)
}

type PaymentClient interface {
	AuthorizePayment(ctx context.Context, orderID string, amount int64) (string, error)
}
