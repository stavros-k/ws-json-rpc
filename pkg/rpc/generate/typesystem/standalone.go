package typesystem

import (
	"fmt"
	"sort"
	"strings"
)

// ToStandaloneGo generates complete Go code for a single type with package and imports.
// This is useful for documentation or embedding standalone type definitions.
func ToStandaloneGo(node TypeNode, packageName string) (string, error) {
	var buf strings.Builder

	// Package declaration
	buf.WriteString(fmt.Sprintf("package %s\n\n", packageName))

	// Add imports if needed
	imports := node.GetGoImports()
	if len(imports) > 0 {
		sort.Strings(imports)
		buf.WriteString("import (\n")
		for _, imp := range imports {
			buf.WriteString(fmt.Sprintf("\t\"%s\"\n", imp))
		}
		buf.WriteString(")\n\n")
	}

	// Add type definition
	code, err := node.ToGoString()
	if err != nil {
		return "", err
	}
	buf.WriteString(code)

	return buf.String(), nil
}

// ToStandaloneTypeScript generates complete TypeScript code for a single type.
// TypeScript doesn't typically need imports for primitive types, so this is mainly
// for consistency and potential future extension.
func ToStandaloneTypeScript(node TypeNode) (string, error) {
	// TypeScript doesn't need package declarations or imports for most cases
	// Just return the type definition directly
	return node.ToTypeScriptString()
}

// ToStandaloneCSharp generates complete C# code for a single type with namespace and usings.
// This is useful for documentation or embedding standalone type definitions.
func ToStandaloneCSharp(node TypeNode, namespace string) (string, error) {
	var buf strings.Builder

	// Add using statements for common types
	needsUsings := false
	code, err := node.ToCSharpString()
	if err != nil {
		return "", err
	}

	// Check if we need System.Text.Json for JsonPropertyName
	if strings.Contains(code, "[JsonPropertyName") {
		needsUsings = true
	}

	if needsUsings {
		buf.WriteString("using System.Text.Json.Serialization;\n\n")
	}

	// Namespace declaration
	buf.WriteString(fmt.Sprintf("namespace %s\n{\n", namespace))

	// Indent the type definition
	lines := strings.Split(strings.TrimSpace(code), "\n")
	for _, line := range lines {
		if line != "" {
			buf.WriteString("    ")
		}
		buf.WriteString(line)
		buf.WriteString("\n")
	}

	buf.WriteString("}\n")

	return buf.String(), nil
}
