package usecase

import (
	"context"
	"fmt"
	"log"
	"math"
	"time"

	"order/notification/internal/domain"

	"github.com/redis/go-redis/v9"
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
	sender      domain.EmailSender
	redisClient *redis.Client
	maxRetries  int
}

func NewNotificationUseCase(sender domain.EmailSender, redisClient *redis.Client, maxRetries int) NotificationUseCase {
	return &notificationUseCase{
		sender:      sender,
		redisClient: redisClient,
		maxRetries:  maxRetries,
	}
}

func (u *notificationUseCase) ProcessPaymentCompleted(ctx context.Context, event NotificationEvent) error {
	// Idempotency check before side effects
	idemKey := fmt.Sprintf("processed:%s", event.MessageID) // Using message_id as payment_id logic

	// Try to set the key. If it exists, it means we already processed it.
	// TTL is set to 24 hours to prevent duplicates during delayed retries.
	isNew, err := u.redisClient.SetNX(ctx, idemKey, "processing", 24*time.Hour).Result()
	if err != nil {
		log.Printf("[Notification] Error checking idempotency key in Redis: %v", err)
		// Fail open for idempotency check? Usually we want to fail closed if we can't guarantee exactly-once,
		// but depending on business rules we might proceed. For now, we will return err to retry later.
		return fmt.Errorf("failed to check idempotency: %w", err)
	}

	if !isNew {
		log.Printf("[Notification] Message %s already processed. Skipping.", event.MessageID)
		return nil // Return nil so it can be ACKed
	}

	amountStr := fmt.Sprintf("$%.2f", float64(event.Amount)/100.0) // Assuming amount is in cents

	job := domain.EmailJob{
		To:      event.CustomerEmail,
		Subject: fmt.Sprintf("Payment Completed for Order #%s", event.OrderID),
		Body:    fmt.Sprintf("Your payment of %s has been successfully processed.", amountStr),
	}

	// Exponential Backoff
	var lastErr error
	for attempt := 0; attempt <= u.maxRetries; attempt++ {
		if attempt > 0 {
			// delay = initial_delay * 2^(attempt-1). Start with 2s, 4s, 8s...
			delay := time.Duration(2*math.Pow(2, float64(attempt-1))) * time.Second
			log.Printf("[Notification] Attempt %d failed. Retrying in %v...", attempt, delay)
			time.Sleep(delay)
		}

		err := u.sender.Send(ctx, job)
		if err == nil {
			log.Printf("[Notification] Successfully processed and sent email for MessageID %s", event.MessageID)

			// Mark as permanently done by updating TTL or just leave the 24h TTL from SetNX.
			// The key is already there, so we are good.
			return nil
		}

		lastErr = err
	}

	// If we exhausted retries, we failed.
	// We need to delete the idempotency key so that it can be retried from the DLQ later if needed,
	// or we can leave it to prevent infinite loops. But usually, if processing failed, we remove the lock.
	u.redisClient.Del(ctx, idemKey)

	return fmt.Errorf("failed to send email after %d attempts: %w", u.maxRetries+1, lastErr)
}
