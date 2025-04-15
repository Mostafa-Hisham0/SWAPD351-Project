package service

import (
	"context"
	"testing"

	"rtcs/internal/model"
	"rtcs/internal/repository"

	"github.com/google/uuid"
)

type mockRepository struct {
	repository.Repository
	chats     map[uuid.UUID]*model.Chat
	chatUsers map[uuid.UUID]map[uuid.UUID]bool
	createErr error
	getErr    error
	listErr   error
	addErr    error
	removeErr error
}

func (m *mockRepository) CreateChat(ctx context.Context, chat *model.Chat) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.chats[chat.ID] = chat
	return nil
}

func (m *mockRepository) GetChat(ctx context.Context, id uuid.UUID) (*model.Chat, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	return m.chats[id], nil
}

func (m *mockRepository) ListChats(ctx context.Context, userID uuid.UUID) ([]*model.Chat, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	var chats []*model.Chat
	for id, chat := range m.chats {
		if m.chatUsers[id][userID] {
			chats = append(chats, chat)
		}
	}
	return chats, nil
}

func (m *mockRepository) AddUserToChat(ctx context.Context, chatID, userID uuid.UUID) error {
	if m.addErr != nil {
		return m.addErr
	}
	if m.chatUsers[chatID] == nil {
		m.chatUsers[chatID] = make(map[uuid.UUID]bool)
	}
	m.chatUsers[chatID][userID] = true
	return nil
}

func (m *mockRepository) RemoveUserFromChat(ctx context.Context, chatID, userID uuid.UUID) error {
	if m.removeErr != nil {
		return m.removeErr
	}
	delete(m.chatUsers[chatID], userID)
	return nil
}

func TestChatService_CreateChat(t *testing.T) {
	ctx := context.Background()
	repo := &mockRepository{
		chats:     make(map[uuid.UUID]*model.Chat),
		chatUsers: make(map[uuid.UUID]map[uuid.UUID]bool),
	}
	service := NewChatService(repo)

	creatorID := uuid.New()
	chat, err := service.CreateChat(ctx, "test chat", creatorID)
	if err != nil {
		t.Fatalf("CreateChat failed: %v", err)
	}

	if chat.Name != "test chat" {
		t.Errorf("Expected chat name 'test chat', got '%s'", chat.Name)
	}

	// Verify creator was added to chat
	if !repo.chatUsers[chat.ID][creatorID] {
		t.Error("Creator was not added to chat")
	}
}

func TestChatService_JoinLeaveChat(t *testing.T) {
	ctx := context.Background()
	repo := &mockRepository{
		chats:     make(map[uuid.UUID]*model.Chat),
		chatUsers: make(map[uuid.UUID]map[uuid.UUID]bool),
	}
	service := NewChatService(repo)

	// Create a chat
	creatorID := uuid.New()
	chat, _ := service.CreateChat(ctx, "test chat", creatorID)

	// Join chat
	userID := uuid.New()
	err := service.JoinChat(ctx, chat.ID, userID)
	if err != nil {
		t.Fatalf("JoinChat failed: %v", err)
	}

	// Verify user was added
	if !repo.chatUsers[chat.ID][userID] {
		t.Error("User was not added to chat")
	}

	// Leave chat
	err = service.LeaveChat(ctx, chat.ID, userID)
	if err != nil {
		t.Fatalf("LeaveChat failed: %v", err)
	}

	// Verify user was removed
	if repo.chatUsers[chat.ID][userID] {
		t.Error("User was not removed from chat")
	}
}

func TestChatService_CreatorCannotLeave(t *testing.T) {
	ctx := context.Background()
	repo := &mockRepository{
		chats:     make(map[uuid.UUID]*model.Chat),
		chatUsers: make(map[uuid.UUID]map[uuid.UUID]bool),
	}
	service := NewChatService(repo)

	// Create a chat
	creatorID := uuid.New()
	chat, _ := service.CreateChat(ctx, "test chat", creatorID)

	// Try to leave as creator
	err := service.LeaveChat(ctx, chat.ID, creatorID)
	if err == nil {
		t.Error("Expected error when creator tries to leave chat")
	}
}
