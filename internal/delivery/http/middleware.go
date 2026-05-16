package http

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// RateLimitMiddleware implements a Fixed Window rate limiting algorithm using Redis.
func RateLimitMiddleware(client *redis.Client, limit int64, window time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Use IP address as the identifier for rate limiting
		ip := c.ClientIP()
		if ip == "" {
			ip = "unknown"
		}

		key := fmt.Sprintf("rate_limit:%s", ip)

		// Increment the counter for this IP
		count, err := client.Incr(c.Request.Context(), key).Result()
		if err != nil {
			log.Printf("[RateLimiter] Error incrementing key %s: %v. Allowing request to pass.", key, err)
			c.Next()
			return
		}

		// If this is the first request in the window, set the expiration
		if count == 1 {
			client.Expire(c.Request.Context(), key, window)
		}

		// Check if the limit has been exceeded
		if count > limit {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "too many requests, please try again later",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
