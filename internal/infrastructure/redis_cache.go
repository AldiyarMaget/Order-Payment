package infrastructure

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"order/internal/domain"

	"github.com/redis/go-redis/v9"
)

type RedisOrderCache struct {
	client *redis.Client
}

func NewRedisOrderCache(client *redis.Client) domain.OrderCache {
	return &RedisOrderCache{
		client: client,
	}
}

func (c *RedisOrderCache) Get(ctx context.Context, id string) (*domain.Order, error) {
	key := fmt.Sprintf("order:%s", id)
	val, err := c.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil, nil // Cache miss
	} else if err != nil {
		log.Printf("[Redis] Failed to GET order %s: %v. Falling back to DB.", id, err)
		return nil, nil // Fail Open
	}

	var order domain.Order
	if err := json.Unmarshal([]byte(val), &order); err != nil {
		log.Printf("[Redis] Failed to unmarshal order %s: %v", id, err)
		return nil, nil
	}

	return &order, nil
}

func (c *RedisOrderCache) Set(ctx context.Context, order *domain.Order, ttl time.Duration) error {
	key := fmt.Sprintf("order:%s", order.ID)
	data, err := json.Marshal(order)
	if err != nil {
		return err
	}

	if err := c.client.Set(ctx, key, data, ttl).Err(); err != nil {
		log.Printf("[Redis] Failed to SET order %s: %v", order.ID, err)
		// Fail Open: we don't return an error so the main flow isn't interrupted
	}

	return nil
}

func (c *RedisOrderCache) Delete(ctx context.Context, id string) error {
	key := fmt.Sprintf("order:%s", id)
	if err := c.client.Del(ctx, key).Err(); err != nil {
		log.Printf("[Redis] Failed to DELETE order %s: %v", id, err)
	}
	return nil
}
