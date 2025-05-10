package service

import (
	"context"
	"errors"
	"rtcs/internal/model"
	"rtcs/internal/repository"

	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// AuthService handles user authentication
type AuthService struct {
	userRepo  *repository.UserRepository
	jwtSecret []byte
}

// NewAuthService creates a new authentication service
func NewAuthService(userRepo *repository.UserRepository, jwtSecret string) *AuthService {
	return &AuthService{
		userRepo:  userRepo,
		jwtSecret: []byte(jwtSecret),
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

// GetOrCreateGoogleUser gets or creates a user from Google OAuth data
func (s *AuthService) GetOrCreateGoogleUser(ctx context.Context, email, name, picture string) (*model.User, error) {
	// Try to get existing user by email
	user, err := s.userRepo.GetByEmail(ctx, email)
	if err == nil && user != nil {
		return user, nil
	}

	// Create new user if not found
	user = &model.User{
		Email:    email,
		Username: email, // Use email as username for now
		Name:     name,
		Picture:  picture,
		AuthType: "google",
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

// GenerateToken generates a JWT token for a user
func (s *AuthService) GenerateToken(userID string) (string, error) {
	claims := &jwt.RegisteredClaims{
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		NotBefore: jwt.NewNumericDate(time.Now()),
		Subject:   userID,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.jwtSecret)
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
	token, err := s.GenerateToken(user.ID.String())
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
	// Parse and validate token
	parsedToken, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		return s.jwtSecret, nil
	})

	if err != nil {
		return uuid.Nil, err
	}

	if !parsedToken.Valid {
		return uuid.Nil, errors.New("invalid token")
	}

	// Get claims
	claims, ok := parsedToken.Claims.(jwt.RegisteredClaims)
	if !ok {
		return uuid.Nil, errors.New("invalid token claims")
	}

	// Parse user ID from claims
	userID, err := uuid.Parse(claims.Subject)
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
