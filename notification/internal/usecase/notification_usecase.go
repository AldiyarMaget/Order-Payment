package usecase

import (
	"context"
	"fmt"
	"log"
	"sync"
)

type NotificationEvent struct {
	OrderID       string `json:"order_id"`
	Amount        int64  `json:"amount"`
	CustomerEmail string `json:"customer_email"`
	Status        string `json:"status"`
	MessageID     string `json:"message_id"`
}

type NotificationUseCase interface {
	ProcessPaymentCompleted(ctx context.Context, event NotificationEvent) error
}

type notificationUseCase struct {
	processedMessages sync.Map
}

func NewNotificationUseCase() NotificationUseCase {
	return &notificationUseCase{}
}

func (u *notificationUseCase) checkIdempotency(messageID string) bool {
	_, loaded := u.processedMessages.LoadOrStore(messageID, true)
	return loaded
}

func (u *notificationUseCase) ProcessPaymentCompleted(ctx context.Context, event NotificationEvent) error {
	// Idempotency check before side effects
	if u.checkIdempotency(event.MessageID) {
		log.Printf("[Notification] Message %s already processed. Skipping.", event.MessageID)
		return nil // Return nil so it can be ACKed
	}

	// Simulate business logic (sending email)
	amountStr := fmt.Sprintf("$%.2f", float64(event.Amount)/100.0) // Assuming amount is in cents
	fmt.Printf("[Notification] Sent email to %s for Order #%s. Amount: %s\n", event.CustomerEmail, event.OrderID, amountStr)

	return nil
}
