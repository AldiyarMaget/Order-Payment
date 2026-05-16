package usecase

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"order/payment/internal/domain"
	"order/payment/internal/infrastructure"
)

var ErrPaymentExists = errors.New("payment already processed for this order")

type PaymentUseCase interface {
	ProcessPayment(ctx context.Context, orderID string, amount int64) (*domain.Payment, error)
	GetPaymentByOrderID(ctx context.Context, orderID string) (*domain.Payment, error)
	ListPayments(ctx context.Context, status string) ([]*domain.Payment, error)
}

type paymentUseCase struct {
	repo   domain.PaymentRepository
	broker infrastructure.MessageBroker
}

func NewPaymentUseCase(repo domain.PaymentRepository, broker infrastructure.MessageBroker) PaymentUseCase {
	return &paymentUseCase{repo: repo, broker: broker}
}

func (u *paymentUseCase) ProcessPayment(ctx context.Context, orderID string, amount int64) (*domain.Payment, error) {
	existing, err := u.repo.GetByOrderID(ctx, orderID)
	if err == nil && existing != nil {
		return existing, ErrPaymentExists
	}

	status := domain.PaymentStatusAuthorized
	var txID string
	if amount > 100000 {
		status = domain.PaymentStatusDeclined
	} else {
		txID = uuid.New().String()
	}

	payment := &domain.Payment{
		ID:            uuid.New().String(),
		OrderID:       orderID,
		TransactionID: txID,
		Amount:        amount,
		Status:        status,
	}

	if err := u.repo.Create(ctx, payment); err != nil {
		return nil, err
	}

	// Publish event strictly after the database transaction is successfully committed
	// We'll simulate customer_email by generating one from orderID for now since the contract doesn't have it
	event := infrastructure.PaymentEvent{
		OrderID:       payment.OrderID,
		Amount:        payment.Amount,
		CustomerEmail: "user" + payment.OrderID + "@example.com",
		Status:        string(payment.Status),
		MessageID:     uuid.New().String(),
	}

	_ = u.broker.PublishPaymentCompleted(ctx, event)

	return payment, nil
}

func (u *paymentUseCase) GetPaymentByOrderID(ctx context.Context, orderID string) (*domain.Payment, error) {
	return u.repo.GetByOrderID(ctx, orderID)
}

func (u *paymentUseCase) ListPayments(ctx context.Context, status string) ([]*domain.Payment, error) {
	return u.repo.ListByStatus(ctx, status)
}
