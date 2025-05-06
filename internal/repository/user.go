package repository

import (
	"context"
	"rtcs/internal/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(ctx context.Context, user *model.User) error {
	return r.db.WithContext(ctx).Create(user).Error
}

func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	var user model.User
	err := r.db.WithContext(ctx).First(&user, "id = ?", id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) GetByUsername(ctx context.Context, username string) (*model.User, error) {
	var user model.User
	err := r.db.WithContext(ctx).First(&user, "username = ?", username).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

// UpdateProfile updates a user's profile information
func (r *UserRepository) UpdateProfile(ctx context.Context, id uuid.UUID, profile *model.UserProfile) error {
	return r.db.WithContext(ctx).Model(&model.User{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"display_name": profile.DisplayName,
			"avatar_url":   profile.AvatarURL,
			"about":        profile.About,
		}).Error
}

// GetProfiles retrieves profiles for multiple users
func (r *UserRepository) GetProfiles(ctx context.Context, userIDs []uuid.UUID) (map[uuid.UUID]*model.UserProfile, error) {
	var users []model.User
	err := r.db.WithContext(ctx).
		Where("id IN ?", userIDs).
		Find(&users).Error

	if err != nil {
		return nil, err
	}

	profiles := make(map[uuid.UUID]*model.UserProfile)
	for _, user := range users {
		profiles[user.ID] = user.ToProfile()
	}

	return profiles, nil
}
