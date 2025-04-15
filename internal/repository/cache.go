package repository

import (
	"context"
	"encoding/json"
	"time"

	"rtcs/internal/model"

	"github.com/redis/go-redis/v9"
)

// Cache defines the interface for caching operations
type Cache interface {
	GetChatHistory(ctx context.Context, chatID string) ([]model.Message, error)
	SetChatHistory(ctx context.Context, chatID string, messages []model.Message) error
	DeleteChatHistory(ctx context.Context, chatID string) error
}

// RedisCache implements the Cache interface using Redis
type RedisCache struct {
	client *redis.Client
	ttl    time.Duration
}

// NewRedisCache creates a new Redis cache
func NewRedisCache(client *redis.Client, ttl time.Duration) Cache {
	return &RedisCache{
		client: client,
		ttl:    ttl,
	}
}

// GetChatHistory retrieves chat history from Redis
func (c *RedisCache) GetChatHistory(ctx context.Context, chatID string) ([]model.Message, error) {
	key := "chat:" + chatID + ":messages"
	data, err := c.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var messages []model.Message
	err = json.Unmarshal(data, &messages)
	if err != nil {
		return nil, err
	}

	return messages, nil
}

// SetChatHistory stores chat history in Redis
func (c *RedisCache) SetChatHistory(ctx context.Context, chatID string, messages []model.Message) error {
	key := "chat:" + chatID + ":messages"
	data, err := json.Marshal(messages)
	if err != nil {
		return err
	}

	return c.client.Set(ctx, key, data, c.ttl).Err()
}

// DeleteChatHistory removes chat history from Redis
func (c *RedisCache) DeleteChatHistory(ctx context.Context, chatID string) error {
	key := "chat:" + chatID + ":messages"
	return c.client.Del(ctx, key).Err()
}
