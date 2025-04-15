package repository

import (
	"context"
	"errors"
	"time"

	"rtcs/internal/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type chatRepository struct {
	db *gorm.DB
}

func NewChatRepository(db *gorm.DB) *chatRepository {
	return &chatRepository{db: db}
}

func (r *chatRepository) CreateMessage(ctx context.Context, message *model.Message) error {
	return r.db.WithContext(ctx).Create(message).Error
}

func (r *chatRepository) GetMessage(ctx context.Context, id uuid.UUID) (*model.Message, error) {
	var message model.Message
	err := r.db.WithContext(ctx).First(&message, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &message, err
}

func (r *chatRepository) ListMessages(ctx context.Context, chatID uuid.UUID, limit, offset int) ([]*model.Message, error) {
	var messages []*model.Message
	err := r.db.WithContext(ctx).
		Where("chat_id = ?", chatID).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&messages).Error
	return messages, err
}

func (r *chatRepository) DeleteMessage(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&model.Message{}, "id = ?", id).Error
}

func (r *chatRepository) CreateChat(ctx context.Context, chat *model.Chat) error {
	return r.db.WithContext(ctx).Create(chat).Error
}

func (r *chatRepository) GetChat(ctx context.Context, id uuid.UUID) (*model.Chat, error) {
	var chat model.Chat
	err := r.db.WithContext(ctx).First(&chat, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &chat, err
}

func (r *chatRepository) ListChats(ctx context.Context, userID uuid.UUID) ([]*model.Chat, error) {
	var chats []*model.Chat
	err := r.db.WithContext(ctx).
		Joins("JOIN chat_users ON chat_users.chat_id = chats.id").
		Where("chat_users.user_id = ?", userID).
		Find(&chats).Error
	return chats, err
}

func (r *chatRepository) AddUserToChat(ctx context.Context, chatID, userID uuid.UUID) error {
	return r.db.WithContext(ctx).Create(&model.ChatUser{
		ChatID:   chatID,
		UserID:   userID,
		JoinedAt: time.Now(),
	}).Error
}

func (r *chatRepository) RemoveUserFromChat(ctx context.Context, chatID, userID uuid.UUID) error {
	return r.db.WithContext(ctx).
		Where("chat_id = ? AND user_id = ?", chatID, userID).
		Delete(&model.ChatUser{}).Error
}
