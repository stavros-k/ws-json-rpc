package rpcapi

import (
	"github.com/google/uuid"
)

// EventDataMap - A map of event data indexed by event ID
type EventDataMap map[string]SomeEvent

// EventKind - All the available event topics
type EventKind string

const (
	// Data created
	EventKindDataCreated EventKind = "data.created"
	// Data updated
	EventKindDataUpdated EventKind = "data.updated"
)

// Valid returns true if the EventKind value is valid
func (e EventKind) Valid() bool {
	switch e {
	case EventKindDataCreated, EventKindDataUpdated:
		return true
	default:
		return false
	}
}

// MethodKind - All the available RPC methods
type MethodKind string

const (
	// Ping
	MethodKindPing MethodKind = "ping"
	// Subscribe
	MethodKindSubscribe MethodKind = "subscribe"
	// Unsubscribe
	MethodKindUnsubscribe MethodKind = "unsubscribe"
	// Create user
	MethodKindUserCreate MethodKind = "user.create"
	// Update user
	MethodKindUserUpdate MethodKind = "user.update"
	// Delete user
	MethodKindUserDelete MethodKind = "user.delete"
	// List users
	MethodKindUserList MethodKind = "user.list"
	// Get user
	MethodKindUserGet MethodKind = "user.get"
)

// Valid returns true if the MethodKind value is valid
func (e MethodKind) Valid() bool {
	switch e {
	case MethodKindPing, MethodKindSubscribe, MethodKindUnsubscribe, MethodKindUserCreate, MethodKindUserUpdate, MethodKindUserDelete, MethodKindUserList, MethodKindUserGet:
		return true
	default:
		return false
	}
}

// PingResult - Result for the Ping method
type PingResult struct {
	// A message describing the result
	Message string `json:"message"`
	// The status of the ping
	Status PingStatus `json:"status"`
}

// PingStatus - Status for the Ping method
type PingStatus string

const (
	// Success
	PingStatusSuccess PingStatus = "success"
	// Error
	PingStatusError PingStatus = "error"
)

// Valid returns true if the PingStatus value is valid
func (e PingStatus) Valid() bool {
	switch e {
	case PingStatusSuccess, PingStatusError:
		return true
	default:
		return false
	}
}

// SomeEvent - Result for the SomeEvent method
type SomeEvent struct {
	// The unique identifier for the result
	ID uuid.UUID `json:"id"`
}

// StatusResult - Result for the Status method
type StatusResult PingResult

// StringMap - A map with string values for storing key-value pairs
type StringMap map[string]string

// SubscribeParams - Parameters for the Subscribe method
type SubscribeParams struct {
	// The event topic to subscribe to
	Event EventKind `json:"event"`
}

// SubscribeResult - Result for the Subscribe method
type SubscribeResult struct {
	// Whether the subscribe was successful
	Success bool `json:"success"`
}

// UnsubscribeParams - Parameters for the Unsubscribe method
type UnsubscribeParams struct {
	// The event topic to unsubscribe from
	Event EventKind `json:"event"`
}

// UnsubscribeResult - Result for the Unsubscribe method
type UnsubscribeResult struct {
	// Whether the unsubscribe was successful
	Success bool `json:"success"`
}

