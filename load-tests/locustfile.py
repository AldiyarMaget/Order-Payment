from locust import HttpUser, task, between
import random
import uuid

class MicroservicesUser(HttpUser):
    wait_time = between(0.1, 1.0) # High intensity, wait between 0.1s to 1s

    @task(3)
    def create_order(self):
        # Assuming order-service endpoint for creating an order
        order_payload = {
            "item_id": random.randint(1, 100),
            "quantity": random.randint(1, 5),
            "customer_id": f"cust_{random.randint(1, 1000)}"
        }
        # Assuming the order service runs on localhost:50052/api/v1/orders or similar. 
        # Using a generic endpoint /orders for demonstration.
        with self.client.post("http://localhost:8080/orders", json=order_payload, catch_response=True) as response:
            if response.status_code in [200, 201]:
                response.success()
            else:
                response.failure(f"Failed to create order: {response.status_code}")

    @task(2)
    def process_payment(self):
        # Assuming payment-service endpoint for processing a payment
        payment_payload = {
            "order_id": str(uuid.uuid4()),
            "amount": random.uniform(10.0, 500.0),
            "currency": "USD"
        }
        # Using a generic endpoint /payments for demonstration.
        with self.client.post("http://localhost:8081/payments", json=payment_payload, catch_response=True) as response:
            if response.status_code in [200, 201]:
                response.success()
            else:
                response.failure(f"Failed to process payment: {response.status_code}")

    @task(1)
    def get_order_status(self):
        # Assuming order-service endpoint for fetching order status
        order_id = str(uuid.uuid4())
        with self.client.get(f"http://localhost:8080/orders/{order_id}", catch_response=True) as response:
            if response.status_code in [200, 404]: # 404 is acceptable if order doesn't exist
                response.success()
            else:
                response.failure(f"Failed to fetch order status: {response.status_code}")
