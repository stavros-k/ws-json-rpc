package generate

import "github.com/coder/guts/bindings"

// RouteMetadataCollector is the interface that RouteBuilder uses to collect route metadata.
type RouteMetadataCollector interface {
	RegisterRoute(route *RouteInfo)
	Generate() error
}

// NoopCollector is a no-op implementation of RouteMetadataCollector.
type NoopCollector struct{}

func (n *NoopCollector) RegisterRoute(route *RouteInfo) {}
func (n *NoopCollector) Generate() error                { return nil }

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
