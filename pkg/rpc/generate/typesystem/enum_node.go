package typesystem

import (
	"bytes"
	"fmt"
	"go/format"
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

// GetGoImports returns the Go imports needed for this enum type.
func (e *EnumNode) GetGoImports() []string {
	return nil // Enums don't need any imports
}

// GetValues returns the enum values.
func (e *EnumNode) GetValues() []EnumValue {
	return e.values
}

// ToGoString generates Go code for the enum type.
func (e *EnumNode) ToGoString() (string, error) {
	var buf bytes.Buffer

	// Generate type definition
	buf.WriteString(fmt.Sprintf("// %s - %s\n", e.name, e.desc))
	buf.WriteString(fmt.Sprintf("type %s string\n\n", e.name))

	// Generate constants
	buf.WriteString("const (\n")
	for _, val := range e.values {
		constName := e.generateConstNameWithPrefix(val.Value)
		if val.Description != "" {
			buf.WriteString(fmt.Sprintf("\t// %s\n", val.Description))
		}
		buf.WriteString(fmt.Sprintf("\t%s %s = %q\n", constName, e.name, val.Value))
	}
	buf.WriteString(")\n\n")

	// Generate Valid() method
	buf.WriteString(fmt.Sprintf("// Valid returns true if the %s value is valid\n", e.name))
	buf.WriteString(fmt.Sprintf("func (e %s) Valid() bool {\n", e.name))
	buf.WriteString("\tswitch e {\n")
	buf.WriteString("\tcase ")

	constNames := make([]string, len(e.values))
	for i, val := range e.values {
		constNames[i] = e.generateConstNameWithPrefix(val.Value)
	}
	buf.WriteString(strings.Join(constNames, ", "))
	buf.WriteString(":\n")
	buf.WriteString("\t\treturn true\n")
	buf.WriteString("\tdefault:\n")
	buf.WriteString("\t\treturn false\n")
	buf.WriteString("\t}\n")
	buf.WriteString("}\n")

	// Format the generated code
	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		return "", fmt.Errorf("failed to format Go code: %w\nGenerated code:\n%s", err, buf.String())
	}

	return string(formatted), nil
}

// ToTypeScriptString generates TypeScript code for the enum type.
func (e *EnumNode) ToTypeScriptString() (string, error) {
	var buf bytes.Buffer

	buf.WriteString(fmt.Sprintf("// %s\n", e.desc))
	buf.WriteString(fmt.Sprintf("export const %s = {\n", e.name))

	for _, val := range e.values {
		constName := e.generateConstName(val.Value)
		if val.Description != "" {
			buf.WriteString(fmt.Sprintf("  /** %s */\n", val.Description))
		}
		buf.WriteString(fmt.Sprintf("  %s: %q,\n", constName, val.Value))
	}

	buf.WriteString("} as const;\n\n")
	buf.WriteString(fmt.Sprintf("export type %s = typeof %s[keyof typeof %s];\n\n", e.name, e.name, e.name))

	// Generate type guard with switch statement
	buf.WriteString(fmt.Sprintf("export function is%s(value: unknown): value is %s {\n", e.name, e.name))
	buf.WriteString("\tswitch (value) {\n")
	for _, val := range e.values {
		buf.WriteString(fmt.Sprintf("\t\tcase %q:\n", val.Value))
	}
	buf.WriteString("\t\t\treturn true;\n")
	buf.WriteString("\t\tdefault:\n")
	buf.WriteString("\t\t\treturn false;\n")
	buf.WriteString("\t}\n")
	buf.WriteString("}\n")

	return buf.String(), nil
}

// ToCSharpString generates C# code for the enum type.
func (e *EnumNode) ToCSharpString() (string, error) {
	var buf bytes.Buffer

	buf.WriteString(fmt.Sprintf("/// <summary>\n/// %s\n/// </summary>\n", e.desc))
	buf.WriteString(fmt.Sprintf("public static class %s\n{\n", e.name))

	// Generate constants
	for _, val := range e.values {
		constName := e.generateConstName(val.Value)
		if val.Description != "" {
			buf.WriteString(fmt.Sprintf("    /// <summary>%s</summary>\n", val.Description))
		}
		buf.WriteString(fmt.Sprintf("    public const string %s = %q;\n", constName, val.Value))
	}

	buf.WriteString("\n")

	// Generate IsValid method
	buf.WriteString("    /// <summary>\n")
	buf.WriteString(fmt.Sprintf("    /// Returns true if the value is a valid %s\n", e.name))
	buf.WriteString("    /// </summary>\n")
	buf.WriteString("    public static bool IsValid(string value)\n")
	buf.WriteString("    {\n")
	buf.WriteString("        switch (value)\n")
	buf.WriteString("        {\n")

	for _, val := range e.values {
		buf.WriteString(fmt.Sprintf("            case %q:\n", val.Value))
	}

	buf.WriteString("                return true;\n")
	buf.WriteString("            default:\n")
	buf.WriteString("                return false;\n")
	buf.WriteString("        }\n")
	buf.WriteString("    }\n")

	buf.WriteString("}\n")

	return buf.String(), nil
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

// generateConstNameWithPrefix generates a constant name from an enum value with type prefix.
// Examples: "user.create" -> "MyTypeUserCreate", "active" -> "MyTypeActive"
func (e *EnumNode) generateConstNameWithPrefix(value string) string {
	return e.name + e.generateConstName(value)
}
