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
