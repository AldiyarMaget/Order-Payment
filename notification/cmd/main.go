package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"order/notification/internal/delivery"
	"order/notification/internal/usecase"
)

func main() {
	log.Println("Starting Notification Service...")

	rmqURL := os.Getenv("RMQ_URL")
	if rmqURL == "" {
		rmqURL = "amqp://guest:guest@localhost:5672/"
	}

	uc := usecase.NewNotificationUseCase()
	consumer, err := delivery.NewRabbitMQConsumer(rmqURL, uc)
	if err != nil {
		log.Fatalf("failed to initialize rabbitmq consumer: %v", err)
	}
	defer consumer.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := consumer.Start(ctx); err != nil {
		log.Fatalf("failed to start consumer: %v", err)
	}

	// Graceful Shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down gracefully...")
	cancel() // This stops the consumer loop
	consumer.Close()
	log.Println("Notification service stopped cleanly.")
}
