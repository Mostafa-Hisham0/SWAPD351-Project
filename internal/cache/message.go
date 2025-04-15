package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"rtcs/internal/model"
	"time"

	"github.com/redis/go-redis/v9"
)

type MessageCache struct {
	client *redis.Client
}

func NewMessageCache(client *redis.Client) *MessageCache {
	return &MessageCache{
		client: client,
	}
}

func (c *MessageCache) SetMessage(ctx context.Context, message *model.Message) error {
	key := fmt.Sprintf("message:%s", message.ID)
	data, err := json.Marshal(message)
	if err != nil {
		return err
	}
	return c.client.Set(ctx, key, data, 24*time.Hour).Err()
}

func (c *MessageCache) GetMessage(ctx context.Context, messageID string) (*model.Message, error) {
	key := fmt.Sprintf("message:%s", messageID)
	data, err := c.client.Get(ctx, key).Bytes()
	if err != nil {
		return nil, err
	}
	var message model.Message
	if err := json.Unmarshal(data, &message); err != nil {
		return nil, err
	}
	return &message, nil
}

func (c *MessageCache) DeleteMessage(ctx context.Context, messageID string) error {
	key := fmt.Sprintf("message:%s", messageID)
	return c.client.Del(ctx, key).Err()
}

func (c *MessageCache) SetChatMessages(ctx context.Context, chatID string, messages []*model.Message) error {
	key := fmt.Sprintf("chat:%s:messages", chatID)
	data, err := json.Marshal(messages)
	if err != nil {
		return err
	}
	return c.client.Set(ctx, key, data, 1*time.Hour).Err()
}

func (c *MessageCache) GetChatMessages(ctx context.Context, chatID string) ([]*model.Message, error) {
	key := fmt.Sprintf("chat:%s:messages", chatID)
	data, err := c.client.Get(ctx, key).Bytes()
	if err != nil {
		return nil, err
	}
	var messages []*model.Message
	if err := json.Unmarshal(data, &messages); err != nil {
		return nil, err
	}
	return messages, nil
}
