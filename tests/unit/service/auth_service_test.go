package service_test

import (
	"context"
	"testing"

	"rtcs/internal/model"
	"rtcs/internal/service"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockUserRepository is a mock implementation of the UserRepository interface
type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) Create(ctx context.Context, user *model.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.User), args.Error(1)
}

func (m *MockUserRepository) GetByUsername(ctx context.Context, username string) (*model.User, error) {
	args := m.Called(ctx, username)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.User), args.Error(1)
}

func TestAuthService_Register(t *testing.T) {
	tests := []struct {
		name          string
		username      string
		password      string
		mockSetup     func(*MockUserRepository)
		expectedError bool
	}{
		{
			name:     "successful registration",
			username: "test@example.com",
			password: "password123",
			mockSetup: func(m *MockUserRepository) {
				m.On("GetByUsername", mock.Anything, "test@example.com").Return(nil, nil)
				m.On("Create", mock.Anything, mock.MatchedBy(func(user *model.User) bool {
					return user.Username == "test@example.com"
				})).Return(nil)
			},
			expectedError: false,
		},
		{
			name:     "user already exists",
			username: "existing@example.com",
			password: "password123",
			mockSetup: func(m *MockUserRepository) {
				m.On("GetByUsername", mock.Anything, "existing@example.com").Return(&model.User{
					Username: "existing@example.com",
				}, nil)
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock repository
			mockRepo := &MockUserRepository{}
			tt.mockSetup(mockRepo)

			// Create auth service with mock repository
			authService := service.NewAuthService(mockRepo)

			// Test registration
			_, err := authService.Register(context.Background(), tt.username, tt.password)

			// Assert results
			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			// Verify mock expectations
			mockRepo.AssertExpectations(t)
		})
	}
}

func TestAuthService_Login(t *testing.T) {
	tests := []struct {
		name          string
		username      string
		password      string
		mockSetup     func(*MockUserRepository)
		expectedError bool
	}{
		{
			name:     "successful login",
			username: "test@example.com",
			password: "password123",
			mockSetup: func(m *MockUserRepository) {
				m.On("GetByUsername", mock.Anything, "test@example.com").Return(&model.User{
					Username: "test@example.com",
					Password: "$2a$10$abcdefghijklmnopqrstuvwxyz", // Mocked hashed password
				}, nil)
			},
			expectedError: false,
		},
		{
			name:     "user not found",
			username: "nonexistent@example.com",
			password: "password123",
			mockSetup: func(m *MockUserRepository) {
				m.On("GetByUsername", mock.Anything, "nonexistent@example.com").Return(nil, nil)
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock repository
			mockRepo := &MockUserRepository{}
			tt.mockSetup(mockRepo)

			// Create auth service with mock repository
			authService := service.NewAuthService(mockRepo)

			// Test login
			token, err := authService.Login(context.Background(), tt.username, tt.password)

			// Assert results
			if tt.expectedError {
				assert.Error(t, err)
				assert.Empty(t, token)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, token)
			}

			// Verify mock expectations
			mockRepo.AssertExpectations(t)
		})
	}
}
