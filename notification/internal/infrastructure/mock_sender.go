package infrastructure

import (
	"context"
	"errors"
	"log"
	"math/rand"
	"time"

	"order/notification/internal/domain"
)

type MockSender struct{}

func NewMockSender() domain.EmailSender {
	return &MockSender{}
}

func (s *MockSender) Send(ctx context.Context, job domain.EmailJob) error {
	// Simulate network delay
	delay := time.Duration(rand.Intn(500)) * time.Millisecond
	time.Sleep(delay)

	// Simulate random failure (30% chance)
	if rand.Float32() < 0.3 {
		log.Printf("[MockSender] Simulated failure sending to %s", job.To)
		return errors.New("simulated email provider failure")
	}

	log.Printf("[MockSender] Successfully sent email to %s (Subject: %s)", job.To, job.Subject)
	return nil
}
