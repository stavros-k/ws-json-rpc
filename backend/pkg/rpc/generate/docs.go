package generate

// This file (docs.go) defines the data structures for API documentation,
// including types, methods, events, and their associated metadata.

import (
	"errors"
	"ws-json-rpc/backend/pkg/utils"
)

// Ref represents a JSON Schema reference to another type.
// Used in API documentation to link parameters and results to their type definitions.
type Ref struct {
	Ref string `json:"$ref"` // Type name (e.g., "PingParams")
}

// FieldMetadata contains documentation and type information for a single field.
type FieldMetadata struct {
	Name        string   `json:"name"`                  // Field name
	Type        string   `json:"type"`                  // TypeScript type representation
	Description string   `json:"description,omitempty"` // Field description from comments
	Optional    bool     `json:"optional"`              // Whether field is optional (has ?)
	EnumValues  []string `json:"enumValues,omitempty"`  // Possible values if type is an enum/union
}

// UsedBy represents where a type is used (method parameter, method result, or event result).
type UsedBy struct {
	Type   string `json:"type"`   // "method" or "event"
	Target string `json:"target"` // Method/event name
	Role   string `json:"role"`   // "param" or "result"
}

// TypeDocs contains all documentation and code representations for a single type.
// This includes descriptions, examples, and metadata about the type structure.
type TypeDocs struct {
	Description        string          `json:"description"`                  // Human-readable type description
	JsonRepresentation string          `json:"jsonRepresentation,omitempty"` // Example JSON instance (only for explicitly registered types)
	TSType             string          `json:"tsType"`                       // TypeScript type definition
	Kind               string          `json:"kind"`                         // Type kind (e.g., "Object", "String Enum", "Union")
	EnumValues         []string        `json:"enumValues,omitempty"`         // Possible values if type is an enum/union
	Fields             []FieldMetadata `json:"fields,omitempty"`             // Field metadata extracted from TypeScript AST
	References         []string        `json:"references,omitempty"`         // Types this type references
	ReferencedBy       []string        `json:"referencedBy,omitempty"`       // Types that reference this type (computed)
	UsedBy             []UsedBy        `json:"usedBy,omitempty"`             // Methods/events that use this type (computed)
}

// Protocols indicates which communication protocols support a method or event.
type Protocols struct {
	HTTP bool `json:"http"` // Available via HTTP POST
	WS   bool `json:"ws"`   // Available via WebSocket
}

// ErrorDoc documents a possible error that a method can return.
type ErrorDoc struct {
	Title       string `json:"title"`       // Short error name
	Description string `json:"description"` // Detailed error description
	Code        int    `json:"code"`        // Error code
	Message     string `json:"message"`     // Example error message
}

// Example represents a sample request-response pair for a method or event.
// The ParamsObj and ResultObj fields are used to provide actual Go objects,
// which are then serialized to JSON strings in the Params and Result fields.
type Example struct {
	Title       string `json:"title"`       // Example name
	Description string `json:"description"` // What this example demonstrates
	Params      string `json:"params"`      // Serialized params JSON (set automatically)
	Result      string `json:"result"`      // Serialized result JSON (set automatically)

	ResultObj any `json:"-"` // Go object for result (not serialized, used for generation)
	ParamsObj any `json:"-"` // Go object for params (not serialized, used for generation)
}

// Validate ensures that the example uses the object fields (ParamsObj/ResultObj)
// rather than the string fields (Params/Result), which are set automatically.
func (e *Example) Validate() error {
	if e.Params != "" || e.Result != "" {
		return errors.New("example should use ParamsObj and ResultObj fields instead of Params and Result strings")
	}

	return nil
}

// EventDocs contains complete documentation for a WebSocket event.
// Events are unidirectional server-to-client messages.
type EventDocs struct {
	Title       string    `json:"title"`       // Event name
	Description string    `json:"description"` // Detailed description
	Group       string    `json:"group"`       // Logical grouping (e.g., "User", "Game")
	Tags        []string  `json:"tags"`        // Categorization tags
	Deprecated  bool      `json:"deprecated"`  // Whether this event is deprecated
	Protocols   Protocols `json:"protocols"`   // Supported protocols (WS only for events)
	ResultType  Ref       `json:"resultType"`  // Type of the event data
	Examples    []Example `json:"examples"`    // Usage examples
}

// Validate checks that all examples in the event documentation are valid.
func (e *EventDocs) Validate() error {
	for _, ex := range e.Examples {
		if err := ex.Validate(); err != nil {
			return err
		}
	}

	return nil
}

// NoNilSlices ensures slice fields are empty arrays rather than nil for JSON serialization.
// This produces cleaner JSON output ([] instead of null).
func (e *EventDocs) NoNilSlices() {
	if e.Examples == nil {
		e.Examples = make([]Example, 0)
	}

	if e.Tags == nil {
		e.Tags = make([]string, 0)
	}
}

// MethodDocs contains complete documentation for an RPC method.
// Methods are bidirectional request-response calls available over HTTP and/or WebSocket.
type MethodDocs struct {
	Title       string     `json:"title"`       // Method name
	Description string     `json:"description"` // Detailed description
	Group       string     `json:"group"`       // Logical grouping (e.g., "User", "Game")
	Tags        []string   `json:"tags"`        // Categorization tags
	Deprecated  bool       `json:"deprecated"`  // Whether this method is deprecated
	Protocols   Protocols  `json:"protocols"`   // Supported protocols (HTTP and/or WS)
	ResultType  Ref        `json:"resultType"`  // Type of the response
	ParamType   Ref        `json:"paramType"`   // Type of the request parameters
	Examples    []Example  `json:"examples"`    // Usage examples
	Errors      []ErrorDoc `json:"errors"`      // Possible errors

	NoHTTP bool `json:"-"` // Internal flag: if true, disable HTTP support
}

// Validate checks that all examples in the method documentation are valid.
func (m *MethodDocs) Validate() error {
	for _, ex := range m.Examples {
		if err := ex.Validate(); err != nil {
			return err
		}
	}

	return nil
}

// NoNilSlices ensures slice fields are empty arrays rather than nil for JSON serialization.
// This produces cleaner JSON output ([] instead of null).
func (m *MethodDocs) NoNilSlices() {
	if m.Examples == nil {
		m.Examples = make([]Example, 0)
	}

	if m.Errors == nil {
		m.Errors = make([]ErrorDoc, 0)
	}

	if m.Tags == nil {
		m.Tags = make([]string, 0)
	}
}

// Info contains metadata about the API.
type Info struct {
	Title       string `json:"title"`       // API name
	Version     string `json:"version"`     // API version (e.g., "1.0.0")
	Description string `json:"description"` // API description
}

// Docs is the complete API documentation structure.
// This is the top-level object that gets serialized to JSON for the documentation website.
type Docs struct {
	Info           Info                  `json:"info"`           // API metadata
	Methods        map[string]MethodDocs `json:"methods"`        // RPC methods (method name -> docs)
	Events         map[string]EventDocs  `json:"events"`         // WebSocket events (event name -> docs)
	Types          map[string]TypeDocs   `json:"types"`          // Type definitions (type name -> docs)
	DatabaseSchema string                `json:"databaseSchema"` // SQL database schema
}

type DocsOptions struct {
	Title       string
	Description string
}

// NewDocs creates a new Docs instance with default values.
// Initializes empty maps for methods, events, and types, and sets API metadata.
func NewDocs(opt DocsOptions) *Docs {
	return &Docs{
		Info: Info{
			Title:       opt.Title,
			Version:     utils.GetVersionShort(),
			Description: opt.Description,
		},
		Methods: make(map[string]MethodDocs),
		Events:  make(map[string]EventDocs),
		Types:   make(map[string]TypeDocs),
	}
}
