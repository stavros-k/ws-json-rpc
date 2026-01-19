package typesystem

import (
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// EnumValue represents a single value in an enum.
type EnumValue struct {
	Value       string
	Description string
}

// EnumNode represents an enumeration type.
type EnumNode struct {
	name          string
	desc          string
	values        []EnumValue
	rawDefinition string
}

// NewEnumNode creates a new EnumNode.
func NewEnumNode(name, desc string, values []EnumValue, rawDef string) *EnumNode {
	return &EnumNode{
		name:          name,
		desc:          desc,
		values:        values,
		rawDefinition: rawDef,
	}
}

// GetName returns the enum type name.
func (e *EnumNode) GetName() string {
	return e.name
}

// GetDescription returns the enum description.
func (e *EnumNode) GetDescription() string {
	return e.desc
}

// GetKind returns the type kind.
func (e *EnumNode) GetKind() TypeKind {
	return TypeKindEnum
}

// GetRawDefinition returns the raw JSON definition.
func (e *EnumNode) GetRawDefinition() string {
	return e.rawDefinition
}

// GetValues returns the enum values.
func (e *EnumNode) GetValues() []EnumValue {
	return e.values
}

// generateConstName generates a constant name from an enum value without prefix.
// Examples: "user.create" -> "UserCreate", "dark-blue" -> "DarkBlue"
func (e *EnumNode) generateConstName(value string) string {
	// Split by dots and dashes
	parts := strings.FieldsFunc(value, func(r rune) bool {
		return r == '.' || r == '-' || r == '_'
	})

	// Title case each part
	caser := cases.Title(language.English)
	for i, part := range parts {
		parts[i] = caser.String(part)
	}

	return strings.Join(parts, "")
}
