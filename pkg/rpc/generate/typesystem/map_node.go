package typesystem

import (
	"bytes"
	"fmt"
	"go/format"
)

// MapNode represents a map/dictionary type.
type MapNode struct {
	name          string
	desc          string
	keyType       *PropertyType
	valueType     *PropertyType
	rawDefinition string
}

// NewMapNode creates a new MapNode.
func NewMapNode(name, desc string, keyType, valueType *PropertyType, rawDef string) *MapNode {
	return &MapNode{
		name:          name,
		desc:          desc,
		keyType:       keyType,
		valueType:     valueType,
		rawDefinition: rawDef,
	}
}

// GetName returns the map type name.
func (m *MapNode) GetName() string {
	return m.name
}

// GetDescription returns the map description.
func (m *MapNode) GetDescription() string {
	return m.desc
}

// GetKind returns the type kind.
func (m *MapNode) GetKind() TypeKind {
	return TypeKindMap
}

// GetRawDefinition returns the raw JSON definition.
func (m *MapNode) GetRawDefinition() string {
	return m.rawDefinition
}

// GetGoImports returns the Go imports needed for this map type.
func (m *MapNode) GetGoImports() []string {
	imports := make(map[string]struct{})
	m.keyType.CollectGoImports(imports)
	m.valueType.CollectGoImports(imports)

	result := make([]string, 0, len(imports))
	for imp := range imports {
		result = append(result, imp)
	}
	return result
}

// GetKeyType returns the key type reference string.
func (m *MapNode) GetKeyType() string {
	if m.keyType.Ref != "" {
		return m.keyType.Ref
	}
	return m.keyType.ToGoType()
}

// GetValueType returns the value type reference string.
func (m *MapNode) GetValueType() string {
	if m.valueType.Ref != "" {
		return m.valueType.Ref
	}
	return m.valueType.ToGoType()
}

// IsValueRef returns whether the value type is a reference to another type.
func (m *MapNode) IsValueRef() bool {
	return m.valueType.Ref != ""
}

// GetReferences returns a list of type names that this map references.
func (m *MapNode) GetReferences() []string {
	refs := make(map[string]struct{})
	CollectRefs(m.keyType, refs)
	CollectRefs(m.valueType, refs)

	result := make([]string, 0, len(refs))
	for ref := range refs {
		result = append(result, ref)
	}
	return result
}

// ToGoString generates Go code for the map type.
func (m *MapNode) ToGoString() (string, error) {
	var buf bytes.Buffer

	buf.WriteString(fmt.Sprintf("// %s - %s\n", m.name, m.desc))

	keyTypeStr := m.keyType.ToGoType()
	valueTypeStr := m.valueType.ToGoType()
	buf.WriteString(fmt.Sprintf("type %s map[%s]%s\n", m.name, keyTypeStr, valueTypeStr))

	// Format the generated code
	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		return "", fmt.Errorf("failed to format Go code: %w\nGenerated code:\n%s", err, buf.String())
	}

	return string(formatted), nil
}

// ToTypeScriptString generates TypeScript code for the map type.
func (m *MapNode) ToTypeScriptString() (string, error) {
	var buf bytes.Buffer

	buf.WriteString(fmt.Sprintf("// %s\n", m.desc))

	valueTypeStr := m.valueType.ToTypeScriptType()
	buf.WriteString(fmt.Sprintf("export type %s = Record<string, %s>;\n", m.name, valueTypeStr))

	return buf.String(), nil
}

// ToCSharpString generates C# code for the map type.
func (m *MapNode) ToCSharpString() (string, error) {
	var buf bytes.Buffer

	buf.WriteString(fmt.Sprintf("/// <summary>\n/// %s\n/// </summary>\n", m.desc))

	keyTypeStr := m.keyType.ToCSharpType()
	valueTypeStr := m.valueType.ToCSharpType()
	buf.WriteString(fmt.Sprintf("public class %s : Dictionary<%s, %s> { }\n", m.name, keyTypeStr, valueTypeStr))

	return buf.String(), nil
}
