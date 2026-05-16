package repository

import (
	"context"
	"database/sql"
	"errors"

	"order/payment/internal/domain"
)

type postgresPaymentRepository struct {
	db *sql.DB
}

func NewPostgresPaymentRepository(db *sql.DB) domain.PaymentRepository {
	return &postgresPaymentRepository{db: db}
}

func (r *postgresPaymentRepository) ListByStatus(ctx context.Context, status string) ([]*domain.Payment, error) {
	query := "SELECT id, order_id, transaction_id, amount, status FROM payments WHERE status = $1"
	rows, err := r.db.QueryContext(ctx, query, status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var payments []*domain.Payment
	for rows.Next() {
		p := &domain.Payment{}
		var txID sql.NullString
		if err := rows.Scan(&p.ID, &p.OrderID, &txID, &p.Amount, &p.Status); err != nil {
			return nil, err
		}
		if txID.Valid {
			p.TransactionID = txID.String
		}
		payments = append(payments, p)
	}
	return payments, nil
}

func (r *postgresPaymentRepository) Create(ctx context.Context, payment *domain.Payment) error {
	query := `
		INSERT INTO payments (id, order_id, transaction_id, amount, status)
		VALUES ($1, $2, $3, $4, $5)
	`
	_, err := r.db.ExecContext(ctx, query, payment.ID, payment.OrderID, payment.TransactionID, payment.Amount, payment.Status)
	return err
}

func (r *postgresPaymentRepository) GetByOrderID(ctx context.Context, orderID string) (*domain.Payment, error) {
	query := `
		SELECT id, order_id, transaction_id, amount, status
		FROM payments
		WHERE order_id = $1
	`
	row := r.db.QueryRowContext(ctx, query, orderID)

	var p domain.Payment
	var txID sql.NullString
	err := row.Scan(&p.ID, &p.OrderID, &txID, &p.Amount, &p.Status)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("payment not found")
		}
		return nil, err
	}
	if txID.Valid {
		p.TransactionID = txID.String
	}
	return &p, nil
}
