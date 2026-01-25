package generate

import "github.com/coder/guts/bindings"

// RouteMetadataCollector is the interface that RouteBuilder uses to collect route metadata.
type RouteMetadataCollector interface {
	RegisterRoute(route *RouteInfo) error
	Generate() error
}

// NoopCollector is a no-op implementation of RouteMetadataCollector.
type NoopCollector struct{}

func (n *NoopCollector) RegisterRoute(route *RouteInfo) error { return nil }
func (n *NoopCollector) Generate() error                      { return nil }

// ExternalTypeInfo holds metadata about external Go types.
type ExternalTypeInfo struct {
	GoType        string // Original Go type (e.g., "time.Time")
	OpenAPIFormat string // OpenAPI format (e.g., "date-time")
}

// createTimeTypeKeyword creates a LiteralKeyword for time.Time and registers it as an external type.
// Each call creates a new keyword pointer and registers it in the external types map.
//
//nolint:ireturn
func (g *OpenAPICollector) createTimeTypeKeyword() bindings.ExpressionType {
	keyword := bindings.KeywordString
	keywordPtr := &keyword

	g.externalTypes[keywordPtr] = &ExternalTypeInfo{
		GoType: "time.Time", OpenAPIFormat: "date-time",
	}

	return keywordPtr
}

// createURLTypeKeyword creates a LiteralKeyword for net/url.URL and registers it as an external type.
// Each call creates a new keyword pointer and registers it in the external types map.
//
//nolint:ireturn
func (g *OpenAPICollector) createURLTypeKeyword() bindings.ExpressionType {
	keyword := bindings.KeywordString
	keywordPtr := &keyword

	g.externalTypes[keywordPtr] = &ExternalTypeInfo{
		GoType: "net/url.URL", OpenAPIFormat: "uri",
	}

	return keywordPtr
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
