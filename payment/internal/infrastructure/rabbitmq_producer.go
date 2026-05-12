package infrastructure

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type rabbitMQBroker struct {
	conn *amqp.Connection
	ch   *amqp.Channel
}

func NewRabbitMQBroker(url string) (MessageBroker, error) {
	var conn *amqp.Connection
	var err error

	for i := 0; i < 10; i++ {
		conn, err = amqp.Dial(url)
		if err == nil {
			break
		}

		backoff := time.Duration(2+i) * time.Second
		if backoff > 5*time.Second {
			backoff = 5 * time.Second
		}

		log.Printf("Failed to connect to RabbitMQ (attempt %d/10): %v. Retrying in %v...", i+1, err, backoff)
		time.Sleep(backoff)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ after retries: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to open a channel: %w", err)
	}

	// Ensure the queue exists
	args := amqp.Table{
		"x-dead-letter-exchange":    "dlx.payment.completed",
		"x-dead-letter-routing-key": "payment.completed",
	}

	_, err = ch.QueueDeclare(
		"payment.completed", // name
		true,                // durable
		false,               // delete when unused
		false,               // exclusive
		false,               // no-wait
		args,                // arguments
	)
	if err != nil {
		return nil, fmt.Errorf("failed to declare a queue: %w", err)
	}

	return &rabbitMQBroker{
		conn: conn,
		ch:   ch,
	}, nil
}

func (b *rabbitMQBroker) PublishPaymentCompleted(ctx context.Context, event PaymentEvent) error {
	body, err := json.Marshal(event)
	if err != nil {
		return err
	}

	err = b.ch.PublishWithContext(ctx,
		"",                  // exchange
		"payment.completed", // routing key
		false,               // mandatory
		false,               // immediate
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         body,
			DeliveryMode: amqp.Persistent,
			MessageId:    event.MessageID,
		})
	if err != nil {
		return fmt.Errorf("failed to publish a message: %w", err)
	}

	log.Printf("[Payment] Published payment.completed event for Order %s", event.OrderID)
	return nil
}

func (b *rabbitMQBroker) Close() error {
	if b.ch != nil {
		_ = b.ch.Close()
	}
	if b.conn != nil {
		return b.conn.Close()
	}
	return nil
}
