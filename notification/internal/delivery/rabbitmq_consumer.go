package delivery

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"order/notification/internal/usecase"

	amqp "github.com/rabbitmq/amqp091-go"
)

type RabbitMQConsumer struct {
	conn    *amqp.Connection
	ch      *amqp.Channel
	usecase usecase.NotificationUseCase
}

func NewRabbitMQConsumer(url string, uc usecase.NotificationUseCase) (*RabbitMQConsumer, error) {
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

	// 1. Declare DLX
	err = ch.ExchangeDeclare(
		"dlx.payment.completed", // name
		"direct",                // type
		true,                    // durable
		false,                   // auto-deleted
		false,                   // internal
		false,                   // no-wait
		nil,                     // args
	)
	if err != nil {
		return nil, fmt.Errorf("failed to declare DLX: %w", err)
	}

	// 2. Declare DLQ
	_, err = ch.QueueDeclare(
		"dlq.payment.completed",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to declare DLQ: %w", err)
	}

	// 3. Bind DLQ to DLX
	err = ch.QueueBind(
		"dlq.payment.completed", // queue name
		"payment.completed",     // routing key (same as original or specific)
		"dlx.payment.completed", // exchange
		false,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to bind DLQ: %w", err)
	}

	// 4. Declare Main Queue with DLX configuration
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
		return nil, fmt.Errorf("failed to declare main queue: %w", err)
	}

	return &RabbitMQConsumer{
		conn:    conn,
		ch:      ch,
		usecase: uc,
	}, nil
}

func (c *RabbitMQConsumer) Start(ctx context.Context) error {
	msgs, err := c.ch.Consume(
		"payment.completed", // queue
		"",                  // consumer
		false,               // auto-ack = false
		false,               // exclusive
		false,               // no-local
		false,               // no-wait
		nil,                 // args
	)
	if err != nil {
		return fmt.Errorf("failed to register a consumer: %w", err)
	}

	log.Println("[Notification] Consumer started, waiting for messages...")

	go func() {
		sem := make(chan struct{}, 2)
		var wg sync.WaitGroup

		for {
			select {
			case <-ctx.Done():
				log.Println("[Notification] Context cancelled, waiting for workers to finish...")
				wg.Wait()
				log.Println("[Notification] All workers stopped.")
				return
			case msg, ok := <-msgs:
				if !ok {
					return
				}

				sem <- struct{}{}
				wg.Add(1)

				go func(m amqp.Delivery) {
					defer wg.Done()
					defer func() { <-sem }()

					workerID := time.Now().UnixNano() % 10000
					ts := time.Now().Format("15:04:05.000")

					var event usecase.NotificationEvent
					if err := json.Unmarshal(m.Body, &event); err != nil {
						log.Printf("[%s] [Worker-%d] Failed to unmarshal message: %v. Sending NACK.", ts, workerID, err)
						m.Nack(false, false)
						return
					}

					log.Printf("[%s] [Worker-%d] Started processing MessageID: %s", ts, workerID, event.MessageID)

					err := c.usecase.ProcessPaymentCompleted(ctx, event)
					
					ts = time.Now().Format("15:04:05.000")
					if err == nil {
						log.Printf("[%s] [Worker-%d] Successfully processed MessageID: %s", ts, workerID, event.MessageID)
						m.Ack(false)
					} else {
						log.Printf("[%s] [Worker-%d] MessageID: %s failed processing, routing to DLQ. Error: %v", ts, workerID, event.MessageID, err)
						m.Nack(false, false)
					}
				}(msg)
			}
		}
	}()

	return nil
}

func (c *RabbitMQConsumer) Close() error {
	if c.ch != nil {
		_ = c.ch.Close()
	}
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}
