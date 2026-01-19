package typesystem

// TypeNode is the interface that all type nodes must implement.
type TypeNode interface {
	GetName() string
	GetDescription() string
	GetKind() TypeKind
	GetRawDefinition() string // Returns the raw JSON definition from .type.json
}

// PropertyType represents the type of a property or field.
type PropertyType struct {
	// Primitive type (string, number, integer, boolean) or empty for reference
	Primitive PrimitiveType

	// Format for string types (date-time, uuid, email, uri)
	Format StringFormat

	// Reference to another type
	Ref string

	// For array types
	Items *PropertyType

	// For map types
	MapKey   *PropertyType
	MapValue *PropertyType

	// Nullability
	Nullable bool
}

// FieldMetadata contains metadata about an object field.
type FieldMetadata struct {
	Name        string
	Description string
	Type        *PropertyType
	Optional    bool // If false (default), field is required
	Nullable    bool
}

// CollectRefs recursively collects type references from a PropertyType.
func CollectRefs(pt *PropertyType, refs map[string]struct{}) {
	if pt == nil {
		return
	}
	if pt.Ref != "" {
		refs[pt.Ref] = struct{}{}
	}
	if pt.Items != nil {
		CollectRefs(pt.Items, refs)
	}
	if pt.MapKey != nil {
		CollectRefs(pt.MapKey, refs)
	}
	if pt.MapValue != nil {
		CollectRefs(pt.MapValue, refs)
	}
}
