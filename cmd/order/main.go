package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"

	delivHTTP "order/internal/delivery/http"
	"order/internal/infrastructure"
	"order/internal/repository"
	"order/internal/usecase"
)

func runMigrations(db *sql.DB, filepath string) error {
	content, err := os.ReadFile(filepath)
	if err != nil {
		return err
	}
	_, err = db.Exec(string(content))
	return err
}

func main() {
	if err := godotenv.Overload(".env"); err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:postgres@localhost:5432/order_db?sslmode=disable"
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("database ping failed: %v", err)
	}

	log.Println("Running database migrations...")
	if err := runMigrations(db, "migrations/000001_create_orders_table.up.sql"); err != nil {
		log.Fatalf("failed to run migrations: %v", err)
	}
	log.Println("Database migrations applied.")

	paymentURL := os.Getenv("PAYMENT_SERVICE_URL")
	if paymentURL == "" {
		paymentURL = "http://localhost:8081"
	}

	orderRepo := repository.NewPostgresOrder(db)
	
	httpClient := &http.Client{
		Timeout: 2 * time.Second,
	}
	paymentClient := infrastructure.NewPaymentClient(httpClient, paymentURL)
	orderUseCase := usecase.NewOrderUseCase(orderRepo, paymentClient)

	router := gin.Default()
	delivHTTP.NewOrderHandler(router, orderUseCase)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Order Service listening on :%s", port)
	if err := http.ListenAndServe(":"+port, router); err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}
