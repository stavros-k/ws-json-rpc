package generator

import (
	"errors"
	"fmt"
	"go/ast"
	"path"
	"strconv"
	"strings"

	"golang.org/x/tools/go/packages"
)

type TypeKind string

func (t TypeKind) String() string {
	return string(t)
}

const (
	StructType TypeKind = "struct"
	EnumType   TypeKind = "enum"
	BasicType  TypeKind = "basic"
	SliceType  TypeKind = "slice"
	MapType    TypeKind = "map"
)

type Comment struct {
	// Comment above the declaration
	Above  string
	Inline string // Comment on the same line
}

func (c Comment) IsEmpty() bool {
	return c.Above == "" && c.Inline == ""
}

func (c Comment) String() string {
	var sb strings.Builder
	sb.WriteString(c.Above)
	if c.Inline != "" {
		sb.WriteString(" | ")
		sb.WriteString(c.Inline)
	}
	return sb.String()
}

type EnumValue struct {
	Name    string
	Value   string // The actual value, (ie "1", "foo", etc)
	Comment Comment
}

func (ev EnumValue) String() string {
	var sb strings.Builder
	sb.WriteString(ev.Name)
	sb.WriteString(" = ")
	sb.WriteString(ev.Value)
	if ev.Comment.IsEmpty() {
		return sb.String()
	}

	if !ev.Comment.IsEmpty() {
		sb.WriteString(" // ")
		sb.WriteString(ev.Comment.String())
		return sb.String()
	}

	return sb.String()
}

type Position struct {
	Package  string
	Filename string
	Line     int
}

func (p Position) String() string {
	return fmt.Sprintf("%s - %s:%d", p.Package, p.Filename, p.Line)
}

type TypeInfo struct {
	Name       string // Name of the type (ie TypeInfo)
	Kind       TypeKind
	Underlying string // ie for enums Kind is EnumType, Underlying is int, string, etc
	Comment    Comment
	Position   Position
	Fields     []FieldInfo // For struct types
	EnumValues []EnumValue // For enum types
}

func (ti TypeInfo) String() string {
	var sb strings.Builder

	sb.WriteString("Name: ")
	sb.WriteString(ti.Name)
	sb.WriteString("\n")
	sb.WriteString("  Kind: ")
	sb.WriteString(ti.Kind.String())
	sb.WriteString("\n")
	sb.WriteString("  Underlying: ")
	sb.WriteString(ti.Underlying)
	sb.WriteString("\n")

	if !ti.Comment.IsEmpty() {
		sb.WriteString("  Comment: ")
		sb.WriteString(ti.Comment.String())
		sb.WriteString("\n")
	}
	sb.WriteString("  Position: ")
	sb.WriteString(ti.Position.String())
	sb.WriteString("\n")

	if len(ti.Fields) > 0 {
		sb.WriteString("  Fields:")
		for _, field := range ti.Fields {
			sb.WriteString("\n    - ")
			sb.WriteString(field.String())
		}
	}

	if len(ti.EnumValues) > 0 {
		sb.WriteString("  Values:")
		for _, ev := range ti.EnumValues {
			sb.WriteString("\n    - ")
			sb.WriteString(ev.String())
		}
	}

	return sb.String()
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

func (fti FieldTypeInfo) String() string {
	var sb strings.Builder

	if fti.IsPointer {
		sb.WriteString("*")
	}

	if fti.IsSlice {
		sb.WriteString("[]")
		sb.WriteString(fti.ValueType.String())
	} else if fti.IsArray {
		sb.WriteString("[")
		sb.WriteString(fti.ArrayLength)
		sb.WriteString("]")
		sb.WriteString(fti.ValueType.String())
	} else if fti.IsMap {
		sb.WriteString("map[")
		sb.WriteString(fti.KeyType.String())
		sb.WriteString("]")
		sb.WriteString(fti.ValueType.String())
	} else {
		sb.WriteString(fti.BaseType)
	}

	return sb.String()
}

type FieldInfo struct {
	Name        string
	Type        *FieldTypeInfo
	JSONName    string
	JSONOptions []string
	Comment     Comment
}

func (fi FieldInfo) String() string {
	var sb strings.Builder
	if fi.Name != "" {
		sb.WriteString(fi.Name)
		sb.WriteString(" ")
	} else {
		sb.WriteString("'embedded' ")
	}
	sb.WriteString("(")
	sb.WriteString(fi.Type.String())
	sb.WriteString(")")

	if fi.JSONName != "" {
		sb.WriteString(" `")
		sb.WriteString(fi.JSONName)
		sb.WriteString("`")
		for _, opt := range fi.JSONOptions {
			sb.WriteString(", ")
			sb.WriteString(opt)
		}
	}

	if !fi.Comment.IsEmpty() {
		sb.WriteString(" // ")
		sb.WriteString(fi.Comment.String())
	}

	return sb.String()
}

func (g *GoParser) fmtError(pkg *packages.Package, decl *ast.GenDecl, err error) error {
	var sb strings.Builder

	// Package
	sb.WriteString("Error: \n")
	sb.WriteString("  Package: ")
	sb.WriteString(pkg.PkgPath)
	sb.WriteString("\n")

	sb.WriteString("  Position: ")
	sb.WriteString(path.Base(g.fset.File(decl.Pos()).Name()))
	sb.WriteString(":")
	sb.WriteString(strconv.Itoa(pkg.Fset.Position(decl.Pos()).Line))
	sb.WriteString("\n")

	// Declaration
	sb.WriteString("  Declaration: ")
	sb.WriteString(decl.Tok.String())
	if len(decl.Specs) > 0 {
		switch s := decl.Specs[0].(type) {
		case *ast.TypeSpec:
			if s.Name.Name != "" {
				sb.WriteString(" (type: ")
				sb.WriteString(s.Name.Name)
				sb.WriteString(")")
			}
		case *ast.ValueSpec:
			if len(s.Names) > 0 && s.Names[0].Name != "" {
				sb.WriteString(" (const: ")
				sb.WriteString(s.Names[0].Name)
				sb.WriteString(")")
			}
		}
	}
	sb.WriteString("\n")

	// Error
	sb.WriteString("  Message: ")
	sb.WriteString(err.Error())
	sb.WriteString("\n")

	return errors.New(sb.String())
}
