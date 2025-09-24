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

type Comment struct {
	// Comment above the declaration
	Above  string
	Inline string // Comment on the same line
}

func (c Comment) IsEmpty() bool {
	return c.Above == "" && c.Inline == ""
}

type EnumValue struct {
	Name    string
	Value   string // The actual value, (ie "1", "foo", etc)
	Comment Comment
}

type Position struct {
	Package  string
	Filename string
	Line     int
}

type TypeInfo struct {
	Name       string
	Kind       TypeKind
	Underlying TypeDetails // Interface for type-specific details
	Comment    Comment
	Position   Position
}

type FieldTypeInfo struct {
	IsPointer   bool
	IsSlice     bool
	IsArray     bool
	IsMap       bool
	IsEmbedded  bool           // For embedded fields
	BaseType    string         // For simple types: "User", "string", etc.
	KeyType     *FieldTypeInfo // For maps: recursive type info
	ValueType   *FieldTypeInfo // For maps, slices, arrays: recursive type info
	ArrayLength string         // For fixed arrays: [5]int
}

type FieldInfo struct {
	Name        string
	Type        *FieldTypeInfo
	JSONName    string
	JSONOptions []string
	Comment     Comment
}

// Interface for type-specific details
type TypeDetails interface {
	String() string      // For debugging
	GetBaseType() string // Returns the base type (for basic/enum) or empty
}

// Basic type (string, int, etc) or type alias (type UUID string)
type BasicDetails struct {
	BaseType string // "string", "int", "bool", etc.
}

func (b BasicDetails) GetBaseType() string { return b.BaseType }

// Enum type
type EnumDetails struct {
	BaseType   string // "string", "int", etc.
	EnumValues []EnumValue
}

func (e EnumDetails) GetBaseType() string { return e.BaseType }

// Struct type
type StructDetails struct {
	Fields []FieldInfo
}

func (s StructDetails) GetBaseType() string { return "" }

// Slice type ([]T or type MySlice []T)
type SliceDetails struct {
	ElementType *FieldTypeInfo
}

func (s SliceDetails) GetBaseType() string { return "" }

// Array type ([N]T)
type ArrayDetails struct {
	ElementType *FieldTypeInfo
	Length      string // "5", "10", etc.
}

func (a ArrayDetails) GetBaseType() string { return "" }

// Map type (map[K]V)
type MapDetails struct {
	KeyType   *FieldTypeInfo
	ValueType *FieldTypeInfo
}

func (m MapDetails) GetBaseType() string { return "" }

// Pointer type (*T or type MyPtr *T)
type PointerDetails struct {
	PointedType *FieldTypeInfo
}

func (p PointerDetails) GetBaseType() string {
	return ""
}
