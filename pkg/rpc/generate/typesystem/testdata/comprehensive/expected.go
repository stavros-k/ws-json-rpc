package models

import (
	"github.com/google/uuid"
	"time"
)

// Status - User status
type Status string

const (
	// User is active
	StatusActive Status = "active"
	// User is inactive
	StatusInactive Status = "inactive"
)

// Valid returns true if the Status value is valid
func (e Status) Valid() bool {
	switch e {
	case StatusActive, StatusInactive:
		return true
	default:
		return false
	}
}

// StringMap - A string map
type StringMap map[string]string

// Tags - A list of tags
type Tags []string

// User - User entity
type User struct {
	// When the user was created
	CreatedAt time.Time `json:"createdAt"`
	// User ID
	ID UserID `json:"id"`
	// User status
	Status Status `json:"status"`
	// User tags
	Tags []string `json:"tags"`
}

// UserID - Unique identifier for a user
type UserID uuid.UUID
