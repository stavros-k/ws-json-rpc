package typesystem

import (
	"bytes"
	"fmt"
	"go/format"
)

// ArrayNode represents an array type.
type ArrayNode struct {
	name          string
	desc          string
	itemType      *PropertyType
	rawDefinition string
}

// NewArrayNode creates a new ArrayNode.
func NewArrayNode(name, desc string, itemType *PropertyType, rawDef string) *ArrayNode {
	return &ArrayNode{
		name:          name,
		desc:          desc,
		itemType:      itemType,
		rawDefinition: rawDef,
	}
}

// GetName returns the array type name.
func (a *ArrayNode) GetName() string {
	return a.name
}

// GetDescription returns the array description.
func (a *ArrayNode) GetDescription() string {
	return a.desc
}

// GetKind returns the type kind.
func (a *ArrayNode) GetKind() TypeKind {
	return TypeKindArray
}

// GetRawDefinition returns the raw JSON definition.
func (a *ArrayNode) GetRawDefinition() string {
	return a.rawDefinition
}

// GetGoImports returns the Go imports needed for this array type.
func (a *ArrayNode) GetGoImports() []string {
	imports := make(map[string]struct{})
	a.itemType.CollectGoImports(imports)

	result := make([]string, 0, len(imports))
	for imp := range imports {
		result = append(result, imp)
	}
	return result
}

// ToGoString generates Go code for the array type.
func (a *ArrayNode) ToGoString() (string, error) {
	var buf bytes.Buffer

	buf.WriteString(fmt.Sprintf("// %s - %s\n", a.name, a.desc))

	itemTypeStr := a.itemType.ToGoType()
	buf.WriteString(fmt.Sprintf("type %s []%s\n", a.name, itemTypeStr))

	// Format the generated code
	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		return "", fmt.Errorf("failed to format Go code: %w\nGenerated code:\n%s", err, buf.String())
	}

	return string(formatted), nil
}

// ToTypeScriptString generates TypeScript code for the array type.
func (a *ArrayNode) ToTypeScriptString() (string, error) {
	var buf bytes.Buffer

	buf.WriteString(fmt.Sprintf("// %s\n", a.desc))

	itemTypeStr := a.itemType.ToTypeScriptType()
	buf.WriteString(fmt.Sprintf("export type %s = Array<%s>;\n", a.name, itemTypeStr))

	return buf.String(), nil
}

// ToCSharpString generates C# code for the array type.
func (a *ArrayNode) ToCSharpString() (string, error) {
	var buf bytes.Buffer

	buf.WriteString(fmt.Sprintf("/// <summary>\n/// %s\n/// </summary>\n", a.desc))

	itemTypeStr := a.itemType.ToCSharpType()
	buf.WriteString(fmt.Sprintf("public class %s : List<%s> { }\n", a.name, itemTypeStr))

	return buf.String(), nil
}
