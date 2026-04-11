CREATE TABLE IF NOT EXISTS payments (
    id UUID PRIMARY KEY,
    order_id VARCHAR(255) NOT NULL,
    transaction_id VARCHAR(255) DEFAULT '',
    amount BIGINT NOT NULL,
    status VARCHAR(50) NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_payments_order_id ON payments(order_id);
