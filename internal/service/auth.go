package service

import (
	"context"
	"errors"
	"rtcs/internal/middleware"
	"rtcs/internal/model"
	"rtcs/internal/repository"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// AuthService handles user authentication
type AuthService struct {
	userRepo *repository.UserRepository
}

// NewAuthService creates a new authentication service
func NewAuthService(userRepo *repository.UserRepository) *AuthService {
	return &AuthService{
		userRepo: userRepo,
	}
}

// LoginRequest represents the login request body
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// LoginResponse represents the login response
type LoginResponse struct {
	Token string `json:"token"`
}

// Login authenticates a user and returns a JWT token
func (s *AuthService) Login(ctx context.Context, username, password string) (string, error) {
	// Get user by username
	user, err := s.userRepo.GetByUsername(ctx, username)
	if err != nil {
		return "", errors.New("invalid credentials")
	}

	// Compare password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return "", errors.New("invalid credentials")
	}

	// Generate token
	token, err := middleware.GenerateToken(user.ID.String())
	if err != nil {
		return "", err
	}

	return token, nil
}

// Register creates a new user
func (s *AuthService) Register(ctx context.Context, username, password string) (*model.User, error) {
	// Check if username already exists
	existingUser, err := s.userRepo.GetByUsername(ctx, username)
	if err == nil && existingUser != nil {
		return nil, errors.New("username already exists")
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	// Create user
	user := &model.User{
		Username: username,
		Password: string(hashedPassword),
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

func (s *AuthService) ValidateToken(ctx context.Context, token string) (uuid.UUID, error) {
	// Validate token
	claims, err := middleware.ValidateToken(token)
	if err != nil {
		return uuid.Nil, err
	}

	// Parse user ID from claims
	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		return uuid.Nil, err
	}

	// Check if user exists
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil || user == nil {
		return uuid.Nil, errors.New("user not found")
	}

	return userID, nil
}
