package main

import (
	"database/sql"
	"log"
	"net"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"google.golang.org/grpc"
	"net/http"

	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"

	grpc_delivery "order/payment/internal/delivery/grpc"
	delivery "order/payment/internal/delivery/http"
	"order/payment/internal/repository"
	"order/payment/internal/usecase"

	contract "github.com/AldiyarMaget/aitu-go-sdk/contract"
	"github.com/prometheus/client_golang/prometheus/promhttp"
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
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))
	delivery.NewPaymentHandler(r, uc)

	go func() {
		http.Handle("/metrics", promhttp.Handler())
		log.Println("Starting metrics server on :8080")
		if err := http.ListenAndServe(":8080", nil); err != nil {
			log.Fatalf("Metrics server failed: %v", err)
		}
	}()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}

	grpcPort := os.Getenv("PAYMENT_GRPC_PORT")
	if grpcPort == "" {
		grpcPort = ":50051"
	}

	go func() {
		lis, err := net.Listen("tcp", grpcPort)
		if err != nil {
			log.Fatalf("failed to listen on gRPC port: %v", err)
		}

		grpcServer := grpc.NewServer(
			grpc.ChainUnaryInterceptor(
				grpc_delivery.LoggingInterceptor,
				grpc_prometheus.UnaryServerInterceptor,
			),
		)
		grpcHandler := grpc_delivery.NewPaymentHandler(uc)
		contract.RegisterPaymentServiceServer(grpcServer, grpcHandler)
		grpc_prometheus.Register(grpcServer)

		log.Printf("Payment gRPC service starting on port %s", grpcPort)
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("failed to run payment gRPC service: %v", err)
		}
	}()

	log.Printf("Payment HTTP service starting on port %s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("failed to run payment HTTP service: %v", err)
	}
}
