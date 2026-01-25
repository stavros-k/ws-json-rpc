package httpapi

import (
	"time"

	"ws-json-rpc/backend/pkg/types"
)

// PingResponse is the response to a ping request.
type PingResponse struct {
	// Human-readable message
	Message string `json:"message"`
	// Status of the ping
	Status   PingStatus `json:"status"`
	Metadata *string    `json:"metadata,omitempty"`
}

// PingStatus represents the status of a ping request.
type PingStatus string

const (
	// PingStatusOK means the ping was successful.
	PingStatusOK PingStatus = "OK"
	// PingStatusError means there was an error with the ping.
	// Deprecated: Use PingStatusErrorV2 instead.
	PingStatusError PingStatus = "ERROR"
)

// CreateUserRequest is the request to create a new user.
type CreateUserRequest struct {
	// Username to create
	Username string `json:"username"`
	// Password to create
	Password string `json:"password"`
}

// CreateUserResponse is the response to a create user request.
type CreateUserResponse struct {
	// ID of the created user
	UserID string `json:"userID"`
	// Creation timestamp
	CreatedAt time.Time `json:"createdAt"`
	// URL to the user
	URL *types.URL `json:"url"`
}

// GetTeamRequest is the request to get a team.
type GetTeamRequest struct {
	// ID of the team to get
	TeamID string `json:"teamID"`
}

// User represents a user in the system.
type User struct {
	// ID of the user
	UserID string `json:"userID"`
	// Name of the user
	// Deprecated: Use UserNameV2 instead.
	Name string `json:"name"`
}

// GetTeamResponse is the response to a get team request.
// Deprecated: Use GetTeamResponseV2 instead.
type GetTeamResponse struct {
	// ID of the team
	TeamID string `json:"teamID"`
	// Users in the team
	Users []User `json:"users"`
}
