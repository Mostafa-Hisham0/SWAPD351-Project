package service

import (
	"context"
	"errors"
	"rtcs/internal/model"
	"rtcs/internal/repository"

	"github.com/google/uuid"
)

// ProfileService handles user profile operations
type ProfileService struct {
	userRepo *repository.UserRepository
}

// NewProfileService creates a new profile service
func NewProfileService(userRepo *repository.UserRepository) *ProfileService {
	return &ProfileService{
		userRepo: userRepo,
	}
}

// GetProfile retrieves a user's profile
func (s *ProfileService) GetProfile(ctx context.Context, userID uuid.UUID) (*model.UserProfile, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, errors.New("user not found")
	}

	return user.ToProfile(), nil
}

// UpdateProfile updates a user's profile
func (s *ProfileService) UpdateProfile(ctx context.Context, userID uuid.UUID, profile *model.UserProfile) error {
	// Verify user exists
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return err
	}
	if user == nil {
		return errors.New("user not found")
	}

	// Update profile
	return s.userRepo.UpdateProfile(ctx, userID, profile)
}

// GetProfiles retrieves profiles for multiple users
func (s *ProfileService) GetProfiles(ctx context.Context, userIDs []uuid.UUID) (map[uuid.UUID]*model.UserProfile, error) {
	return s.userRepo.GetProfiles(ctx, userIDs)
}
