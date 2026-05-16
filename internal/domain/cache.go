package domain

import (
	"context"
	"time"
)

type OrderCache interface {
	Get(ctx context.Context, id string) (*Order, error)
	Set(ctx context.Context, order *Order, ttl time.Duration) error
	Delete(ctx context.Context, id string) error
}
