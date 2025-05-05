package limiter

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

type RateLimitMiddleware struct {
	redisClient *redis.Client
	algorithm   string
	limit       int
	window      time.Duration
}

func NewRateLimitMiddleware(rdb *redis.Client, algo string, limit int, windowStr string) (gin.HandlerFunc, error) {
	dur, err := time.ParseDuration(windowStr)
	if err != nil {
		return nil, fmt.Errorf("invalid window duration: %w", err)
	}

	mw := &RateLimitMiddleware{
		redisClient: rdb,
		algorithm:   strings.ToLower(algo),
		limit:       limit,
		window:      dur,
	}

	return mw.handle, nil
}

func (r *RateLimitMiddleware) handle(c *gin.Context) {
	userID := c.ClientIP()
	key := fmt.Sprintf("ratelimit:%s", userID)

	ctx := context.Background()
	pipe := r.redisClient.Pipeline()

	// Clean up old window
	pipe.ZRemRangeByScore(ctx, key, "-inf", fmt.Sprintf("%d", time.Now().Add(-r.window).UnixNano()))

	// Add current request
	pipe.ZAdd(ctx, key, redis.Z{Score: float64(time.Now().UnixNano()), Member: time.Now().UnixNano()})

	// Get count in current window
	pipe.ZCard(ctx, key)

	// Set expiry
	pipe.Expire(ctx, key, r.window)

	results, err := pipe.Exec(ctx)
	if err != nil {
		c.Next()
		return
	}

	count := results[2].(*redis.IntCmd).Val()
	if count > int64(r.limit) {
		c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "rate limit exceeded"})
		return
	}

	c.Next()
}
