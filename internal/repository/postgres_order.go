package repository

import (
	"context"
	"database/sql"
	"errors"

	"order/internal/domain"
)

type PostgresOrder struct {
	db *sql.DB
}

func NewPostgresOrder(db *sql.DB) domain.OrderRepository {
	return &PostgresOrder{db: db}
}

func (r *PostgresOrder) Create(ctx context.Context, order *domain.Order) error {
	query := `INSERT INTO orders (id, customer_id, item_name, amount, status, created_at, idempotency_key) VALUES ($1, $2, $3, $4, $5, $6, $7)`
	_, err := r.db.ExecContext(ctx, query, order.ID, order.CustomerID, order.ItemName, order.Amount, order.Status, order.CreatedAt, order.IdempotencyKey)
	return err
}

func (r *PostgresOrder) GetByID(ctx context.Context, id string) (*domain.Order, error) {
	query := `SELECT id, customer_id, item_name, amount, status, created_at, idempotency_key FROM orders WHERE id = $1`
	var o domain.Order
	err := r.db.QueryRowContext(ctx, query, id).Scan(&o.ID, &o.CustomerID, &o.ItemName, &o.Amount, &o.Status, &o.CreatedAt, &o.IdempotencyKey)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("order not found")
		}
		return nil, err
	}
	return &o, nil
}

func (r *PostgresOrder) GetByIdempotencyKey(ctx context.Context, key string) (*domain.Order, error) {
	query := `SELECT id, customer_id, item_name, amount, status, created_at, idempotency_key FROM orders WHERE idempotency_key = $1`
	var o domain.Order
	err := r.db.QueryRowContext(ctx, query, key).Scan(&o.ID, &o.CustomerID, &o.ItemName, &o.Amount, &o.Status, &o.CreatedAt, &o.IdempotencyKey)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("order not found")
		}
		return nil, err
	}
	return &o, nil
}

func (r *PostgresOrder) UpdateStatus(ctx context.Context, id string, status domain.OrderStatus) error {
	query := `UPDATE orders SET status = $1 WHERE id = $2`
	_, err := r.db.ExecContext(ctx, query, status, id)
	return err
}

func (r *PostgresOrder) GetByAmountRange(ctx context.Context, minAmount, maxAmount int64) ([]domain.Order, error) {
	query := `SELECT id, customer_id, item_name, amount, status, created_at, idempotency_key FROM orders WHERE amount BETWEEN $1 AND $2`
	rows, err := r.db.QueryContext(ctx, query, minAmount, maxAmount)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	orders := []domain.Order{} // initialized as an empty slice, not nil
	for rows.Next() {
		var o domain.Order
		if err := rows.Scan(&o.ID, &o.CustomerID, &o.ItemName, &o.Amount, &o.Status, &o.CreatedAt, &o.IdempotencyKey); err != nil {
			return nil, err
		}
		orders = append(orders, o)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return orders, nil
}
