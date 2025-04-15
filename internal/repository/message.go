package repository

import (
	"context"
	"time"

	"rtcs/internal/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// MessageRepository handles database operations for messages
type MessageRepository struct {
	db *gorm.DB
}

// NewMessageRepository creates a new message repository
func NewMessageRepository(db *gorm.DB) *MessageRepository {
	return &MessageRepository{
		db: db,
	}
}

// SaveMessage saves a message to the database
func (r *MessageRepository) SaveMessage(ctx context.Context, message *model.Message) error {
	return r.db.WithContext(ctx).Create(message).Error
}

// GetMessages retrieves messages for a chat
func (r *MessageRepository) GetMessages(ctx context.Context, chatID uuid.UUID, limit int) ([]*model.Message, error) {
	var messages []*model.Message
	err := r.db.WithContext(ctx).
		Where("chat_id = ?", chatID).
		Order("created_at DESC").
		Limit(limit).
		Find(&messages).Error
	return messages, err
}

// GetMessage retrieves a message by ID
func (r *MessageRepository) GetMessage(ctx context.Context, messageID uuid.UUID) (*model.Message, error) {
	var message model.Message
	err := r.db.WithContext(ctx).First(&message, "id = ?", messageID).Error
	if err != nil {
		return nil, err
	}
	return &message, nil
}

// DeleteMessage deletes a message
func (r *MessageRepository) DeleteMessage(ctx context.Context, messageID uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&model.Message{}, "id = ?", messageID).Error
}

// CreateChatIfNotExists creates a new chat if it doesn't exist
func (r *MessageRepository) CreateChatIfNotExists(ctx context.Context, chat *model.Chat) error {
	result := r.db.WithContext(ctx).FirstOrCreate(chat, model.Chat{ID: chat.ID})
	return result.Error
}

// AddUserToChat adds a user to a chat if they're not already a member
func (r *MessageRepository) AddUserToChat(ctx context.Context, chatID, userID uuid.UUID) error {
	chatUser := &model.ChatUser{
		ChatID:   chatID,
		UserID:   userID,
		JoinedAt: time.Now(),
	}
	result := r.db.WithContext(ctx).FirstOrCreate(chatUser, model.ChatUser{ChatID: chatID, UserID: userID})
	return result.Error
}
