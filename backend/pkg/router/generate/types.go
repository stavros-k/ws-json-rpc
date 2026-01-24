package generate

import (
	"github.com/coder/guts/bindings"
)

// ExternalType represents an external Go type with metadata for OpenAPI generation.
type ExternalType struct {
	bindings.LiteralKeyword

	GoType         string // Original Go type (e.g., "time.Time")
	TypeScriptType string // TypeScript representation (e.g., "string")
	OpenAPIFormat  string // OpenAPI format (e.g., "date-time")
}

// Type kind constants for TypeInfo.
const (
	TypeKindObject     = "object"
	TypeKindStringEnum = "string_enum"
	TypeKindNumberEnum = "number_enum"
	TypeKindUnion      = "union"
	TypeKindAlias      = "alias"
)

// Field type kind constants for FieldType.
const (
	FieldKindPrimitive = "primitive"
	FieldKindArray     = "array"
	FieldKindReference = "reference"
	FieldKindEnum      = "enum"
	FieldKindObject    = "object"
	FieldKindUnknown   = "unknown"
)

// TypeInfo contains comprehensive metadata about a Go type extracted from guts.
type TypeInfo struct {
	Name         string      `json:"name"`        // Type name (e.g., "PingResponse")
	Kind         string      `json:"kind"`        // "Object", "String Enum", "Array", etc.
	Description  string      `json:"description"` // Type-level documentation
	Fields       []FieldInfo `json:"fields,omitempty"`
	EnumValues   []EnumValue `json:"enumValues,omitempty"`
	References   []string    `json:"references,omitempty"`   // Types this type references
	ReferencedBy []string    `json:"referencedBy,omitempty"` // Types that reference this type
	UsedBy       []UsageInfo `json:"usedBy,omitempty"`       // Operations/routes that use this type
	GoType       string      `json:"goType,omitempty"`       // Original Go type (for external types)
}

// FieldType represents the structured type information for a field.
type FieldType struct {
	Kind      string     `json:"kind"`                // "primitive", "array", "reference", "enum", "object", "unknown"
	Type      string     `json:"type"`                // Base type: "string", "User", etc.
	Format    string     `json:"format,omitempty"`    // OpenAPI format (e.g., "date-time")
	Required  bool       `json:"required"`            // Whether the field is required
	Nullable  bool       `json:"nullable,omitempty"`  // For nullable types (T | null)
	ItemsType *FieldType `json:"itemsType,omitempty"` // For arrays: type of array elements
}

// FieldInfo describes a field in a struct (used in high-level API documentation).
type FieldInfo struct {
	Name        string    `json:"name"`
	DisplayType string    `json:"displayType"` // Human-readable type string (e.g., "User[]", "string | null")
	TypeInfo    FieldType `json:"typeInfo"`    // Structured type information
	Description string    `json:"description,omitempty"`
	GoType      string    `json:"goType,omitempty"` // Original Go type (for external types like time.Time)
}

// FieldMetadata is the raw field metadata from guts (used internally).
type FieldMetadata struct {
	Name        string
	Type        string
	Description string
	Optional    bool // Note: This is the inverse of Required in FieldInfo
	EnumValues  []EnumValue
}

// EnumValue represents an enum constant with its documentation.
type EnumValue struct {
	Value       string `json:"value"`
	Description string `json:"description,omitempty"`
}

// UsageInfo tracks where a type is used in operations/routes.
type UsageInfo struct {
	OperationID string `json:"operationID"` // Route operationID
	Role        string `json:"role"`        // "request", "response", "parameter"
}

// RouteInfo contains metadata about a REST route.
type RouteInfo struct {
	OperationID string               `json:"operationID"`
	Method      string               `json:"-"` // Not serialized - used as map key
	Path        string               `json:"-"` // Not serialized - used as map key
	Summary     string               `json:"summary"`
	Description string               `json:"description"`
	Tags        []string             `json:"tags"`
	Deprecated  bool                 `json:"deprecated,omitempty"`
	Request     *RequestInfo         `json:"request,omitempty"`
	Parameters  []ParameterInfo      `json:"parameters,omitempty"`
	Responses   map[int]ResponseInfo `json:"responses"`
}

// RequestInfo describes a request body.
type RequestInfo struct {
	Type        string         `json:"type"`
	Description string         `json:"description,omitempty"`
	Examples    map[string]any `json:"examples,omitempty"`
}

// ParameterInfo describes a route parameter.
type ParameterInfo struct {
	Name        string `json:"name"`
	In          string `json:"in"` // "path", "query", "header"
	Type        string `json:"type"`
	Description string `json:"description"`
	Required    bool   `json:"required"`
}

// ResponseInfo describes a response.
type ResponseInfo struct {
	StatusCode  int            `json:"statusCode"`
	Type        string         `json:"type,omitempty"` // Empty for responses without body
	Description string         `json:"description"`
	Examples    map[string]any `json:"examples,omitempty"`
}

type APIInfo struct {
	Title       string       `json:"title"`
	Version     string       `json:"version"`
	Description string       `json:"description"`
	Servers     []ServerInfo `json:"servers"`
}

// PathRoutes groups routes by HTTP method for a given path.
type PathRoutes struct {
	Routes map[string]*RouteInfo `json:"routes"` // Keyed by HTTP method (GET, POST, etc.)
}

// APIDocumentation is the complete API documentation structure.
type APIDocumentation struct {
	Info           APIInfo                `json:"info"`
	Types          map[string]*TypeInfo   `json:"types"`
	Routes         map[string]*PathRoutes `json:"routes"` // Keyed by path
	DatabaseSchema string                 `json:"databaseSchema"`
}

type ServerInfo struct {
	URL         string
	Description string
}
