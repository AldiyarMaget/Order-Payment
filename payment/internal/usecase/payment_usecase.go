package usecase

import (
	"context"
	"errors"

	"order/payment/internal/domain"
	"github.com/google/uuid"
)

var ErrPaymentExists = errors.New("payment already processed for this order")

type PaymentUseCase interface {
	ProcessPayment(ctx context.Context, orderID string, amount int64) (*domain.Payment, error)
	GetPaymentByOrderID(ctx context.Context, orderID string) (*domain.Payment, error)
}

type paymentUseCase struct {
	repo domain.PaymentRepository
}

func NewPaymentUseCase(repo domain.PaymentRepository) PaymentUseCase {
	return &paymentUseCase{repo: repo}
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

	return payment, nil
}

func (u *paymentUseCase) GetPaymentByOrderID(ctx context.Context, orderID string) (*domain.Payment, error) {
	return u.repo.GetByOrderID(ctx, orderID)
}
