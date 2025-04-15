package service

import (
	"context"
	"testing"

	"rtcs/internal/model"

	"github.com/google/uuid"
)

// MockRepository implements the MessageRepository interface for testing
type MockRepository struct {
	messages map[string]*model.Message
}

func NewMockRepository() *MockRepository {
	return &MockRepository{
		messages: make(map[string]*model.Message),
	}
}

func (m *MockRepository) SaveMessage(ctx context.Context, message *model.Message) error {
	m.messages[message.ID.String()] = message
	return nil
}

func (m *MockRepository) GetMessages(ctx context.Context, chatID uuid.UUID, limit int) ([]*model.Message, error) {
	var messages []*model.Message
	for _, msg := range m.messages {
		if msg.ChatID == chatID {
			messages = append(messages, msg)
		}
	}
	return messages, nil
}

func (m *MockRepository) GetMessage(ctx context.Context, messageID uuid.UUID) (*model.Message, error) {
	if msg, ok := m.messages[messageID.String()]; ok {
		return msg, nil
	}
	return nil, nil
}

func (m *MockRepository) DeleteMessage(ctx context.Context, messageID uuid.UUID) error {
	delete(m.messages, messageID.String())
	return nil
}

func (m *MockRepository) CreateChatIfNotExists(ctx context.Context, chat *model.Chat) error {
	return nil
}

func (m *MockRepository) AddUserToChat(ctx context.Context, chatID, userID uuid.UUID) error {
	return nil
}

// MockCache implements the MessageCache interface for testing
type MockCache struct {
	cache map[string][]*model.Message
}

func NewMockCache() *MockCache {
	return &MockCache{
		cache: make(map[string][]*model.Message),
	}
}

func (m *MockCache) SetMessage(ctx context.Context, message *model.Message) error {
	chatIDStr := message.ChatID.String()
	if messages, ok := m.cache[chatIDStr]; ok {
		messages = append(messages, message)
		m.cache[chatIDStr] = messages
	} else {
		m.cache[chatIDStr] = []*model.Message{message}
	}
	return nil
}

func (m *MockCache) GetMessage(ctx context.Context, messageID string) (*model.Message, error) {
	for _, messages := range m.cache {
		for _, msg := range messages {
			if msg.ID.String() == messageID {
				return msg, nil
			}
		}
	}
	return nil, nil
}

func (m *MockCache) DeleteMessage(ctx context.Context, messageID string) error {
	for chatID, messages := range m.cache {
		for i, msg := range messages {
			if msg.ID.String() == messageID {
				m.cache[chatID] = append(messages[:i], messages[i+1:]...)
				return nil
			}
		}
	}
	return nil
}

func (m *MockCache) SetChatMessages(ctx context.Context, chatID string, messages []*model.Message) error {
	m.cache[chatID] = messages
	return nil
}

func (m *MockCache) GetChatMessages(ctx context.Context, chatID string) ([]*model.Message, error) {
	if messages, ok := m.cache[chatID]; ok {
		return messages, nil
	}
	return nil, nil
}

func TestSendMessage(t *testing.T) {
	// Create mock dependencies
	repo := NewMockRepository()
	cache := NewMockCache()
	svc := NewMessageService(repo, cache)

	ctx := context.Background()

	// Test case 1: Send a valid message
	t.Run("Send valid message", func(t *testing.T) {
		chatID := uuid.New().String()
		userID := uuid.New().String()
		message, err := svc.SendMessage(ctx, chatID, userID, "Hello, world!")
		if err != nil {
			t.Fatalf("SendMessage failed: %v", err)
		}

		// Verify message fields
		if message == nil {
			t.Fatal("Message is nil")
		}
		if message.ChatID.String() != chatID {
			t.Errorf("Expected ChatID '%s', got '%s'", chatID, message.ChatID)
		}
		if message.SenderID.String() != userID {
			t.Errorf("Expected SenderID '%s', got '%s'", userID, message.SenderID)
		}
		if message.Text != "Hello, world!" {
			t.Errorf("Expected Text 'Hello, world!', got '%s'", message.Text)
		}
		if message.ID == uuid.Nil {
			t.Error("Message ID is nil")
		}
		if message.CreatedAt.IsZero() {
			t.Error("CreatedAt is zero")
		}

		// Verify message was saved in repository
		savedMessage, err := repo.GetMessage(ctx, message.ID)
		if err != nil {
			t.Fatalf("GetMessage failed: %v", err)
		}
		if savedMessage == nil {
			t.Error("Message not found in repository")
		}

		// Verify message was cached
		cachedMessage, err := cache.GetMessage(ctx, message.ID.String())
		if err != nil {
			t.Fatalf("GetMessage from cache failed: %v", err)
		}
		if cachedMessage == nil {
			t.Error("Message not found in cache")
		}
	})

	// Test case 2: Send message with empty text
	t.Run("Send message with empty text", func(t *testing.T) {
		message, err := svc.SendMessage(ctx, uuid.New().String(), uuid.New().String(), "")
		if err == nil {
			t.Error("Expected error for empty text, got nil")
		}
		if message != nil {
			t.Error("Expected nil message for empty text")
		}
	})
}
