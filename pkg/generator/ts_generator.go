package generator

import (
	"fmt"
	"os"
	"slices"
	"sort"
	"strconv"
	"strings"
)

type TSGeneratorOptions struct {
	GenerateEnumValues bool
}

type TSGenerator struct {
	renderedTypes map[string]string
	parsedTypes   map[string]*TypeInfo
	Options       TSGeneratorOptions
}

func NewTSGenerator(options *TSGeneratorOptions, types map[string]*TypeInfo) *TSGenerator {
	if options == nil {
		options = &TSGeneratorOptions{}
	}

	return &TSGenerator{
		parsedTypes:   types,
		renderedTypes: make(map[string]string),
		Options:       *options,
	}
}

func (g *TSGenerator) renderTypescriptTypes() {
	for typeName, tsType := range g.parsedTypes {
		g.renderedTypes[typeName] = g.generateType(tsType)
	}
}

func (g *TSGenerator) GetRenderedTypes() map[string]string {
	if len(g.renderedTypes) == 0 {
		g.renderTypescriptTypes()
	}
	return g.renderedTypes
}

func (g *TSGenerator) Generate() {
	if len(g.renderedTypes) == 0 {
		g.renderTypescriptTypes()
	}

	sortedTypesNames := make([]string, 0, len(g.renderedTypes))
	for typeName := range g.renderedTypes {
		sortedTypesNames = append(sortedTypesNames, typeName)
	}
	sort.Strings(sortedTypesNames)

	file, err := os.Create("out.ts")
	if err != nil {
		fmt.Println(err)
		return
	}
	defer file.Close()
	for _, typeName := range sortedTypesNames {
		_, err = file.WriteString(g.renderedTypes[typeName] + "\n")
		if err != nil {
			fmt.Println(err)
			return
		}
	}
	fmt.Println("TypeScript definitions written to out.ts")
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
	switch typ := t.Type.(type) {
	case BasicType:
		return g.generateBasicType(t, typ)
	case SliceType:
		return g.generateSliceType(t, typ)
	case ArrayType:
		return g.generateArrayType(t, typ)
	case MapType:
		return g.generateMapType(t, typ)
	case StructType:
		return g.generateStructType(t, typ)
	case EnumType:
		return g.generateEnumType(t, typ)
	case PointerType:
		return g.generatePointerType(t, typ)
	}
	return ""
}

func (g *TSGenerator) generateBasicType(t *TypeInfo, typ BasicType) string {
	var sb strings.Builder
	generatePosition(&sb, t.Position)
	generateComment(&sb, t.Comment, 0)
	sb.WriteString("export type ")
	sb.WriteString(t.Name)
	sb.WriteString(" = ")
	tstyp := g.goTypeToTSType(typ.Name)
	sb.WriteString(tstyp.Type)
	if tstyp.Annotation != "" {
		sb.WriteString("; // ")
		sb.WriteString(tstyp.Annotation)
	}
	sb.WriteString(";\n")
	return sb.String()
}

func (g *TSGenerator) generateSliceType(t *TypeInfo, typ SliceType) string {
	var sb strings.Builder
	generatePosition(&sb, t.Position)
	generateComment(&sb, t.Comment, 0)
	sb.WriteString("export type ")
	sb.WriteString(t.Name)
	sb.WriteString(" = Array<")
	sb.WriteString(g.generateTypeExpression(typ.Element))
	sb.WriteString(">;\n")
	return sb.String()
}

func (g *TSGenerator) generateArrayType(t *TypeInfo, typ ArrayType) string {
	var sb strings.Builder
	generatePosition(&sb, t.Position)
	generateComment(&sb, t.Comment, 0)
	sb.WriteString("export type ")
	sb.WriteString(t.Name)
	sb.WriteString(" = Array<")
	sb.WriteString(g.generateTypeExpression(typ.Element))
	sb.WriteString(">; // Fixed array[")
	sb.WriteString(strconv.Itoa(typ.Length))
	sb.WriteString("]\n")
	return sb.String()
}

func (g *TSGenerator) generateMapType(t *TypeInfo, typ MapType) string {
	var sb strings.Builder
	generatePosition(&sb, t.Position)
	generateComment(&sb, t.Comment, 0)
	sb.WriteString("export type ")
	sb.WriteString(t.Name)
	sb.WriteString(" = Record<")
	sb.WriteString(g.generateTypeExpression(typ.Key))
	sb.WriteString(", ")
	sb.WriteString(g.generateTypeExpression(typ.Value))
	sb.WriteString(">;\n")
	return sb.String()
}

func (g *TSGenerator) generatePointerType(t *TypeInfo, typ PointerType) string {
	var sb strings.Builder
	generatePosition(&sb, t.Position)
	generateComment(&sb, t.Comment, 0)
	sb.WriteString("export type ")
	sb.WriteString(t.Name)
	sb.WriteString(" = ")
	sb.WriteString(g.generateTypeExpression(typ.Element))
	sb.WriteString(" | null; // Pointer type\n")
	return sb.String()
}

func (g *TSGenerator) generateStructType(t *TypeInfo, typ StructType) string {
	var sb strings.Builder
	generatePosition(&sb, t.Position)
	generateComment(&sb, t.Comment, 0)
	sb.WriteString("export type ")
	sb.WriteString(t.Name)
	sb.WriteString(" = {\n")

	for _, field := range typ.Fields {
		// FIXME: Handle embedded fields
		if _, ok := field.Type.(EmbeddedType); ok {
			// Spread the embedded type
			sb.WriteString("  ...")
			sb.WriteString(field.Name)
			sb.WriteString(";\n")
			continue
		}

		generateComment(&sb, field.Comment, 2)

		sb.WriteString("  ")
		if field.JSONName != "" {
			sb.WriteString(field.JSONName)
		} else {
			sb.WriteString(field.Name)
		}

		// Handle optional fields
		isOptional := false
		if slices.Contains(field.JSONOptions, "omitempty") {
			isOptional = true
		}
		if isOptional {
			sb.WriteString("?")
		}

		sb.WriteString(": ")
		sb.WriteString(g.generateTypeExpression(field.Type))
		sb.WriteString(";\n")
	}

	sb.WriteString("};\n")
	return sb.String()
}

func (g *TSGenerator) generateEnumType(t *TypeInfo, typ EnumType) string {
	var sb strings.Builder
	generatePosition(&sb, t.Position)
	generateComment(&sb, t.Comment, 0)
	sb.WriteString("export type ")
	sb.WriteString(t.Name)
	sb.WriteString(" = \n")

	for i, ev := range typ.EnumValues {
		generateComment(&sb, ev.Comment, 2)
		sb.WriteString("  | ")
		sb.WriteString(ev.Value)
		if i != len(typ.EnumValues)-1 {
			sb.WriteString("\n")
		}
	}
	sb.WriteString(";\n")

	if g.Options.GenerateEnumValues {
		sb.WriteString("\n")
		generatePosition(&sb, t.Position)
		generateComment(&sb, t.Comment, 0)
		sb.WriteString("export const ")
		sb.WriteString(t.Name)
		sb.WriteString("Values = {\n")
		for _, ev := range typ.EnumValues {
			generateComment(&sb, ev.Comment, 2)
			sb.WriteString("  ")
			sb.WriteString(ev.Name)
			sb.WriteString(": ")
			sb.WriteString(ev.Value)
			sb.WriteString(",\n")
		}
		sb.WriteString("} as const;\n")
	}

	return sb.String()
}

// Replace generateFieldType with generateTypeExpression
func (g *TSGenerator) generateTypeExpression(t TypeExpression) string {
	switch typ := t.(type) {
	case BasicType:
		tstyp := g.goTypeToTSType(typ.Name)
		return tstyp.Type
	case SliceType:
		return "Array<" + g.generateTypeExpression(typ.Element) + ">"
	case ArrayType:
		return "Array<" + g.generateTypeExpression(typ.Element) + ">"
	case MapType:
		return "Record<" + g.generateTypeExpression(typ.Key) + ", " + g.generateTypeExpression(typ.Value) + ">"
	case PointerType:
		return g.generateTypeExpression(typ.Element) + " | null"
	default:
		// For struct/enum references, just use the type name
		return "unknown" // or handle other cases
	}
}

func generatePosition(sb *strings.Builder, pos Position) {
	sb.WriteString(fmt.Sprintf("// From: %s.%s:%d\n", pos.Package, pos.Filename, pos.Line))
}

func generateComment(sb *strings.Builder, comment Comment, ident int) {
	if comment.IsEmpty() {
		return
	}

	lines := strings.Lines(comment.String())
	for line := range lines {
		sb.WriteString(strings.Repeat(" ", ident))
		sb.WriteString("// ")
		sb.WriteString(line)
	}
	sb.WriteString("\n")
}
