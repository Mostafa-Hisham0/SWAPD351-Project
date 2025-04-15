package repository

import (
	"context"

	"rtcs/internal/model"

	"github.com/google/uuid"
)

// Repository defines the interface for database operations
type Repository interface {
	// Message methods
	CreateMessage(ctx context.Context, message *model.Message) error
	GetMessage(ctx context.Context, id uuid.UUID) (*model.Message, error)
	ListMessages(ctx context.Context, chatID uuid.UUID, limit, offset int) ([]*model.Message, error)
	DeleteMessage(ctx context.Context, id uuid.UUID) error

	// Chat methods
	CreateChat(ctx context.Context, chat *model.Chat) error
	GetChat(ctx context.Context, id uuid.UUID) (*model.Chat, error)
	ListChats(ctx context.Context, userID uuid.UUID) ([]*model.Chat, error)
	AddUserToChat(ctx context.Context, chatID, userID uuid.UUID) error
	RemoveUserFromChat(ctx context.Context, chatID, userID uuid.UUID) error
}
