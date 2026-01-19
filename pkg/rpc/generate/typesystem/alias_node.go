package typesystem

// AliasNode represents a type alias.
type AliasNode struct {
	name          string
	desc          string
	targetType    *PropertyType
	rawDefinition string
}

// NewAliasNode creates a new AliasNode.
func NewAliasNode(name, desc string, targetType *PropertyType, rawDef string) *AliasNode {
	return &AliasNode{
		name:          name,
		desc:          desc,
		targetType:    targetType,
		rawDefinition: rawDef,
	}
}

// GetName returns the alias type name.
func (a *AliasNode) GetName() string {
	return a.name
}

// GetDescription returns the alias description.
func (a *AliasNode) GetDescription() string {
	return a.desc
}

// GetKind returns the type kind.
func (a *AliasNode) GetKind() TypeKind {
	return TypeKindAlias
}

// GetRawDefinition returns the raw JSON definition.
func (a *AliasNode) GetRawDefinition() string {
	return a.rawDefinition
}
