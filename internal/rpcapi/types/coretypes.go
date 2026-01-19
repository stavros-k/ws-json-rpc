package rpctypes

// EventKind - All the available event topics
type EventKind string

const (
	// Data created
	EventKindDataCreated EventKind = "data.created"
	// Data updated
	EventKindDataUpdated EventKind = "data.updated"
)

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
