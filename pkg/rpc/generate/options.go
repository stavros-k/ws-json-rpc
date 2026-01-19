package generate

// MethodMapping represents a method's request/response types for API client type generation.
type MethodMapping struct {
	Name       string // Method name (e.g., "user.create")
	ParamType  string // Request type name or "null"
	ResultType string // Response type name or "null"
}

// EventMapping represents an event's data type for API client type generation.
type EventMapping struct {
	Name       string // Event name (e.g., "data.created")
	ResultType string // Event data type name or "null"
}
