# Order and Payment Infrastructure

This repository contains the `Order App` and `Payment App` designed according to Clean Architecture and Domain-Driven Design principles.
They communicate synchronously via REST with specific failure cases gracefully handled.

## System Architecture

**Tech Stack:** Go 1.2x, Gin Gonic, PostgreSQL, Docker Compose  
**Ports:**
- **Order Service:** `8080`
- **Payment Service:** `8081`

```mermaid
flowchart TD
    Client[Client POST /orders] -->|HTTP| OrderDelivery[Order Delivery: Gin Router]
    
    subgraph "Order Service"
        OrderDelivery --> OrderUseCase[Order UseCase]
        OrderUseCase --> OrderRepository[Order Repository]
        OrderUseCase --> PaymentClient[Payment Client: REST]
    end

    subgraph "Payment Service"
        PaymentDelivery[Payment Delivery: Gin Router] --> PaymentUseCase[Payment UseCase]
        PaymentUseCase --> PaymentRepository[Payment Repository]
    end

    OrderRepository -->|Read/Write| OrderDB[(PostgreSQL\norders-db :5431)]
    PaymentRepository -->|Read/Write| PaymentDB[(PostgreSQL\npayments-db :5432)]

    PaymentClient -->|HTTP 2s timeout| PaymentDelivery
```

## Setup Instructions

### 1. Start the Databases
The services rely on two independent PostgreSQL databases. To spin them up, use Docker Compose:

```bash
docker-compose up -d
```

This starts:
- `orders-db` accessible externally on `localhost:5431`.
- `payments-db` accessible externally on `localhost:5432`.

### 2. Configure Environment
Copy the example environment variables file and configure it (optional for defaults):
```bash
cp .env.example .env
```
Ensure you provide these environment variables when running if you deviate from the `.env.example` file. 

### 3. Run the Services
Note: The databases apply necessary schema migrations automatically when the services start.

**Terminal 1 (Run Payment Service):**
```bash
cd payment
PORT=8081 PAYMENT_DB_URL=postgres://postgres:postgres@localhost:5432/payment_db?sslmode=disable go run cmd/payment/main.go
```

**Terminal 2 (Run Order Service):**
```bash
PORT=8080 DATABASE_URL=postgres://postgres:postgres@localhost:5431/order_db?sslmode=disable PAYMENT_SERVICE_URL=http://localhost:8081 go run cmd/order/main.go
```

## API Testing & CURL Examples

### 1. Post a Valid Order
This will successfully create an order and call the Payment service to authorize.

```
curl -X POST http://localhost:8080/orders \
-H "Content-Type: application/json" \
-H "Idempotency-Key: unique-key-1" \
-d '{
    "customer_id": "cust-123",
    "item_name": "Premium Widget",
    "amount": 50000
}'
```

### 2. Verify Idempotency 
Run the exact same command with the `-H "Idempotency-Key: unique-key-1"` header. It should return `200 OK` safely returning the existing order ID from the first request rather than creating a new one.

### 3. Decline High Values
The Payment service will decline payments greater than 1,000.00 (100,000 cents).

```bash
curl -X POST http://localhost:8080/orders \
-H "Content-Type: application/json" \
-H "Idempotency-Key: unique-key-2" \
-d '{
    "customer_id": "cust-124",
    "item_name": "Luxury Yacht",
    "amount": 100001
}'
```
This produces an authorization failure from the Payment app and the Order status falls back to `Failed`.

### 4. 2-Second Timeout Scenario (Simulating Outage)
The `Order Service` has a strict `.Timeout = 2 * time.Second` when calling the Payment service. If the Payment Service is offline, the Order Service returns an `HTTP 503 Service Unavailable` status rather than returning a 201 Created.

**How to trigger:**
1. **Stop** the Payment Service terminal process (Ctrl+C).
2. Execute the curl below:

```bash
curl -X POST http://localhost:8080/orders \
-H "Content-Type: application/json" \
-H "Idempotency-Key: unique-key-3" \
-d '{
    "customer_id": "cust-125",
    "item_name": "Timeout Widget",
    "amount": 200
}'
```

Because the Payment Service is offline, the Order service will fail immediately (Connection Refused or Timeout), strictly enforcing the 2-second timeout envelope, and explicitly respond to the client with an `HTTP 503 Service Unavailable` status.
