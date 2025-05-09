package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type User struct {
	ID          uuid.UUID  `json:"id" gorm:"primaryKey;type:uuid;default:uuid_generate_v4()"`
	CreatedAt   time.Time  `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt   time.Time  `json:"updated_at" gorm:"autoUpdateTime"`
	DeletedAt   *time.Time `json:"-" gorm:"index"`
	Username    string     `json:"username" gorm:"uniqueIndex;not null"`
	Password    string     `json:"-" gorm:"not null"` // Don't expose password in JSON
	DisplayName string     `json:"display_name" gorm:"type:varchar(255)"`
	AvatarURL   string     `json:"avatar_url" gorm:"type:varchar(512)"`
	About       string     `json:"about" gorm:"type:text"`
	Email       string     `gorm:"unique"`
	Name        string
	Picture     string
	AuthType    string `gorm:"default:'local'"`
}

// UserProfile represents the public profile of a user
type UserProfile struct {
	ID          uuid.UUID `json:"id"`
	Username    string    `json:"username"`
	DisplayName string    `json:"display_name"`
	AvatarURL   string    `json:"avatar_url"`
	About       string    `json:"about"`
}

// ToProfile converts a User to a UserProfile
func (u *User) ToProfile() *UserProfile {
	return &UserProfile{
		ID:          u.ID,
		Username:    u.Username,
		DisplayName: u.DisplayName,
		AvatarURL:   u.AvatarURL,
		About:       u.About,
	}
}

// BeforeCreate is called before creating a new user
func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	return nil
}
