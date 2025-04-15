package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"rtcs/internal/model"

	"github.com/redis/go-redis/v9"
)

type RedisCache struct {
	client *redis.Client
}

func NewRedisCache(client *redis.Client) *RedisCache {
	return &RedisCache{client: client}
}

func (c *RedisCache) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}

	return c.client.Set(ctx, key, data, expiration).Err()
}

func (c *RedisCache) Get(ctx context.Context, key string, dest interface{}) error {
	data, err := c.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil
		}
		return err
	}

	return json.Unmarshal(data, dest)
}

func (c *RedisCache) Delete(ctx context.Context, key string) error {
	return c.client.Del(ctx, key).Err()
}

func (c *RedisCache) Clear(ctx context.Context) error {
	return c.client.FlushAll(ctx).Err()
}

func (c *RedisCache) SetMessage(ctx context.Context, message *model.Message) error {
	data, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	key := fmt.Sprintf("message:%s", message.ID)
	return c.client.Set(ctx, key, data, 24*time.Hour).Err()
}

func (c *RedisCache) GetMessage(ctx context.Context, messageID string) (*model.Message, error) {
	key := fmt.Sprintf("message:%s", messageID)
	data, err := c.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get message from cache: %w", err)
	}

	var message model.Message
	if err := json.Unmarshal(data, &message); err != nil {
		return nil, fmt.Errorf("failed to unmarshal message: %w", err)
	}

	return &message, nil
}

func (c *RedisCache) DeleteMessage(ctx context.Context, messageID string) error {
	key := fmt.Sprintf("message:%s", messageID)
	return c.client.Del(ctx, key).Err()
}

func (c *RedisCache) SetChatMessages(ctx context.Context, chatID string, messages []*model.Message) error {
	data, err := json.Marshal(messages)
	if err != nil {
		return fmt.Errorf("failed to marshal messages: %w", err)
	}

	key := fmt.Sprintf("chat:%s:messages", chatID)
	return c.client.Set(ctx, key, data, 1*time.Hour).Err()
}

func (c *RedisCache) GetChatMessages(ctx context.Context, chatID string) ([]*model.Message, error) {
	key := fmt.Sprintf("chat:%s:messages", chatID)
	data, err := c.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get chat messages from cache: %w", err)
	}

	var messages []*model.Message
	if err := json.Unmarshal(data, &messages); err != nil {
		return nil, fmt.Errorf("failed to unmarshal messages: %w", err)
	}

	return messages, nil
}
