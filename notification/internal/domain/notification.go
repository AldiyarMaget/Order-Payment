package domain

import (
	"context"
)

type OrderStatus string

type PaymentEvent struct {
	OrderID       string      `json:"order_id"`
	Amount        int64       `json:"amount"`
	CustomerEmail string      `json:"customer_email"`
	Status        OrderStatus `json:"status"`
}

type NotificationRepository interface {
	CheckAndSave(orderID string) (bool, error)
}

type NotificationUsecase interface {
	OnPaymentCompleted(ctx context.Context, event PaymentEvent) error
}
