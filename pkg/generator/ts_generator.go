package generator

import (
	"fmt"
	"os"
	"sort"
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
	for _, t := range typesSlice {
		g.tsTypes[t.Name] = g.generateType(t)
	}

	for _, tsType := range g.tsTypes {
		fmt.Println(tsType)
	}

	file, err := os.Create("out.ts")
	if err != nil {
		fmt.Println(err)
		return
	}
	defer file.Close()
	for _, tsType := range g.tsTypes {
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
	switch t.Kind {
	case BasicKind:
		return g.generateBasicType(t)
	case SliceKind:
		return g.generateSliceType(t)
	case ArrayKind:
		return g.generateArrayType(t)
	case MapKind:
		return g.generateMapType(t)
	case StructKind:
		return g.generateStructType(t)
	case EnumKind:
		return g.generateEnumType(t)
	case PointerKind:
		return g.generatePointerType(t)
	}
	return ""
}

func (g *TSGenerator) generateBasicType(t *TypeInfo) string {
	details := t.Underlying.(BasicDetails)
	var sb strings.Builder
	if !t.Comment.IsEmpty() {
		sb.WriteString("// ")
		sb.WriteString(t.Comment.String())
		sb.WriteString("\n")
	}
	sb.WriteString("export type ")
	sb.WriteString(t.Name)
	sb.WriteString(" = ")
	tstyp := g.goTypeToTSType(details.BaseType)
	sb.WriteString(tstyp.Type)
	if tstyp.Annotation != "" {
		sb.WriteString(" // ")
		sb.WriteString(tstyp.Annotation)
	}
	sb.WriteString(";\n")
	return sb.String()
}

func (g *TSGenerator) generateSliceType(t *TypeInfo) string {
	details := t.Underlying.(SliceDetails)
	var sb strings.Builder
	if !t.Comment.IsEmpty() {
		sb.WriteString("// ")
		sb.WriteString(t.Comment.String())
		sb.WriteString("\n")
	}
	sb.WriteString("export type ")
	sb.WriteString(t.Name)
	sb.WriteString(" = Array<")
	sb.WriteString(g.generateFieldType(details.ElementType))
	sb.WriteString(">;\n")
	return sb.String()
}

func (g *TSGenerator) generateArrayType(t *TypeInfo) string {
	details := t.Underlying.(ArrayDetails)
	var sb strings.Builder
	if !t.Comment.IsEmpty() {
		sb.WriteString("// ")
		sb.WriteString(t.Comment.String())
		sb.WriteString("\n")
	}
	sb.WriteString("export type ")
	sb.WriteString(t.Name)
	sb.WriteString(" = Array<")
	sb.WriteString(g.generateFieldType(details.ElementType))
	sb.WriteString(">; // Fixed array[")
	sb.WriteString(details.Length)
	sb.WriteString("]\n")
	return sb.String()
}

func (g *TSGenerator) generateMapType(t *TypeInfo) string {
	details := t.Underlying.(MapDetails)
	var sb strings.Builder
	if !t.Comment.IsEmpty() {
		sb.WriteString("// ")
		sb.WriteString(t.Comment.String())
		sb.WriteString("\n")
	}
	sb.WriteString("export type ")
	sb.WriteString(t.Name)
	sb.WriteString(" = Record<")
	sb.WriteString(g.generateFieldType(details.KeyType))
	sb.WriteString(", ")
	sb.WriteString(g.generateFieldType(details.ValueType))
	sb.WriteString(">;\n")
	return sb.String()
}

func (g *TSGenerator) generatePointerType(t *TypeInfo) string {
	details := t.Underlying.(PointerDetails)
	var sb strings.Builder
	if !t.Comment.IsEmpty() {
		sb.WriteString("// ")
		sb.WriteString(t.Comment.String())
		sb.WriteString("\n")
	}
	sb.WriteString("export type ")
	sb.WriteString(t.Name)
	sb.WriteString(" = ")
	sb.WriteString(g.generateFieldType(details.PointedType))
	sb.WriteString(" | null; // Pointer type\n")
	return sb.String()
}

func (g *TSGenerator) generateStructType(t *TypeInfo) string {
	details := t.Underlying.(StructDetails)
	var sb strings.Builder
	if !t.Comment.IsEmpty() {
		sb.WriteString("// ")
		sb.WriteString(t.Comment.String())
		sb.WriteString("\n")
	}
	sb.WriteString("export type ")
	sb.WriteString(t.Name)
	sb.WriteString(" = {\n")

	for _, field := range details.Fields {
		if !field.Comment.IsEmpty() {
			sb.WriteString("  // ")
			sb.WriteString(field.Comment.String())
			sb.WriteString("\n")
		}
		sb.WriteString("  ")
		if field.JSONName != "" {
			sb.WriteString(field.JSONName)
		} else if field.Type.IsEmbedded {
			// For embedded fields, spread the type
			sb.WriteString("...")
			sb.WriteString(field.Name)
			continue
		} else {
			sb.WriteString(field.Name)
		}
		sb.WriteString(": ")
		sb.WriteString(g.generateFieldType(field.Type))
		sb.WriteString(";\n")
	}

	sb.WriteString("};\n")
	return sb.String()
}

func (g *TSGenerator) generateEnumType(t *TypeInfo) string {
	details := t.Underlying.(EnumDetails)
	var sb strings.Builder
	if !t.Comment.IsEmpty() {
		sb.WriteString("// ")
		sb.WriteString(t.Comment.String())
		sb.WriteString("\n")
	}
	sb.WriteString("export type ")
	sb.WriteString(t.Name)
	sb.WriteString(" = \n")

	for i, ev := range details.EnumValues {
		if !ev.Comment.IsEmpty() {
			sb.WriteString("  // ")
			sb.WriteString(ev.Comment.String())
			sb.WriteString("\n")
		}
		sb.WriteString("  | ")
		sb.WriteString(ev.Value)
		if i != len(details.EnumValues)-1 {
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
		for _, ev := range details.EnumValues {
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

// New helper method to generate field types
func (g *TSGenerator) generateFieldType(ft *FieldTypeInfo) string {
	if ft.IsPointer {
		// Pointers become nullable
		return g.generateFieldType(&FieldTypeInfo{
			BaseType:  ft.BaseType,
			IsSlice:   ft.IsSlice,
			IsArray:   ft.IsArray,
			IsMap:     ft.IsMap,
			KeyType:   ft.KeyType,
			ValueType: ft.ValueType,
		}) + " | null"
	}

	if ft.IsSlice {
		return "Array<" + g.generateFieldType(ft.ValueType) + ">"
	}

	if ft.IsArray {
		return "Array<" + g.generateFieldType(ft.ValueType) + ">"
	}

	if ft.IsMap {
		return "Record<" + g.generateFieldType(ft.KeyType) + ", " + g.generateFieldType(ft.ValueType) + ">"
	}

	// Basic type
	tstyp := g.goTypeToTSType(ft.BaseType)
	return tstyp.Type
}
