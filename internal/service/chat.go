package service

import (
	"context"
	"errors"

	"rtcs/internal/model"
	"rtcs/internal/repository"

	"github.com/google/uuid"
)

type ChatService struct {
	repo repository.Repository
}

func NewChatService(repo repository.Repository) *ChatService {
	return &ChatService{repo: repo}
}

func (s *ChatService) CreateChat(ctx context.Context, name string, creatorID uuid.UUID) (*model.Chat, error) {
	chatID := uuid.New()
	chat := &model.Chat{
		ID:   chatID,
		Name: name,
	}

	if err := s.repo.CreateChat(ctx, chat); err != nil {
		return nil, err
	}

	// Add creator to the chat
	if err := s.repo.AddUserToChat(ctx, chatID, creatorID); err != nil {
		return nil, err
	}

	return chat, nil
}

func (s *ChatService) GetChat(ctx context.Context, id uuid.UUID) (*model.Chat, error) {
	return s.repo.GetChat(ctx, id)
}

func (s *ChatService) ListChats(ctx context.Context, userID uuid.UUID) ([]*model.Chat, error) {
	return s.repo.ListChats(ctx, userID)
}

func (s *ChatService) JoinChat(ctx context.Context, chatID, userID uuid.UUID) error {
	// Check if chat exists
	chat, err := s.repo.GetChat(ctx, chatID)
	if err != nil {
		return err
	}
	if chat == nil {
		return errors.New("chat not found")
	}

	return s.repo.AddUserToChat(ctx, chatID, userID)
}

func (s *ChatService) LeaveChat(ctx context.Context, chatID, userID uuid.UUID) error {
	// Check if chat exists
	chat, err := s.repo.GetChat(ctx, chatID)
	if err != nil {
		return err
	}
	if chat == nil {
		return errors.New("chat not found")
	}

	// Check if user is the creator by checking if they were the first to join
	chatUsers, err := s.repo.ListChats(ctx, userID)
	if err != nil {
		return err
	}
	if len(chatUsers) > 0 && chatUsers[0].ID == chatID {
		return errors.New("chat creator cannot leave the chat")
	}

	return s.repo.RemoveUserFromChat(ctx, chatID, userID)
}
