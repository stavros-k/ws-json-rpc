package generator

type TypeKind string

const (
	BasicKind   TypeKind = "basic"
	StructKind  TypeKind = "struct"
	EnumKind    TypeKind = "enum"
	SliceKind   TypeKind = "slice"
	ArrayKind   TypeKind = "array"
	MapKind     TypeKind = "map"
	PointerKind TypeKind = "pointer"
)

// Shared interface for all type expressions
type TypeExpression interface {
	String() string
	Kind() TypeKind
}

// Top-level type declaration
type TypeInfo struct {
	Name     string
	Type     TypeExpression // Uses the shared interface
	Comment  Comment
	Position Position
}

// Field in a struct
type FieldInfo struct {
	Name        string
	Type        TypeExpression // Uses the shared interface
	JSONName    string
	JSONOptions []string
	Comment     Comment
	IsEmbedded  bool
}

// Basic type (string, int, etc) or type reference (User, UUID)
type BasicType struct {
	Name string
}

func (b BasicType) Kind() TypeKind { return BasicKind }

// Enum type
type EnumType struct {
	BaseType   string
	EnumValues []EnumValue
}

func (e EnumType) Kind() TypeKind { return EnumKind }

// Struct type
type StructType struct {
	Fields []FieldInfo
}

func (s StructType) Kind() TypeKind { return StructKind }

// Slice type ([]T)
type SliceType struct {
	Element TypeExpression
}

func (s SliceType) Kind() TypeKind { return SliceKind }

// Array type ([N]T)
type ArrayType struct {
	Element TypeExpression
	Length  int
}

func (a ArrayType) Kind() TypeKind { return ArrayKind }

// Map type (map[K]V)
type MapType struct {
	Key   TypeExpression
	Value TypeExpression
}

func (m MapType) Kind() TypeKind { return MapKind }

// Pointer type (*T)
type PointerType struct {
	Element TypeExpression
}

func (p PointerType) Kind() TypeKind { return PointerKind }

// Comment represents comments associated with a type or field
type Comment struct {
	// Comment above the declaration
	Above  string
	Inline string // Comment on the same line
}

// IsEmpty returns true if there are no comments
func (c Comment) IsEmpty() bool {
	return c.Above == "" && c.Inline == ""
}

// EnumValue represents an enum value
type EnumValue struct {
	Name    string
	Value   string // The actual value, (ie "1", "foo", etc)
	Comment Comment
}

// Position represents the location of a type or field
type Position struct {
	Package  string
	Filename string
	Line     int
}
