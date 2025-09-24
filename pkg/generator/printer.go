package generator

import (
	"errors"
	"fmt"
	"go/ast"
	"path"
	"sort"
	"strconv"
	"strings"

	"golang.org/x/tools/go/packages"
)

func (t TypeKind) String() string {
	return string(t)
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
func (p Position) String() string {
	return fmt.Sprintf("%s - %s:%d", p.Package, p.Filename, p.Line)
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
	sb.WriteString(ti.Underlying.String()) // Use the String() method from the interface
	sb.WriteString("\n")

	if !ti.Comment.IsEmpty() {
		sb.WriteString("  Comment: ")
		sb.WriteString(ti.Comment.String())
		sb.WriteString("\n")
	}
	sb.WriteString("  Position: ")
	sb.WriteString(ti.Position.String())
	sb.WriteString("\n")

	// Handle type-specific details
	switch details := ti.Underlying.(type) {
	case StructDetails:
		if len(details.Fields) > 0 {
			sb.WriteString("  Fields:")
			for _, field := range details.Fields {
				sb.WriteString("\n    - ")
				sb.WriteString(field.String())
			}
		}
	case EnumDetails:
		if len(details.EnumValues) > 0 {
			sb.WriteString("  Values:")
			for _, ev := range details.EnumValues {
				sb.WriteString("\n    - ")
				sb.WriteString(ev.String())
			}
		}
	case SliceDetails:
		sb.WriteString("  Element: ")
		sb.WriteString(details.ElementType.String())
	case ArrayDetails:
		sb.WriteString("  Element: ")
		sb.WriteString(details.ElementType.String())
		sb.WriteString(" (length: ")
		sb.WriteString(details.Length)
		sb.WriteString(")")
	case MapDetails:
		sb.WriteString("  Key: ")
		sb.WriteString(details.KeyType.String())
		sb.WriteString("\n  Value: ")
		sb.WriteString(details.ValueType.String())
	case PointerDetails:
		sb.WriteString("  Points to: ")
		sb.WriteString(details.PointedType.String())
	}

	return sb.String()
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

func (b BasicDetails) String() string {
	return b.BaseType
}

func (e EnumDetails) String() string {
	return fmt.Sprintf("enum(%s)", e.BaseType)
}

func (s StructDetails) String() string {
	return fmt.Sprintf("struct with %d fields", len(s.Fields))
}

func (s SliceDetails) String() string {
	return "[]" + s.ElementType.String()
}

func (a ArrayDetails) String() string {
	return fmt.Sprintf("[%s]%s", a.Length, a.ElementType.String())
}

func (m MapDetails) String() string {
	return fmt.Sprintf("map[%s]%s", m.KeyType.String(), m.ValueType.String())
}

func (p PointerDetails) String() string {
	return "*" + p.PointedType.String()
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

func (g *GoParser) printTypes() {
	types := make([]*TypeInfo, 0, len(g.types))
	for _, t := range g.types {
		types = append(types, t)
	}
	sort.Slice(types, func(i, j int) bool {
		return types[i].Name < types[j].Name
	})

	for _, t := range types {
		fmt.Printf("\n- %v\n", t)
	}
}
