package grpc_delivery

import (
	"context"
	"log"
	"time"

	"google.golang.org/grpc"
)

// LoggingInterceptor logs the method name and execution time for each gRPC request.
func LoggingInterceptor(
	ctx context.Context,
	req interface{},
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (interface{}, error) {
	start := time.Now()

	log.Printf("gRPC request started: %s", info.FullMethod)

	resp, err := handler(ctx, req)

	duration := time.Since(start)

	if err != nil {
		log.Printf("gRPC request failed: %s | duration: %v | error: %v", info.FullMethod, duration, err)
	} else {
		log.Printf("gRPC request completed: %s | duration: %v", info.FullMethod, duration)
	}

	return resp, err
}
