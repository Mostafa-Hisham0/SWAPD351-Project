package service

import (
	"context"
	"fmt"
	"time"

	"rtcs/internal/model"

	"github.com/google/uuid"
)

// Message represents a chat message
type Message struct {
	ID        string    `json:"id"`
	ChatID    string    `json:"chat_id"`
	SenderID  string    `json:"sender_id"`
	Text      string    `json:"text"`
	CreatedAt time.Time `json:"created_at"`
}

type MessageRepository interface {
	SaveMessage(ctx context.Context, message *model.Message) error
	GetMessages(ctx context.Context, chatID uuid.UUID, limit int) ([]*model.Message, error)
	GetMessage(ctx context.Context, messageID uuid.UUID) (*model.Message, error)
	DeleteMessage(ctx context.Context, messageID uuid.UUID) error
	CreateChatIfNotExists(ctx context.Context, chat *model.Chat) error
	AddUserToChat(ctx context.Context, chatID, userID uuid.UUID) error
}

type MessageCache interface {
	SetMessage(ctx context.Context, message *model.Message) error
	GetMessage(ctx context.Context, messageID string) (*model.Message, error)
	DeleteMessage(ctx context.Context, messageID string) error
	SetChatMessages(ctx context.Context, chatID string, messages []*model.Message) error
	GetChatMessages(ctx context.Context, chatID string) ([]*model.Message, error)
}

// MessageService defines the interface for message operations
type MessageService struct {
	repo  MessageRepository
	cache MessageCache
}

// NewMessageService creates a new message service
func NewMessageService(repo MessageRepository, cache MessageCache) *MessageService {
	return &MessageService{
		repo:  repo,
		cache: cache,
	}
}

// SendMessage creates a new message
func (s *MessageService) SendMessage(ctx context.Context, chatIDStr, senderIDStr, text string) (*model.Message, error) {
	// Validate input
	if text == "" {
		return nil, fmt.Errorf("message text cannot be empty")
	}
	if chatIDStr == "" {
		return nil, fmt.Errorf("chat ID cannot be empty")
	}

	chatID, err := uuid.Parse(chatIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid chat ID: %w", err)
	}
	senderID, err := uuid.Parse(senderIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid sender ID: %w", err)
	}

	// Create a new chat if it doesn't exist
	chat := &model.Chat{
		ID:   chatID,
		Name: fmt.Sprintf("Chat %s", chatID.String()),
	}
	if err := s.repo.CreateChatIfNotExists(ctx, chat); err != nil {
		return nil, fmt.Errorf("failed to create chat: %w", err)
	}

	// Add user to chat if not already a member
	if err := s.repo.AddUserToChat(ctx, chatID, senderID); err != nil {
		return nil, fmt.Errorf("failed to add user to chat: %w", err)
	}

	message := &model.Message{
		ID:        uuid.New(),
		ChatID:    chatID,
		SenderID:  senderID,
		Text:      text,
		CreatedAt: time.Now(),
	}

	if err := s.repo.SaveMessage(ctx, message); err != nil {
		return nil, err
	}

	if err := s.cache.SetMessage(ctx, message); err != nil {
		// Log error but don't fail the request
		// TODO: Add proper logging
	}

	return message, nil
}

// GetChatHistory retrieves messages for a chat
func (s *MessageService) GetChatHistory(ctx context.Context, chatIDStr string, limit int) ([]*model.Message, error) {
	chatID, err := uuid.Parse(chatIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid chat ID: %w", err)
	}

	// Try to get from cache first
	if messages, err := s.cache.GetChatMessages(ctx, chatIDStr); err == nil && messages != nil {
		return messages, nil
	}

	// If not in cache, get from database
	messages, err := s.repo.GetMessages(ctx, chatID, limit)
	if err != nil {
		return nil, err
	}

	// Update cache
	if err := s.cache.SetChatMessages(ctx, chatIDStr, messages); err != nil {
		// Log error but don't fail the request
		// TODO: Add proper logging
	}

	return messages, nil
}

// DeleteMessage removes a message
func (s *MessageService) DeleteMessage(ctx context.Context, messageIDStr string, userIDStr string) error {
	messageID, err := uuid.Parse(messageIDStr)
	if err != nil {
		return fmt.Errorf("invalid message ID: %w", err)
	}
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return fmt.Errorf("invalid user ID: %w", err)
	}

	// Get the message first to check ownership
	message, err := s.repo.GetMessage(ctx, messageID)
	if err != nil {
		return err
	}

	// Check if the user owns the message
	if message.SenderID != userID {
		return fmt.Errorf("unauthorized: user does not own this message")
	}

	// Delete from database first
	if err := s.repo.DeleteMessage(ctx, messageID); err != nil {
		return err
	}

	// Delete from cache
	if err := s.cache.DeleteMessage(ctx, messageIDStr); err != nil {
		// Log error but don't fail the request
		// TODO: Add proper logging
	}

	return nil
}
