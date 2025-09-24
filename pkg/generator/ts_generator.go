package generator

import (
	"fmt"
	"strings"
)

type TSGeneratorOptions struct {
	GenerateEnumValues bool
}

type TSGenerator struct {
	tsTypes map[string]string
	Options TSGeneratorOptions
}

func NewTSGenerator(options *TSGeneratorOptions) *TSGenerator {
	if options == nil {
		options = &TSGeneratorOptions{}
	}
	return &TSGenerator{
		tsTypes: make(map[string]string),
		Options: *options,
	}
}

func (g *TSGenerator) Generate(types map[string]*TypeInfo) {
	for _, t := range types {
		g.tsTypes[t.Name] = g.generateType(t)
	}

	for _, tsType := range g.tsTypes {
		fmt.Println(tsType)
	}
}

type TSTypeInfo struct {
	Type       string
	Annotation string
}

func (g *TSGenerator) goTypeToTSType(t string) TSTypeInfo {
	// FIXME:
	switch t {
	case "string":
		return TSTypeInfo{Type: "string"}
	case "int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64",
		"float32", "float64":
		return TSTypeInfo{Type: "number", Annotation: t}
	case "bool":
		return TSTypeInfo{Type: "boolean"}
	case "any":
		return TSTypeInfo{Type: "any"}
	default:
		return TSTypeInfo{Type: t}
	}
}

func (g *TSGenerator) generateType(t *TypeInfo) string {
	switch t.Kind {
	case BasicType:
		return g.generateBasicType(t)
	case SliceType:
		return g.generateSliceType(t)
	case MapType:
		return g.generateMapType(t)
	case StructType:
		return g.generateStructType(t)
	case EnumType:
		return g.generateEnumType(t)
	}
	return ""
}

func (g *TSGenerator) generateBasicType(t *TypeInfo) string {
	var sb strings.Builder
	if !t.Comment.IsEmpty() {
		sb.WriteString("// ")
		sb.WriteString(t.Comment.String())
		sb.WriteString("\n")
	}
	sb.WriteString("export type ")
	sb.WriteString(t.Name)
	sb.WriteString(" = ")
	tstyp := g.goTypeToTSType(t.Underlying)
	sb.WriteString(tstyp.Type)
	if tstyp.Annotation != "" {
		sb.WriteString(" // ")
		sb.WriteString(tstyp.Annotation)
	}
	sb.WriteString(";\n")
	return sb.String()
}

func (g *TSGenerator) generateSliceType(t *TypeInfo) string {
	var sb strings.Builder
	if !t.Comment.IsEmpty() {
		sb.WriteString("// ")
		sb.WriteString(t.Comment.String())
		sb.WriteString("\n")
	}
	sb.WriteString("export type ")
	sb.WriteString(t.Name)
	sb.WriteString(" = Array<")
	// FIXME:
	sb.WriteString(">;\n")
	return sb.String()
}

func (g *TSGenerator) generateMapType(t *TypeInfo) string {
	var sb strings.Builder
	if !t.Comment.IsEmpty() {
		sb.WriteString("// ")
		sb.WriteString(t.Comment.String())
		sb.WriteString("\n")
	}
	sb.WriteString("export type ")
	sb.WriteString(t.Name)
	sb.WriteString(" = Record<")
	// FIXME:
	sb.WriteString(", ")
	// FIXME:
	sb.WriteString(">;\n")
	return sb.String()
}

func (g *TSGenerator) generateStructType(t *TypeInfo) string {
	var sb strings.Builder
	if !t.Comment.IsEmpty() {
		sb.WriteString("// ")
		sb.WriteString(t.Comment.String())
		sb.WriteString("\n")
	}
	sb.WriteString("export type ")
	sb.WriteString(t.Name)
	sb.WriteString(" = {\n")

	for _, field := range t.Fields {
		if !field.Comment.IsEmpty() {
			sb.WriteString("  // ")
			sb.WriteString(field.Comment.String())
			sb.WriteString("\n")
		}
		sb.WriteString("  ")
		sb.WriteString(field.Name)
		sb.WriteString(": ")
		tstyp := g.goTypeToTSType(field.Type.BaseType)
		sb.WriteString(tstyp.Type)
		if tstyp.Annotation != "" {
			sb.WriteString(" // ")
			sb.WriteString(tstyp.Annotation)
		}
		sb.WriteString(";\n")
	}

	sb.WriteString("};\n")

	return sb.String()
}

func (g *TSGenerator) generateEnumType(t *TypeInfo) string {
	var sb strings.Builder
	if !t.Comment.IsEmpty() {
		sb.WriteString("// ")
		sb.WriteString(t.Comment.String())
		sb.WriteString("\n")
	}
	sb.WriteString("export type ")
	sb.WriteString(t.Name)
	sb.WriteString(" = \n")

	for i, ev := range t.EnumValues {
		if !ev.Comment.IsEmpty() {
			sb.WriteString("  // ")
			sb.WriteString(ev.Comment.String())
			sb.WriteString("\n")
		}
		sb.WriteString("  | ")
		sb.WriteString(ev.Name)
		if i != len(t.EnumValues)-1 {
			sb.WriteString("\n")
		}
	}
	sb.WriteString(";\n")

	if g.Options.GenerateEnumValues {
		sb.WriteString("\n")
		// Generate enum values as constants
		if !t.Comment.IsEmpty() {
			sb.WriteString("// ")
			sb.WriteString(t.Comment.String())
			sb.WriteString("\n")
		}
		sb.WriteString("export const ")
		sb.WriteString(t.Name)
		sb.WriteString("Values = {\n")
		for _, ev := range t.EnumValues {
			if !ev.Comment.IsEmpty() {
				sb.WriteString("  // ")
				sb.WriteString(ev.Comment.String())
				sb.WriteString("\n")
			}
			sb.WriteString("  ")
			sb.WriteString(ev.Name)
			sb.WriteString(": ")
			sb.WriteString(ev.Value)
			sb.WriteString(",\n")
		}
		sb.WriteString("} as const;\n\n")
	}

	return sb.String()
}
