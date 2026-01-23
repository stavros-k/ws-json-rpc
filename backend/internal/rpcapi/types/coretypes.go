package rpctypes

// EventKind - All the available event topics.
type EventKind string

const (
	// Only for internal use.
	// You are warned
	EventKindDataCreated EventKind = "data.created"
	EventKindDataUpdated EventKind = "data.updated"
)

// MethodKind - All the available RPC methods.
type MethodKind string

const (
	MethodKindPing        MethodKind = "ping"
	MethodKindSubscribe   MethodKind = "subscribe"
	MethodKindUnsubscribe MethodKind = "unsubscribe"
	MethodKindUserCreate  MethodKind = "user.create"
	MethodKindUserUpdate  MethodKind = "user.update"
	MethodKindUserDelete  MethodKind = "user.delete"
	MethodKindUserList    MethodKind = "user.list"
	MethodKindUserGet     MethodKind = "user.get"
)
