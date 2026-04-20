package main

import (
	"database/sql"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"google.golang.org/grpc"

	grpc_delivery "order/internal/delivery/grpc"
	delivHTTP "order/internal/delivery/http"
	"order/internal/infrastructure"
	"order/internal/repository"
	"order/internal/usecase"

	contract "github.com/AldiyarMaget/aitu-go-sdk/contract"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	httpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests.",
		},
		[]string{"method", "endpoint", "code"},
	)
	httpRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Duration of HTTP requests in seconds.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "endpoint", "code"},
	)
)

func init() {
	prometheus.MustRegister(httpRequestsTotal)
	prometheus.MustRegister(httpRequestDuration)
}

func prometheusMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		duration := time.Since(start).Seconds()

		method := c.Request.Method
		endpoint := c.FullPath()
		if endpoint == "" {
			endpoint = "unknown"
		}
		status := strconv.Itoa(c.Writer.Status())

		httpRequestsTotal.WithLabelValues(method, endpoint, status).Inc()
		httpRequestDuration.WithLabelValues(method, endpoint, status).Observe(duration)
	}
}

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

	paymentAddr := os.Getenv("PAYMENT_SERVICE_ADDR")
	if paymentAddr == "" {
		paymentAddr = "localhost:50051"
	}

	orderRepo := repository.NewPostgresOrder(db)

	paymentClient := infrastructure.NewPaymentClient(paymentAddr)
	orderUseCase := usecase.NewOrderUseCase(orderRepo, paymentClient)

	router := gin.Default()
	router.Use(prometheusMiddleware())
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))
	delivHTTP.NewOrderHandler(router, orderUseCase)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	grpcPort := os.Getenv("ORDER_GRPC_PORT")
	if grpcPort == "" {
		grpcPort = ":50052"
	}

	go func() {
		lis, err := net.Listen("tcp", grpcPort)
		if err != nil {
			log.Fatalf("failed to listen on order gRPC port: %v", err)
		}

		grpcServer := grpc.NewServer()
		grpcHandler := grpc_delivery.NewOrderHandler(orderUseCase)
		contract.RegisterOrderTrackingServiceServer(grpcServer, grpcHandler)

		log.Printf("Order gRPC Service listening on %s", grpcPort)
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("failed to serve order gRPC: %v", err)
		}
	}()

	log.Printf("Order HTTP Service listening on :%s", port)
	if err := http.ListenAndServe(":"+port, router); err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}
