package generator

import (
	"fmt"
	"os"
	"sort"
	"strconv"
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
	typesSlice := make([]*TypeInfo, 0, len(types))
	for _, t := range types {
		typesSlice = append(typesSlice, t)
	}
	sort.Slice(typesSlice, func(i, j int) bool {
		return typesSlice[i].Name < typesSlice[j].Name
	})

	renderedTypes := make([]string, 0, len(g.tsTypes))
	for _, t := range typesSlice {
		g.tsTypes[t.Name] = g.generateType(t)
		renderedTypes = append(renderedTypes, g.tsTypes[t.Name])
	}

	fmt.Println(strings.Join(renderedTypes, "\n"))

	file, err := os.Create("out.ts")
	if err != nil {
		fmt.Println(err)
		return
	}
	defer file.Close()
	for _, tsType := range renderedTypes {
		_, err = file.WriteString(tsType)
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
	if !t.Comment.IsEmpty() {
		sb.WriteString("// ")
		sb.WriteString(t.Comment.String())
		sb.WriteString("\n")
	}
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
	if !t.Comment.IsEmpty() {
		sb.WriteString("// ")
		sb.WriteString(t.Comment.String())
		sb.WriteString("\n")
	}
	sb.WriteString("export type ")
	sb.WriteString(t.Name)
	sb.WriteString(" = Array<")
	sb.WriteString(g.generateTypeExpression(typ.Element))
	sb.WriteString(">;\n")
	return sb.String()
}

func (g *TSGenerator) generateArrayType(t *TypeInfo, typ ArrayType) string {
	var sb strings.Builder
	if !t.Comment.IsEmpty() {
		sb.WriteString("// ")
		sb.WriteString(t.Comment.String())
		sb.WriteString("\n")
	}
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
	if !t.Comment.IsEmpty() {
		sb.WriteString("// ")
		sb.WriteString(t.Comment.String())
		sb.WriteString("\n")
	}
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
	if !t.Comment.IsEmpty() {
		sb.WriteString("// ")
		sb.WriteString(t.Comment.String())
		sb.WriteString("\n")
	}
	sb.WriteString("export type ")
	sb.WriteString(t.Name)
	sb.WriteString(" = ")
	sb.WriteString(g.generateTypeExpression(typ.Element))
	sb.WriteString(" | null; // Pointer type\n")
	return sb.String()
}

func (g *TSGenerator) generateStructType(t *TypeInfo, typ StructType) string {
	var sb strings.Builder
	if !t.Comment.IsEmpty() {
		sb.WriteString("// ")
		sb.WriteString(t.Comment.String())
		sb.WriteString("\n")
	}
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

		if !field.Comment.IsEmpty() {
			sb.WriteString("  // ")
			sb.WriteString(field.Comment.String())
			sb.WriteString("\n")
		}
		sb.WriteString("  ")
		if field.JSONName != "" {
			sb.WriteString(field.JSONName)
		} else {
			sb.WriteString(field.Name)
		}

		// Handle optional fields
		isOptional := false
		for _, opt := range field.JSONOptions {
			if opt == "omitempty" {
				isOptional = true
				break
			}
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
	if !t.Comment.IsEmpty() {
		sb.WriteString("// ")
		sb.WriteString(t.Comment.String())
		sb.WriteString("\n")
	}
	sb.WriteString("export type ")
	sb.WriteString(t.Name)
	sb.WriteString(" = \n")

	for i, ev := range typ.EnumValues {
		if !ev.Comment.IsEmpty() {
			sb.WriteString("  // ")
			sb.WriteString(ev.Comment.String())
			sb.WriteString("\n")
		}
		sb.WriteString("  | ")
		sb.WriteString(ev.Value)
		if i != len(typ.EnumValues)-1 {
			sb.WriteString("\n")
		}
	}
	sb.WriteString(";\n")

	if g.Options.GenerateEnumValues {
		sb.WriteString("\n")
		if !t.Comment.IsEmpty() {
			sb.WriteString("// ")
			sb.WriteString(t.Comment.String())
			sb.WriteString("\n")
		}
		sb.WriteString("export const ")
		sb.WriteString(t.Name)
		sb.WriteString("Values = {\n")
		for _, ev := range typ.EnumValues {
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
