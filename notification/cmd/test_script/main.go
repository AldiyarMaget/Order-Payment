package main

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"

	amqp "github.com/rabbitmq/amqp091-go"
)

type NotificationEvent struct {
	OrderID       string `json:"order_id"`
	Amount        int64  `json:"amount"`
	CustomerEmail string `json:"customer_email"`
	Status        string `json:"status"`
	MessageID     string `json:"message_id"`
}

func main() {
	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("Failed to open a channel: %v", err)
	}
	defer ch.Close()

	var wg sync.WaitGroup

	fmt.Println("Sending 10 test messages concurrently...")

	for i := 1; i <= 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			
			event := NotificationEvent{
				OrderID:       fmt.Sprintf("ORD-%04d", id),
				Amount:        1000,
				CustomerEmail: fmt.Sprintf("user%d@example.com", id),
				Status:        "COMPLETED",
				MessageID:     fmt.Sprintf("MSG-TEST-%d", id),
			}

			body, _ := json.Marshal(event)

			err = ch.Publish(
				"",                  // exchange
				"payment.completed", // routing key
				false,               // mandatory
				false,               // immediate
				amqp.Publishing{
					ContentType: "application/json",
					Body:        body,
				})
			if err != nil {
				log.Printf("Failed to publish message %d: %v", id, err)
			} else {
				log.Printf("Published message %d", id)
			}
		}(i)
	}

	wg.Wait()
	fmt.Println("Done sending messages.")
}
