package typesystem

import (
	"bytes"
	"fmt"
	"go/format"
	"ws-json-rpc/pkg/utils"
)

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

// GetGoImports returns the Go imports needed for this alias type.
func (a *AliasNode) GetGoImports() []string {
	imports := make(map[string]struct{})
	a.targetType.CollectGoImports(imports)

	result := make([]string, 0, len(imports))
	for imp := range imports {
		result = append(result, imp)
	}
	return result
}

// GetTargetType returns the target type reference string.
func (a *AliasNode) GetTargetType() string {
	if a.targetType.Ref != "" {
		return a.targetType.Ref
	}
	// For non-ref types, return the type string representation
	return a.targetType.ToGoType()
}

// IsTargetRef returns whether the target type is a reference to another type.
func (a *AliasNode) IsTargetRef() bool {
	return a.targetType.Ref != ""
}

// ToGoString generates Go code for the alias type.
func (a *AliasNode) ToGoString() (string, error) {
	var buf bytes.Buffer

	// Convert name to proper Go naming conventions
	goName := utils.ToPascalCase(a.name)

	buf.WriteString(fmt.Sprintf("// %s - %s\n", goName, a.desc))

	targetTypeStr := a.targetType.ToGoType()
	buf.WriteString(fmt.Sprintf("type %s %s\n", goName, targetTypeStr))

	// Format the generated code
	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		return "", fmt.Errorf("failed to format Go code: %w\nGenerated code:\n%s", err, buf.String())
	}

	return string(formatted), nil
}

// ToTypeScriptString generates TypeScript code for the alias type.
func (a *AliasNode) ToTypeScriptString() (string, error) {
	var buf bytes.Buffer

	// Convert name to proper TypeScript naming conventions (PascalCase)
	tsName := utils.ToPascalCase(a.name)

	buf.WriteString(fmt.Sprintf("// %s\n", a.desc))

	targetTypeStr := a.targetType.ToTypeScriptType()
	buf.WriteString(fmt.Sprintf("export type %s = %s;\n", tsName, targetTypeStr))

	return buf.String(), nil
}
