package rpctypes

import (
	"github.com/google/uuid"
)

// PingResult - Result for the [MethodKindPing] method
type PingResult struct {
	// A message describing the result
	Message string `json:"message"`
	// The status of the ping
	Status PingStatus `json:"status"`
}

// PingStatus - Status for the [MethodKindPing] method
type PingStatus string

const (
	PingStatusSuccess PingStatus = "success"
	PingStatusError   PingStatus = "error"
)

// Valid returns true if the [PingStatus] value is valid
func (e PingStatus) Valid() bool {
	switch e {
	case PingStatusSuccess, PingStatusError:
		return true
	default:
		return false
	}
}

// DataCreatedEvent - Result for the [EventKindDataCreated] event
type DataCreatedEvent struct {
	// The unique identifier for the result
	ID uuid.UUID `json:"id"`
}

// SubscribeParams - Parameters for the [MethodKindSubscribe] method
type SubscribeParams struct {
	// The event topic to subscribe to
	Event EventKind `json:"event"`
}

// SubscribeResult - Result for the [MethodKindSubscribe] method
type SubscribeResult struct {
	// Whether the subscribe was successful
	Success bool `json:"success"`
}

// UnsubscribeParams - Parameters for the [MethodKindUnsubscribe] method
type UnsubscribeParams struct {
	// The event topic to unsubscribe from
	Event EventKind `json:"event"`
}

// UnsubscribeResult - Result for the [MethodKindUnsubscribe] method
type UnsubscribeResult struct {
	// Whether the unsubscribe was successful
	Success bool `json:"success"`
}
