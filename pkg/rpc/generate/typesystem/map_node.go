package typesystem

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
