package typesystem

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
