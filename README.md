# Assignment 2: gRPC Microservices Architecture

## Project Overview
This project consists of two core microservices: **Order Service** and **Payment Service**, communicating efficiently via gRPC. The system demonstrates a modern microservices architecture by handling standard REST API requests, securely executing internal unary gRPC calls between services, and providing real-time updates to clients via server-side streaming.

## Architecture Links
The project utilizes a Contract-First approach, keeping protocol definitions and generated SDKs in dedicated, decoupled repositories:
* **Contract Repository (Protos):** [aitu-protos](https://github.com/AldiyarMaget/aitu-protos)
* **Generated SDK Repository:** [aitu-go-sdk](https://github.com/AldiyarMaget/aitu-go-sdk)

## Tech Stack
* **Language:** Go 1.22
* **Communication:** 
  * gRPC (Unary & Server-Side Streaming)
  * REST API (Gin Framework)
* **Database:** PostgreSQL
* **Infrastructure:** Docker, Docker Compose

## Key Features
* **gRPC Interceptors:** Implemented server-side interceptors for centralized logging of incoming gRPC requests.
* **Observer Pattern:** Implemented in the Order Service for real-time state notification broadcasting to connected gRPC streaming clients.
* **Resilient Error Propagating:** Utilizes `google.golang.org/grpc/status` to accurately relay failure scenarios across service boundaries.

## Architecture Diagram
```mermaid 
graph TD
    Client1[External Client] -->|"REST Request (POST /orders)"| OrderREST["Order Service (REST)<br/>Port: 8080"]
    Client2[External Client] -->|"gRPC Stream (SubscribeReq)"| OrderGRPC["Order Service (gRPC)<br/>Port: 50052"]

    OrderREST -->|"gRPC Unary (ProcessPayment)"| PaymentService["Payment Service<br/>Port: 50051"]

    subgraph Docker Infrastructure
        OrderDB[("Order Database<br/>(PostgreSQL)")]
        PaymentDB[("Payment Database<br/>(PostgreSQL)")]
    end

    OrderREST -.->|TCP| OrderDB
    OrderGRPC -.->|TCP| OrderDB
    PaymentService -.->|TCP| PaymentDB
```

## Execution Instructions

### 1. Environment Configuration (`.env`)
Ensure an `.env` file exists in the root of the project. Below is an example configuration containing necessary variable addresses:
```env
# External REST API Configuration
PORT=8080

# Order Service Dependencies
DATABASE_URL=postgres://postgres:postgres@localhost:5431/order_db?sslmode=disable
PAYMENT_SERVICE_ADDR=localhost:50051
ORDER_GRPC_PORT=:50052

# Payment Service Dependencies
PAYMENT_DB_URL=postgres://postgres:postgres@localhost:5432/payment_db?sslmode=disable
```

### 2. Launch Databases in Docker
Boot up the PostgreSQL instances defined for both services using Docker Compose:
```bash
docker-compose up -d
```

### 3. Start Microservices
Run the microservices from the project root using `go run` or by compiling the binaries.

**Launch the Payment Service:**
This command will start the gRPC server listening securely on port `50051`.
```bash
go run ./payment/cmd/main.go
```

**Launch the Order Service:**
In a separate terminal instance, launch the Order Service. It sequentially launches both the REST interface (Port `8080`) and the gRPC Stream server (Port `50052`).
```bash
go run ./cmd/order/main.go
```

## Assignment 3: Event-Driven Architecture

This system has been upgraded to an Event-Driven Architecture using RabbitMQ to decouple the Payment and Notification services.

### Idempotency Strategy
The `Notification Service` implements an idempotent consumer. Upon receiving an event, it checks an in-memory `sync.Map` store (simulating an idempotent cache/DB) using the event's unique `message_id`. If the message ID exists, the event is considered a duplicate and business logic processing is skipped. Regardless of whether it was skipped or newly processed, a successful manual ACK is always sent to ensure the broker removes the message.

### ACK Logic & Reliability
* **Manual Acknowledgements:** `auto-ack` is disabled in the Consumer. `msg.Ack(false)` is sent strictly *after* the business side effect (logging the email) has completed successfully.
* **Dead Letter Queue (DLQ):** The Notification Service is configured with a retry mechanism. If processing fails 3 times, the message is `NACK`ed with `requeue=false`, routing it to a configured Dead Letter Exchange/Queue for inspection and manual intervention.
* **Transactional Publishing:** The Payment Service (Producer) publishes messages to RabbitMQ *only after* the PostgreSQL transaction (recording the payment) has successfully committed. This eliminates "phantom events" ensuring events perfectly reflect the system of record.
