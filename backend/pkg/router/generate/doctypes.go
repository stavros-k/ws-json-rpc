package generate

// TypeInfo contains comprehensive metadata about a Go type extracted from guts.
type TypeInfo struct {
	Name            string           `json:"name"`            // Type name (e.g., "PingResponse")
	Kind            string           `json:"kind"`            // "Object", "String Enum", "Array", etc.
	Description     string           `json:"description"`     // Type-level documentation
	Deprecated      *DeprecationInfo `json:"deprecated"`      // Deprecation information
	Fields          []FieldInfo      `json:"fields"`          // For object types: fields
	EnumValues      []EnumValue      `json:"enumValues"`      // For enum types: enum constants
	References      []string         `json:"references"`      // Types this type references
	ReferencedBy    []string         `json:"referencedBy"`    // Types that reference this type
	UsedBy          []UsageInfo      `json:"usedBy"`          // Operations/routes that use this type
	Representations Representations  `json:"representations"` // JSON, JSON Schema, and TypeScript representations of the type
	GoType          string           `json:"-"`               // Original Go type (for external types)
}

type Representations struct {
	JSON       string `json:"json"`       // JSON representation of the type (zero value example)
	JSONSchema string `json:"jsonSchema"` // JSON Schema representation of the type
	TS         string `json:"ts"`         // TypeScript representation of the type
}

// FieldType represents the structured type information for a field.
type FieldType struct {
	Kind      string     `json:"kind"`      // "primitive", "array", "reference", "enum", "object", "unknown"
	Type      string     `json:"type"`      // Base type: "string", "User", etc.
	Format    string     `json:"format"`    // OpenAPI format (e.g., "date-time")
	Required  bool       `json:"required"`  // Whether the field is required
	Nullable  bool       `json:"nullable"`  // For nullable types (T | null)
	ItemsType *FieldType `json:"itemsType"` // For arrays: type of array elements
}

// FieldInfo describes a field in a struct (used in high-level API documentation).
type FieldInfo struct {
	Name        string           `json:"name"`        // Field name
	DisplayType string           `json:"displayType"` // Human-readable type string (e.g., "User[]", "string | null")
	TypeInfo    FieldType        `json:"typeInfo"`    // Structured type information
	Description string           `json:"description"` // Field documentation
	Deprecated  *DeprecationInfo `json:"deprecated"`  // Deprecation information
	GoType      string           `json:"goType"`      // Original Go type (for external types like time.Time)
}

// EnumValue represents an enum constant with its documentation.
type EnumValue struct {
	Value       string           `json:"value"`
	Description string           `json:"description"`
	Deprecated  *DeprecationInfo `json:"deprecated"`
}

// DeprecationInfo contains deprecation details.
type DeprecationInfo struct {
	Message string `json:"message"` // Deprecation message
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
	Deprecated  bool                 `json:"deprecated"`
	Request     *RequestInfo         `json:"request"`
	Parameters  []ParameterInfo      `json:"parameters"`
	Responses   map[int]ResponseInfo `json:"responses"`
}

// RequestInfo describes a request body.
type RequestInfo struct {
	TypeName            string            `json:"type"` // Extracted type name (set by generator)
	TypeValue           any               `json:"-"`    // Zero value of the type (set by route builder)
	Description         string            `json:"description"`
	ExamplesStringified map[string]string `json:"examples"`
	Examples            map[string]any    `json:"-"`
}

// ParameterInfo describes a route parameter.
type ParameterInfo struct {
	Name        string `json:"name"`
	In          string `json:"in"`   // "path", "query", "header"
	TypeName    string `json:"type"` // Extracted type name (set by generator)
	TypeValue   any    `json:"-"`    // Zero value of the type (set by route builder)
	Description string `json:"description"`
	Required    bool   `json:"required"`
}

// ResponseInfo describes a response.
type ResponseInfo struct {
	StatusCode          int               `json:"statusCode"`
	TypeName            string            `json:"type"` // Extracted type name (set by generator), empty for responses without body
	TypeValue           any               `json:"-"`    // Zero value of the type (set by route builder)
	Description         string            `json:"description"`
	ExamplesStringified map[string]string `json:"examples"`
	Examples            map[string]any    `json:"-"`
}

// PathRoutes groups routes by HTTP method for a given path.
type PathRoutes struct {
	Verbs map[string]*RouteInfo `json:"verbs"` // Keyed by HTTP method (GET, POST, etc.)
}

// APIDocumentation is the complete API documentation structure.
type APIDocumentation struct {
	Info           APIInfo                `json:"info"`
	Types          map[string]*TypeInfo   `json:"types"`
	Routes         map[string]*PathRoutes `json:"routes"` // Keyed by path
	DatabaseSchema string                 `json:"databaseSchema"`
}

// APIInfo contains API metadata.
type APIInfo struct {
	Title       string       `json:"title"`
	Version     string       `json:"version"`
	Description string       `json:"description"`
	Servers     []ServerInfo `json:"servers"`
}

// ServerInfo contains server information.
type ServerInfo struct {
	URL         string
	Description string
}
