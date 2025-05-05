package limiter

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

type ConcurrencyMiddleware struct {
	redisClient   *redis.Client
	maxConcurrent int
	expire        time.Duration
}

func NewConcurrencyMiddleware(rdb *redis.Client, maxConcurrent int, expire time.Duration) gin.HandlerFunc {
	cm := &ConcurrencyMiddleware{
		redisClient:   rdb,
		maxConcurrent: maxConcurrent,
		expire:        expire,
	}
	return cm.handle
}

func (c *ConcurrencyMiddleware) handle(ctx *gin.Context) {
	userID := ctx.ClientIP()

	allowed, key, err := c.increment(userID)

	if err != nil {
		ctx.Next()
		return
	}

	if !allowed {
		ctx.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "too many concurrent requests"})
		return
	}

	defer func() {
		if err := c.decrement(userID, key); err != nil {
		}
	}()

	ctx.Next()
}

func (c *ConcurrencyMiddleware) increment(userID string) (bool, string, error) {
	rdb := c.redisClient
	ctx := context.Background()

	key := fmt.Sprintf("concurrency:%s", userID)

	current, err := rdb.Get(ctx, key).Int()
	if err != nil && err != redis.Nil {
		return false, key, err
	}

	if current >= c.maxConcurrent {
		return false, key, nil
	}

	pipe := rdb.Pipeline()
	pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, c.expire)
	_, err = pipe.Exec(ctx)

	return err == nil, key, err
}

func (c *ConcurrencyMiddleware) decrement(userID string, key string) error {
	ctx := context.Background()
	return c.redisClient.Decr(ctx, key).Err()
}
