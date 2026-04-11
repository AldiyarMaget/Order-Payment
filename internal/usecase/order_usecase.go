package usecase

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"order/internal/domain"
)

var (
	ErrCannotCancelPaid   = errors.New("cannot cancel a paid order")
	ErrOrderNotFound      = errors.New("order not found")
	ErrInvalidState       = errors.New("invalid state transition")
	ErrPaymentUnavailable = errors.New("payment service unavailable")
)

type OrderUseCase interface {
	CreateOrder(ctx context.Context, customerID, itemName string, amount int64, idempotencyKey string) (*domain.Order, error)
	CancelOrder(ctx context.Context, id string) error
	GetOrder(ctx context.Context, id string) (*domain.Order, error)
	GetByAmountRange(ctx context.Context, min, max int64) ([]domain.Order, error)
}

type orderUseCase struct {
	repo          domain.OrderRepository
	paymentClient domain.PaymentClient
}

func NewOrderUseCase(repo domain.OrderRepository, paymentClient domain.PaymentClient) OrderUseCase {
	return &orderUseCase{
		repo:          repo,
		paymentClient: paymentClient,
	}
}

func (u *orderUseCase) CreateOrder(ctx context.Context, customerID, itemName string, amount int64, idempotencyKey string) (*domain.Order, error) {
	existingOrder, err := u.repo.GetByIdempotencyKey(ctx, idempotencyKey)
	if err == nil && existingOrder != nil {
		return existingOrder, nil
	}

	order := &domain.Order{
		ID:             uuid.New().String(),
		CustomerID:     customerID,
		ItemName:       itemName,
		Amount:         amount,
		Status:         domain.StatusPending,
		CreatedAt:      time.Now(),
		IdempotencyKey: idempotencyKey,
	}

	if err := u.repo.Create(ctx, order); err != nil {
		return nil, err
	}

	paymentStatus, err := u.paymentClient.AuthorizePayment(ctx, order.ID, order.Amount)

	if err != nil {
		_ = u.repo.UpdateStatus(ctx, order.ID, domain.StatusFailed)
		return nil, ErrPaymentUnavailable
	}

	newStatus := domain.StatusPaid
	if paymentStatus == "Declined" {
		newStatus = domain.StatusFailed
	} else if paymentStatus == "Authorized" {
		newStatus = domain.StatusPaid
	}

	if err := u.repo.UpdateStatus(ctx, order.ID, newStatus); err != nil {
	}
	order.Status = newStatus

	return order, nil
}

func (u *orderUseCase) CancelOrder(ctx context.Context, id string) error {
	order, err := u.repo.GetByID(ctx, id)
	if err != nil {
		return ErrOrderNotFound
	}

	if order.Status == domain.StatusPaid {
		return ErrCannotCancelPaid
	}

	if order.Status != domain.StatusPending {
		return ErrInvalidState
	}

	return u.repo.UpdateStatus(ctx, id, domain.StatusCancelled)
}

func (u *orderUseCase) GetOrder(ctx context.Context, id string) (*domain.Order, error) {
	order, err := u.repo.GetByID(ctx, id)
	if err != nil {
		return nil, ErrOrderNotFound
	}
	return order, nil
}

func (u *orderUseCase) GetByAmountRange(ctx context.Context, min, max int64) ([]domain.Order, error) {
	return u.repo.GetByAmountRange(ctx, min, max)
}
