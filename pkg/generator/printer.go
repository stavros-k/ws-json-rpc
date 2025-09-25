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
	sb.WriteString("  Type: ")
	sb.WriteString(ti.Type.Kind().String())
	sb.WriteString("\n")
	sb.WriteString("  Underlying: ")
	sb.WriteString(ti.Type.String())
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
	switch t := ti.Type.(type) {
	case StructType:
		if len(t.Fields) > 0 {
			sb.WriteString("  Fields:")
			for _, field := range t.Fields {
				sb.WriteString("\n    - ")
				sb.WriteString(field.String())
			}
		}
	case EnumType:
		if len(t.EnumValues) > 0 {
			sb.WriteString("  Values:")
			for _, ev := range t.EnumValues {
				sb.WriteString("\n    - ")
				sb.WriteString(ev.String())
			}
		}
	case SliceType:
		sb.WriteString("  Element: ")
		sb.WriteString(t.Element.String())
	case ArrayType:
		sb.WriteString("  Element: ")
		sb.WriteString(t.Element.String())
		sb.WriteString(" (length: ")
		sb.WriteString(strconv.Itoa(t.Length))
		sb.WriteString(")")
	case MapType:
		sb.WriteString("  Key: ")
		sb.WriteString(t.Key.String())
		sb.WriteString("\n  Value: ")
		sb.WriteString(t.Value.String())
	case PointerType:
		sb.WriteString("  Points to: ")
		sb.WriteString(t.Element.String())
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

// String methods for the unified types
func (b BasicType) String() string {
	return b.Name
}

func (e EnumType) String() string {
	return fmt.Sprintf("enum(%s)", e.BaseType)
}

func (s StructType) String() string {
	return fmt.Sprintf("struct with %d fields", len(s.Fields))
}

func (s SliceType) String() string {
	return "[]" + s.Element.String()
}

func (a ArrayType) String() string {
	return fmt.Sprintf("[%d]%s", a.Length, a.Element.String())
}

func (m MapType) String() string {
	return fmt.Sprintf("map[%s]%s", m.Key.String(), m.Value.String())
}

func (p PointerType) String() string {
	return "*" + p.Element.String()
}

func (e EmbeddedType) String() string {
	return "embedded:" + e.Type.String()
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
	for _, spec := range decl.Specs {
		switch s := spec.(type) {
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
		sb.WriteString("\n")
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
