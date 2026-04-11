package main

import (
	"database/sql"
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"

	delivery "order/payment/internal/delivery/http"
	"order/payment/internal/repository"
	"order/payment/internal/usecase"
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
	if err := godotenv.Overload("payment/.env"); err != nil {
		godotenv.Overload(".env")
	}

	dbConnStr := os.Getenv("PAYMENT_DB_URL")
	if dbConnStr == "" {
		dbConnStr = "postgres://postgres:postgres@localhost:5432/payment_db?sslmode=disable"
	}

	db, err := sql.Open("postgres", dbConnStr)
	if err != nil {
		log.Fatalf("failed to connect to payment database: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("failed to ping payment database: %v", err)
	}

	log.Println("Running payment database migrations...")
	if err := runMigrations(db, "payment/migrations/000001_create_payments_table.up.sql"); err != nil {
		log.Fatalf("failed to run migrations: %v", err)
	}
	log.Println("Payment database migrations applied.")

	repo := repository.NewPostgresPaymentRepository(db)
	uc := usecase.NewPaymentUseCase(repo)

	r := gin.Default()
	delivery.NewPaymentHandler(r, uc)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}

	log.Printf("Payment service starting on port %s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("failed to run payment service: %v", err)
	}
}
