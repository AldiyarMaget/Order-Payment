package usecase

import (
	"context"
	"errors"
	"fmt"
	"sync"
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
	SubscribeToOrder(ctx context.Context, orderID string) (<-chan *domain.Order, func())
}

type orderUseCase struct {
	repo          domain.OrderRepository
	paymentClient domain.PaymentClient

	subscribers map[string][]chan *domain.Order
	mu          sync.RWMutex
}

func NewOrderUseCase(repo domain.OrderRepository, paymentClient domain.PaymentClient) OrderUseCase {
	return &orderUseCase{
		repo:          repo,
		paymentClient: paymentClient,
		subscribers:   make(map[string][]chan *domain.Order),
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

	u.notify(order) // Initial notification

	paymentStatus, err := u.paymentClient.AuthorizePayment(ctx, order.ID, order.Amount)

	if err != nil {
		_ = u.repo.UpdateStatus(ctx, order.ID, domain.StatusFailed)
		return nil, fmt.Errorf("%w: %w", ErrPaymentUnavailable, err)
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

	u.notify(order) // Notification after payment result

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

	if err := u.repo.UpdateStatus(ctx, id, domain.StatusCancelled); err != nil {
		return err
	}
	order.Status = domain.StatusCancelled
	u.notify(order)

	return nil
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

func (u *orderUseCase) SubscribeToOrder(ctx context.Context, orderID string) (<-chan *domain.Order, func()) {
	ch := make(chan *domain.Order, 1)

	u.mu.Lock()
	u.subscribers[orderID] = append(u.subscribers[orderID], ch)
	u.mu.Unlock()

	// Immediately push the current state to the client
	order, err := u.GetOrder(ctx, orderID)
	if err == nil && order != nil {
		ch <- order
	}

	cleanup := func() {
		u.mu.Lock()
		defer u.mu.Unlock()
		subs := u.subscribers[orderID]
		for i, subCh := range subs {
			if subCh == ch {
				u.subscribers[orderID] = append(subs[:i], subs[i+1:]...)
				break
			}
		}
		close(ch)
	}

	return ch, cleanup
}

func (u *orderUseCase) notify(order *domain.Order) {
	u.mu.RLock()
	defer u.mu.RUnlock()

	subs, exists := u.subscribers[order.ID]
	if !exists {
		return
	}

	for _, ch := range subs {
		select {
		case ch <- order:
		default: // Avoid blocking if a client channel is full
		}
	}
}
