package service

import (
	"context"
	"rtcs/internal/model"
)

// Repository defines the interface for database operations
type Repository interface {
	CreateMessage(ctx context.Context, message *model.Message) error
	GetChatHistory(ctx context.Context, chatID string) ([]model.Message, error)
	GetMessage(ctx context.Context, messageID string) (*model.Message, error)
	DeleteMessage(ctx context.Context, messageID string) error
}

// Cache defines the interface for caching operations
type Cache interface {
	GetChatHistory(ctx context.Context, chatID string) ([]model.Message, error)
	SetChatHistory(ctx context.Context, chatID string, messages []model.Message) error
	DeleteChatHistory(ctx context.Context, chatID string) error
}
