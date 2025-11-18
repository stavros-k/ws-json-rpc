package typesystem

import (
	"bytes"
	"fmt"
	"go/format"
	"sort"
	"ws-json-rpc/pkg/utils"
)

// ObjectNode represents an object type with fields.
type ObjectNode struct {
	name          string
	desc          string
	fields        []FieldMetadata
	rawDefinition string
}

// NewObjectNode creates a new ObjectNode.
func NewObjectNode(name, desc string, fields []FieldMetadata, rawDef string) *ObjectNode {
	return &ObjectNode{
		name:          name,
		desc:          desc,
		fields:        fields,
		rawDefinition: rawDef,
	}
}

// GetName returns the object type name.
func (o *ObjectNode) GetName() string {
	return o.name
}

// GetDescription returns the object description.
func (o *ObjectNode) GetDescription() string {
	return o.desc
}

// GetKind returns the type kind.
func (o *ObjectNode) GetKind() TypeKind {
	return TypeKindObject
}

// GetRawDefinition returns the raw JSON definition.
func (o *ObjectNode) GetRawDefinition() string {
	return o.rawDefinition
}

// GetGoImports returns the Go imports needed for this object type.
func (o *ObjectNode) GetGoImports() []string {
	imports := make(map[string]struct{})

	for _, field := range o.fields {
		field.Type.CollectGoImports(imports)
	}

	// Convert map to sorted slice
	result := make([]string, 0, len(imports))
	for imp := range imports {
		result = append(result, imp)
	}
	return result
}

// GetFields returns the object fields.
func (o *ObjectNode) GetFields() []FieldMetadata {
	return o.fields
}

// GetReferences returns a list of type names that this object references.
func (o *ObjectNode) GetReferences() []string {
	refs := make(map[string]struct{})
	for _, field := range o.fields {
		CollectRefs(field.Type, refs)
	}

	result := make([]string, 0, len(refs))
	for ref := range refs {
		result = append(result, ref)
	}
	sort.Strings(result)
	return result
}

// ToGoString generates Go code for the object type.
func (o *ObjectNode) ToGoString() (string, error) {
	var buf bytes.Buffer

	// Generate type definition
	buf.WriteString(fmt.Sprintf("// %s - %s\n", o.name, o.desc))
	buf.WriteString(fmt.Sprintf("type %s struct {\n", o.name))

	// Sort fields alphabetically for consistent output
	fields := make([]FieldMetadata, len(o.fields))
	copy(fields, o.fields)
	sort.Slice(fields, func(i, j int) bool {
		return fields[i].Name < fields[j].Name
	})

	for _, field := range fields {
		// Generate field comment
		if field.Description != "" {
			buf.WriteString(fmt.Sprintf("\t// %s\n", field.Description))
		}

		// Generate field with proper initialism handling
		fieldName := utils.ToPascalCase(field.Name)
		fieldType := field.Type.ToGoType()
		jsonTag := field.Name

		if field.Optional {
			jsonTag += ",omitzero"
		}

		buf.WriteString(fmt.Sprintf("\t%s %s `json:\"%s\"`\n", fieldName, fieldType, jsonTag))
	}

	buf.WriteString("}\n")

	// Format the generated code
	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		return "", fmt.Errorf("failed to format Go code: %w\nGenerated code:\n%s", err, buf.String())
	}

	return string(formatted), nil
}

// ToTypeScriptString generates TypeScript code for the object type.
func (o *ObjectNode) ToTypeScriptString() (string, error) {
	var buf bytes.Buffer

	buf.WriteString(fmt.Sprintf("// %s\n", o.desc))
	buf.WriteString(fmt.Sprintf("export type %s = {\n", o.name))

	// Sort fields alphabetically
	fields := make([]FieldMetadata, len(o.fields))
	copy(fields, o.fields)
	sort.Slice(fields, func(i, j int) bool {
		return fields[i].Name < fields[j].Name
	})

	for _, field := range fields {
		if field.Description != "" {
			buf.WriteString(fmt.Sprintf("  /** %s */\n", field.Description))
		}

		optionalMark := ""
		if field.Optional {
			optionalMark = "?"
		}

		fieldType := field.Type.ToTypeScriptType()
		buf.WriteString(fmt.Sprintf("  %s%s: %s;\n", field.Name, optionalMark, fieldType))
	}

	buf.WriteString("};\n")

	return buf.String(), nil
}

// ToCSharpString generates C# code for the object type.
func (o *ObjectNode) ToCSharpString() (string, error) {
	var buf bytes.Buffer

	buf.WriteString(fmt.Sprintf("/// <summary>\n/// %s\n/// </summary>\n", o.desc))
	buf.WriteString(fmt.Sprintf("public class %s\n{\n", o.name))

	// Sort fields alphabetically
	fields := make([]FieldMetadata, len(o.fields))
	copy(fields, o.fields)
	sort.Slice(fields, func(i, j int) bool {
		return fields[i].Name < fields[j].Name
	})

	for i, field := range fields {
		// Add empty line between fields (but not before the first one)
		if i > 0 {
			buf.WriteString("\n")
		}

		if field.Description != "" {
			buf.WriteString(fmt.Sprintf("    /// <summary>%s</summary>\n", field.Description))
		}

		fieldName := utils.ToPascalCase(field.Name)
		fieldType := field.Type.ToCSharpType()

		// Add JSON property attribute for proper serialization
		buf.WriteString(fmt.Sprintf("    [JsonPropertyName(\"%s\")]\n", field.Name))
		buf.WriteString(fmt.Sprintf("    public %s %s { get; set; }\n", fieldType, fieldName))
	}

	buf.WriteString("}\n")

	return buf.String(), nil
}
