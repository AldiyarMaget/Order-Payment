package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"order/notification/internal/delivery"
	"order/notification/internal/domain"
	"order/notification/internal/infrastructure"
	"order/notification/internal/usecase"

	"github.com/redis/go-redis/v9"
)

func main() {
	log.Println("Starting Notification Service...")

	rmqURL := os.Getenv("RMQ_URL")
	if rmqURL == "" {
		rmqURL = "amqp://guest:guest@localhost:5672/"
	}

	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}

	redisClient := redis.NewClient(&redis.Options{
		Addr: redisAddr,
	})
	if err := redisClient.Ping(context.Background()).Err(); err != nil {
		log.Fatalf("failed to connect to Redis: %v", err)
	}

	var sender domain.EmailSender
	providerMode := os.Getenv("PROVIDER_MODE")
	if providerMode == "REAL" {
		host := os.Getenv("SMTP_HOST")
		port := os.Getenv("SMTP_PORT")
		user := os.Getenv("SMTP_USER")
		pass := os.Getenv("SMTP_PASSWORD")
		sender = infrastructure.NewSMTPSender(host, port, user, pass)
	} else {
		sender = infrastructure.NewMockSender()
	}

	maxRetriesStr := os.Getenv("MAX_RETRIES")
	maxRetries, err := strconv.Atoi(maxRetriesStr)
	if err != nil || maxRetries <= 0 {
		maxRetries = 3
	}

	uc := usecase.NewNotificationUseCase(sender, redisClient, maxRetries)
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
