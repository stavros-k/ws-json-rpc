package rpctypes

import (
	"github.com/google/uuid"
)

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

func (e DataCreated) SchemaDescription() string {
	return "User account information"
}

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
type DataCreated struct {
	// The unique identifier for the result
	ID uuid.UUID `json:"id"`
}

// SubscribeParams - Parameters for the Subscribe method
type SubscribeParams struct {
	// The event topic to subscribe to
	Event EventKind   `json:"event"`
	Data  DataCreated `json:"data"`
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
