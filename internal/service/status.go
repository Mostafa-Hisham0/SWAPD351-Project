package service

import (
	"context"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	// Key prefix for user status in Redis
	userStatusPrefix = "user:status:"
	// How long a user status stays valid without updates
	userStatusTTL = 300 * time.Second // Increased TTL to 5 minutes
)

// UserStatus represents a user's online status
type UserStatus struct {
	UserID   string    `json:"user_id"`
	Status   string    `json:"status"` // "online" or "offline"
	LastSeen time.Time `json:"last_seen"`
}

// StatusService manages user online/offline status
type StatusService struct {
	redisClient *redis.Client
}

// NewStatusService creates a new status service
func NewStatusService(redisClient *redis.Client) *StatusService {
	return &StatusService{
		redisClient: redisClient,
	}
}

// SetUserOnline marks a user as online
func (s *StatusService) SetUserOnline(ctx context.Context, userID string) error {
	key := userStatusPrefix + userID
	log.Printf("[STATUS] Setting user %s as ONLINE", userID)

	// First, check if the key exists
	exists, err := s.redisClient.Exists(ctx, key).Result()
	if err != nil {
		log.Printf("[STATUS ERROR] Error checking if user %s exists: %v", userID, err)
	}

	// Set the user as online
	err = s.redisClient.Set(ctx, key, "online", userStatusTTL).Err()
	if err != nil {
		log.Printf("[STATUS ERROR] Failed to set user %s as online: %v", userID, err)
		return err
	}

	if exists == 0 {
		log.Printf("[STATUS] Created new status entry for user %s", userID)
	} else {
		log.Printf("[STATUS] Updated existing status for user %s to online", userID)
	}

	// Double-check that the status was set correctly
	status, err := s.redisClient.Get(ctx, key).Result()
	if err != nil {
		log.Printf("[STATUS ERROR] Failed to verify status for user %s: %v", userID, err)
	} else {
		log.Printf("[STATUS] Verified user %s status is now: %s", userID, status)
	}

	return nil
}

// SetUserOffline marks a user as offline
func (s *StatusService) SetUserOffline(ctx context.Context, userID string) error {
	key := userStatusPrefix + userID
	log.Printf("[STATUS] Setting user %s as OFFLINE", userID)

	err := s.redisClient.Set(ctx, key, "offline", userStatusTTL).Err()
	if err != nil {
		log.Printf("[STATUS ERROR] Failed to set user %s as offline: %v", userID, err)
		return err
	}

	// Double-check that the status was set correctly
	status, err := s.redisClient.Get(ctx, key).Result()
	if err != nil {
		log.Printf("[STATUS ERROR] Failed to verify status for user %s: %v", userID, err)
	} else {
		log.Printf("[STATUS] Verified user %s status is now: %s", userID, status)
	}

	return nil
}

// GetUserStatus gets a user's online status
func (s *StatusService) GetUserStatus(ctx context.Context, userID string) (string, error) {
	key := userStatusPrefix + userID
	status, err := s.redisClient.Get(ctx, key).Result()

	if err == redis.Nil {
		log.Printf("[STATUS] User %s status not found in Redis, defaulting to offline", userID)
		return "offline", nil // User not found in Redis, consider offline
	}

	if err != nil {
		log.Printf("[STATUS ERROR] Failed to get status for user %s: %v", userID, err)
		return "offline", err
	}

	log.Printf("[STATUS] Retrieved status for user %s: %s", userID, status)
	return status, nil
}

// RefreshUserStatus refreshes a user's TTL to prevent expiration
func (s *StatusService) RefreshUserStatus(ctx context.Context, userID string) error {
	key := userStatusPrefix + userID

	// First check if the key exists
	exists, err := s.redisClient.Exists(ctx, key).Result()
	if err != nil {
		log.Printf("[STATUS ERROR] Error checking if user %s exists: %v", userID, err)
		return err
	}

	if exists == 0 {
		log.Printf("[STATUS] User %s not found during refresh, setting to online", userID)
		return s.SetUserOnline(ctx, userID)
	}

	// Get current status
	status, err := s.redisClient.Get(ctx, key).Result()
	if err != nil {
		log.Printf("[STATUS ERROR] Failed to get status for user %s during refresh: %v", userID, err)
		return err
	}

	// Always set to online during refresh, regardless of previous status
	log.Printf("[STATUS] Refreshing status for user %s (was: %s)", userID, status)
	err = s.redisClient.Set(ctx, key, "online", userStatusTTL).Err()
	if err != nil {
		log.Printf("[STATUS ERROR] Failed to refresh status for user %s: %v", userID, err)
		return err
	}

	log.Printf("[STATUS] Successfully refreshed status for user %s to online", userID)
	return nil
}

// GetAllOnlineUsers gets all currently online users
func (s *StatusService) GetAllOnlineUsers(ctx context.Context) ([]string, error) {
	pattern := userStatusPrefix + "*"
	keys, err := s.redisClient.Keys(ctx, pattern).Result()
	if err != nil {
		log.Printf("[STATUS ERROR] Failed to get keys from Redis: %v", err)
		return nil, err
	}

	log.Printf("[STATUS] Found %d user status keys in Redis", len(keys))
	var onlineUsers []string

	for _, key := range keys {
		userID := key[len(userStatusPrefix):]
		status, err := s.redisClient.Get(ctx, key).Result()
		if err != nil {
			log.Printf("[STATUS ERROR] Failed to get status for user %s: %v", userID, err)
			continue
		}

		log.Printf("[STATUS] User %s has status: %s", userID, status)
		if status == "online" {
			onlineUsers = append(onlineUsers, userID)
		}
	}

	log.Printf("[STATUS] Found %d online users out of %d total users", len(onlineUsers), len(keys))
	return onlineUsers, nil
}

// GetAllUserStatuses gets all user statuses
func (s *StatusService) GetAllUserStatuses(ctx context.Context) (map[string]string, error) {
	pattern := userStatusPrefix + "*"
	keys, err := s.redisClient.Keys(ctx, pattern).Result()
	if err != nil {
		log.Printf("[STATUS ERROR] Failed to get keys from Redis: %v", err)
		return nil, err
	}

	statuses := make(map[string]string)
	for _, key := range keys {
		userID := key[len(userStatusPrefix):]
		status, err := s.redisClient.Get(ctx, key).Result()
		if err != nil {
			log.Printf("[STATUS ERROR] Failed to get status for user %s: %v", userID, err)
			statuses[userID] = "offline" // Default to offline on error
			continue
		}

		statuses[userID] = status
	}

	log.Printf("[STATUS] Retrieved statuses for %d users", len(statuses))
	return statuses, nil
}

// FlushAllStatuses clears all status data (for debugging)
func (s *StatusService) FlushAllStatuses(ctx context.Context) error {
	pattern := userStatusPrefix + "*"
	keys, err := s.redisClient.Keys(ctx, pattern).Result()
	if err != nil {
		return err
	}

	if len(keys) > 0 {
		err = s.redisClient.Del(ctx, keys...).Err()
		if err != nil {
			return err
		}
	}

	log.Printf("[STATUS] Flushed all %d status entries", len(keys))
	return nil
}
