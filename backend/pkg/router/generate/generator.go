package generate

// RouteMetadataCollector is the interface that RouteBuilder uses to collect route metadata.
type RouteMetadataCollector interface {
	RegisterRoute(route *RouteInfo) error
	Generate() error
}

// NoopCollector is a no-op implementation of RouteMetadataCollector.
type NoopCollector struct{}

func (n *NoopCollector) RegisterRoute(route *RouteInfo) error { return nil }
func (n *NoopCollector) Generate() error                      { return nil }

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
