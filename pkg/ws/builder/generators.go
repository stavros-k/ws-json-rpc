package builder

import (
	"fmt"
	"strings"
)

type GoGenerator struct {
	packageName string
}

type TypeScriptGenerator struct{}

func NewGoGenerator(packageName string) *GoGenerator {
	return &GoGenerator{packageName: packageName}
}

func NewTypeScriptGenerator() *TypeScriptGenerator {
	return &TypeScriptGenerator{}
}

func (g *GoGenerator) GenerateType(typeDef TypeDefinition) (string, error) {
	var builder strings.Builder

	if typeDef.Description != "" {
		builder.WriteString(fmt.Sprintf("// %s\n", typeDef.Description))
	}

	// Handle enum types differently
	if typeDef.Type == EnumType {
		builder.WriteString(fmt.Sprintf("type %s string\n\n", typeDef.Name))
		builder.WriteString("const (\n")
		for _, value := range typeDef.EnumValues {
			constName := fmt.Sprintf("%s%s", typeDef.Name, strings.Title(value))
			builder.WriteString(fmt.Sprintf("\t%s %s = \"%s\"\n", constName, typeDef.Name, value))
		}
		builder.WriteString(")\n\n")
		return builder.String(), nil
	}

	// Handle struct types
	builder.WriteString(fmt.Sprintf("type %s struct {\n", typeDef.Name))

	for _, field := range typeDef.Fields {
		builder.WriteString(g.generateGoField(field))
	}

	builder.WriteString("}\n\n")

	if g.hasJSONDateFields(typeDef) {
		builder.WriteString(g.generateJSONDateMarshallers(typeDef))
	}

	return builder.String(), nil
}

func (g *GoGenerator) generateGoField(field FieldDefinition) string {
	var fieldType string
	var tags []string

	switch field.Type {
	case StringType:
		fieldType = "string"
	case NumberType:
		fieldType = "float64"
	case IntegerType:
		fieldType = "int"
	case BooleanType:
		fieldType = "bool"
	case ArrayType:
		if field.ArrayOf != nil {
			elementType := g.getGoType(*field.ArrayOf)
			fieldType = fmt.Sprintf("[]%s", elementType)
		} else {
			fieldType = "[]interface{}"
		}
	case ObjectType:
		if field.NestedType != nil {
			fieldType = field.NestedType.Name
		} else {
			fieldType = "interface{}"
		}
	case EnumType:
		fieldType = "string"
	case JSONDateType:
		fieldType = "JSONDate"
	case CustomType:
		fieldType = field.CustomType
	default:
		fieldType = "interface{}"
	}

	if field.Optional {
		fieldType = "*" + fieldType
	}

	jsonTag := field.JSONTag
	if jsonTag == "" {
		jsonTag = fmt.Sprintf(`json:"%s"`, field.Name)
		if field.JSONOmitEmpty {
			jsonTag = fmt.Sprintf(`json:"%s,omitempty"`, field.Name)
		}
	}
	tags = append(tags, jsonTag)

	tagStr := "`" + strings.Join(tags, " ") + "`"

	result := fmt.Sprintf("\t%s %s %s\n",
		strings.Title(field.Name),
		fieldType,
		tagStr)

	if field.Description != "" {
		result = fmt.Sprintf("\t// %s\n%s", field.Description, result)
	}

	return result
}

func (g *GoGenerator) getGoType(typeDef TypeDefinition) string {
	switch typeDef.Type {
	case StringType:
		return "string"
	case NumberType:
		return "float64"
	case IntegerType:
		return "int"
	case BooleanType:
		return "bool"
	case ObjectType:
		return typeDef.Name
	case JSONDateType:
		return "JSONDate"
	default:
		return "interface{}"
	}
}

func (g *GoGenerator) hasJSONDateFields(typeDef TypeDefinition) bool {
	for _, field := range typeDef.Fields {
		if field.Type == JSONDateType {
			return true
		}
	}
	return false
}

func (g *GoGenerator) generateJSONDateMarshallers(typeDef TypeDefinition) string {
	var builder strings.Builder

	builder.WriteString(g.generateJSONDateType())

	return builder.String()
}

func (g *GoGenerator) generateJSONDateType() string {
	return `// JSONDate represents a date that serializes to {"type": "date", "value": "ISO8601"}
type JSONDate struct {
	time.Time
}

func (jd JSONDate) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf(` + "`" + `{"type": "date", "value": "%s"}` + "`" + `, jd.Time.Format(time.RFC3339))), nil
}

func (jd *JSONDate) UnmarshalJSON(data []byte) error {
	var obj struct {
		Type  string ` + "`json:\"type\"`" + `
		Value string ` + "`json:\"value\"`" + `
	}
	if err := json.Unmarshal(data, &obj); err != nil {
		return err
	}
	if obj.Type != "date" {
		return fmt.Errorf("expected type 'date', got '%s'", obj.Type)
	}
	t, err := time.Parse(time.RFC3339, obj.Value)
	if err != nil {
		return err
	}
	jd.Time = t
	return nil
}

func NewJSONDate(t time.Time) JSONDate {
	return JSONDate{Time: t}
}

func NewJSONDateNow() JSONDate {
	return JSONDate{Time: time.Now()}
}

`
}

func (g *GoGenerator) GeneratePackage(schema Schema) (string, error) {
	var builder strings.Builder

	builder.WriteString(fmt.Sprintf("package %s\n\n", g.packageName))
	builder.WriteString("import (\n")
	builder.WriteString("\t\"encoding/json\"\n")
	builder.WriteString("\t\"fmt\"\n")
	builder.WriteString("\t\"time\"\n")
	builder.WriteString(")\n\n")

	// Collect custom field types and generate their marshallers

	for _, typeDef := range schema.Types {
		typeCode, err := g.GenerateType(typeDef)
		if err != nil {
			return "", fmt.Errorf("failed to generate type %s: %w", typeDef.Name, err)
		}
		builder.WriteString(typeCode)
	}

	return builder.String(), nil
}

func (ts *TypeScriptGenerator) GenerateType(typeDef TypeDefinition) string {
	var builder strings.Builder

	if typeDef.Description != "" {
		builder.WriteString(fmt.Sprintf("/**\n * %s\n */\n", typeDef.Description))
	}

	// Handle enum types differently
	if typeDef.Type == EnumType {
		builder.WriteString(fmt.Sprintf("export type %s = ", typeDef.Name))
		for i, value := range typeDef.EnumValues {
			if i > 0 {
				builder.WriteString(" | ")
			}
			builder.WriteString(fmt.Sprintf("\"%s\"", value))
		}
		builder.WriteString(";\n\n")
		return builder.String()
	}

	// Handle interface types
	builder.WriteString(fmt.Sprintf("export interface %s {\n", typeDef.Name))

	for _, field := range typeDef.Fields {
		builder.WriteString(ts.generateTypeScriptField(field))
	}

	builder.WriteString("}\n\n")

	// Individual type converters are now handled by the combined utilities

	return builder.String()
}

func (ts *TypeScriptGenerator) generateTypeScriptField(field FieldDefinition) string {
	var fieldType string

	switch field.Type {
	case StringType:
		fieldType = "string"
	case NumberType, IntegerType:
		fieldType = "number"
	case BooleanType:
		fieldType = "boolean"
	case ArrayType:
		if field.ArrayOf != nil {
			elementType := ts.getTypeScriptType(*field.ArrayOf)
			fieldType = fmt.Sprintf("%s[]", elementType)
		} else {
			fieldType = "any[]"
		}
	case ObjectType:
		if field.NestedType != nil {
			fieldType = field.NestedType.Name
		} else {
			fieldType = "object"
		}
	case EnumType:
		if len(field.EnumValues) > 0 {
			values := make([]string, len(field.EnumValues))
			for i, v := range field.EnumValues {
				values[i] = fmt.Sprintf("'%s'", v)
			}
			fieldType = strings.Join(values, " | ")
		} else {
			fieldType = "string"
		}
	case JSONDateType:
		fieldType = "Date"
	case CustomType:
		fieldType = field.CustomType
	default:
		fieldType = "any"
	}

	optional := ""
	if field.Optional {
		optional = "?"
	}

	result := fmt.Sprintf("  %s%s: %s;\n", field.Name, optional, fieldType)

	if field.Description != "" {
		result = fmt.Sprintf("  /** %s */\n%s", field.Description, result)
	}

	return result
}

func (ts *TypeScriptGenerator) getTypeScriptType(typeDef TypeDefinition) string {
	switch typeDef.Type {
	case StringType:
		return "string"
	case NumberType, IntegerType:
		return "number"
	case BooleanType:
		return "boolean"
	case ObjectType:
		return typeDef.Name
	case JSONDateType:
		return "Date"
	default:
		return "any"
	}
}

func (ts *TypeScriptGenerator) hasJSONDateFields(typeDef TypeDefinition) bool {
	for _, field := range typeDef.Fields {
		if field.Type == JSONDateType {
			return true
		}
	}
	return false
}

func (ts *TypeScriptGenerator) GenerateModule(schema Schema) string {
	var builder strings.Builder

	builder.WriteString("// Generated TypeScript types\n\n")

	// Check what custom types we need utilities for
	needsJSONDate := ts.schemaHasJSONDateFields(schema)
	var customTypes []string

	if needsJSONDate {
		customTypes = append(customTypes, "JSONDate")
	}

	for _, typeDef := range schema.Types {
		builder.WriteString(ts.GenerateType(typeDef))
	}

	// Only generate utilities if we have custom types
	if len(customTypes) > 0 {
		builder.WriteString(ts.generateCombinedUtilities(customTypes))
	}

	return builder.String()
}

func (ts *TypeScriptGenerator) schemaHasJSONDateFields(schema Schema) bool {
	for _, typeDef := range schema.Types {
		if ts.hasJSONDateFields(typeDef) {
			return true
		}
	}
	return false
}

func (ts *TypeScriptGenerator) generateCombinedUtilities(customTypes []string) string {
	var builder strings.Builder

	builder.WriteString("// JSON utilities for custom types\n")

	// Generate combined replacer
	builder.WriteString("export function jsonReplacer(key: string, value: any): any {\n")

	for _, customType := range customTypes {
		switch customType {
		case "JSONDate":
			builder.WriteString("  // Handle Date objects\n")
			builder.WriteString("  if (value instanceof Date) {\n")
			builder.WriteString("    return { type: 'date', value: value.toISOString() };\n")
			builder.WriteString("  }\n\n")
		}
	}

	builder.WriteString("  return value;\n")
	builder.WriteString("}\n\n")

	// Generate combined reviver
	builder.WriteString("export function jsonReviver(key: string, value: any): any {\n")
	builder.WriteString("  if (value && typeof value === 'object' && value.type) {\n")

	for _, customType := range customTypes {
		switch customType {
		case "JSONDate":
			builder.WriteString("    // Handle JSONDate objects\n")
			builder.WriteString("    if (value.type === 'date') {\n")
			builder.WriteString("      return new Date(value.value);\n")
			builder.WriteString("    }\n\n")
		}
	}

	builder.WriteString("  }\n")
	builder.WriteString("  return value;\n")
	builder.WriteString("}\n\n")

	// Generate convenience functions for JSON.parse/stringify with custom types
	builder.WriteString("// Convenience functions\n")
	builder.WriteString("export function parseJSON<T = any>(json: string): T {\n")
	builder.WriteString("  return JSON.parse(json, jsonReviver);\n")
	builder.WriteString("}\n\n")

	builder.WriteString("export function stringifyJSON(obj: any, space?: string | number): string {\n")
	builder.WriteString("  return JSON.stringify(obj, jsonReplacer, space);\n")
	builder.WriteString("}\n\n")

	return builder.String()
}
