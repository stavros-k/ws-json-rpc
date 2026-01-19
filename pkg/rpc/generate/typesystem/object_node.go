package typesystem

import (
	"sort"
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
