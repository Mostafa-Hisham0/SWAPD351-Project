package model

import (
	"time"

	"github.com/google/uuid"
)

// Chat represents a chat room
type Chat struct {
	ID        uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Name      string     `gorm:"type:varchar(255);not null" json:"name"`
	CreatedAt time.Time  `gorm:"index" json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `gorm:"index" json:"deleted_at,omitempty"`
	Users     []User     `gorm:"many2many:chat_users;" json:"users,omitempty"`
}

// ChatUser represents a user's membership in a chat
type ChatUser struct {
	ChatID   uuid.UUID `gorm:"type:uuid;primaryKey" json:"chat_id"`
	UserID   uuid.UUID `gorm:"type:uuid;primaryKey" json:"user_id"`
	JoinedAt time.Time `gorm:"not null;default:CURRENT_TIMESTAMP" json:"joined_at"`
	Chat     *Chat     `gorm:"foreignKey:ChatID" json:"-"`
	User     *User     `gorm:"foreignKey:UserID" json:"-"`
}
