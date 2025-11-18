package typesystem

// TypeKind represents the kind of a type definition.
type TypeKind string

// Type kind constants
const (
	TypeKindEnum   TypeKind = "enum"
	TypeKindObject TypeKind = "object"
	TypeKindAlias  TypeKind = "alias"
	TypeKindMap    TypeKind = "map"
	TypeKindArray  TypeKind = "array"
)

// PrimitiveType represents a primitive type.
type PrimitiveType string

// Primitive type constants
const (
	PrimitiveTypeString  PrimitiveType = "string"
	PrimitiveTypeNumber  PrimitiveType = "number"
	PrimitiveTypeInteger PrimitiveType = "integer"
	PrimitiveTypeBoolean PrimitiveType = "boolean"
)

// StringFormat represents a string format.
type StringFormat string

// String format constants
const (
	StringFormatDateTime StringFormat = "date-time"
	StringFormatUUID     StringFormat = "uuid"
)
