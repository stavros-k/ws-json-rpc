// Package generate provides API documentation generation from Go type definitions.
// This file (guts.go) handles TypeScript AST parsing and metadata extraction
// using the github.com/coder/guts library to parse Go structs and generate
// TypeScript type definitions with full metadata.
package generate

import (
	"fmt"
	"log/slog"
	"os"
	"sort"
	"strings"

	"github.com/coder/guts"
	"github.com/coder/guts/bindings"
	"github.com/coder/guts/config"
)

// GutsGenerator handles TypeScript AST parsing and metadata extraction from Go types.
type GutsGenerator struct {
	tsParser *guts.Typescript
	vm       *bindings.Bindings
	l        *slog.Logger
}

// NewGutsGenerator parses the Go types directory and generates a TypeScript AST for metadata extraction.
func NewGutsGenerator(l *slog.Logger, goTypesDirPath string) (*GutsGenerator, error) {
	var err error
	l = l.With(slog.String("component", "guts-generator"))
	l.Debug("Creating guts generator", slog.String("goTypesDirPath", goTypesDirPath))

	gutsGenerator := &GutsGenerator{l: l}

	gutsGenerator.vm, err = bindings.New()
	if err != nil {
		return nil, fmt.Errorf("failed to create bindings VM: %w", err)
	}

	gutsGenerator.tsParser, err = newTypescriptASTFromGoTypesDir(l, goTypesDirPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create TypeScript AST from go types dir: %w", err)
	}

	l.Info("Guts generator created successfully")
	return gutsGenerator, nil
}

// newTypescriptASTFromGoTypesDir creates a TypeScript AST from Go type definitions,
// preserving comments and applying transformations for TypeScript compatibility.
func newTypescriptASTFromGoTypesDir(l *slog.Logger, goTypesDirPath string) (*guts.Typescript, error) {
	l.Debug("Parsing Go types directory", slog.String("path", goTypesDirPath))

	goParser, err := guts.NewGolangParser()
	if err != nil {
		return nil, fmt.Errorf("failed to create guts parser: %w", err)
	}
	goParser.PreserveComments()

	if _, err := os.Stat(goTypesDirPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("go types dir path %s does not exist", goTypesDirPath)
	}

	if err := goParser.IncludeGenerate(goTypesDirPath); err != nil {
		return nil, fmt.Errorf("failed to include go types dir for parsing: %w", err)
	}

	l.Debug("Generating TypeScript AST from Go types")

	ts, err := goParser.ToTypescript()
	if err != nil {
		return nil, fmt.Errorf("failed to generate TypeScript AST: %w", err)
	}

	ts.ApplyMutations(
		config.EnumAsTypes,
		config.EnumLists,
		config.ExportTypes,
		config.InterfaceToType,
	)

	l.Debug("TypeScript AST generated successfully")
	return ts, nil
}

// WriteTypescriptASTToFile serializes and writes TypeScript type definitions to a file.
func (g *GutsGenerator) WriteTypescriptASTToFile(ts *guts.Typescript, filePath string) error {
	g.l.Debug("Serializing TypeScript AST", slog.String("file", filePath))

	str, err := ts.Serialize()
	if err != nil {
		return fmt.Errorf("failed to serialize TypeScript AST: %w", err)
	}

	err = os.WriteFile(filePath, []byte(str), 0644)
	if err != nil {
		return fmt.Errorf("failed to write TypeScript AST to file: %w", err)
	}

	g.l.Info("TypeScript types written", slog.String("file", filePath))
	return nil
}

// SerializeNode converts a type name to its TypeScript string representation.
func (g *GutsGenerator) SerializeNode(name string) (string, error) {
	g.l.Debug("Serializing node", slog.String("type", name))

	node, exists := g.tsParser.Node(name)
	if !exists {
		return "", fmt.Errorf("node %s not found in TypeScript AST", name)
	}

	typescriptNode, err := g.vm.ToTypescriptNode(node)
	if err != nil {
		return "", fmt.Errorf("failed to convert node to TypeScript: %w", err)
	}

	serializedNode, err := g.vm.SerializeToTypescript(typescriptNode)
	if err != nil {
		return "", fmt.Errorf("failed to serialize node to TypeScript: %w", err)
	}

	var str strings.Builder
	for line := range strings.SplitSeq(serializedNode, "\n") {
		if strings.HasPrefix(line, "// From") {
			continue
		}
		str.WriteString(line + "\n")
	}

	return strings.TrimSpace(str.String()), nil
}

// ExtractReferences returns all type names referenced by the given type, deduplicated and sorted.
func (g *GutsGenerator) ExtractReferences(name string) ([]string, error) {
	node, exists := g.tsParser.Node(name)
	if !exists {
		return nil, fmt.Errorf("node %s not found in TypeScript AST", name)
	}

	refs := make(map[string]struct{})
	g.collectTypeReferences(node, refs)

	// Convert to sorted slice
	refList := make([]string, 0, len(refs))
	for ref := range refs {
		refList = append(refList, ref)
	}

	// Sort for deterministic output
	sort.Strings(refList)

	g.l.Debug("Extracted type references", slog.String("type", name), slog.Int("count", len(refList)))
	return refList, nil
}

// ExtractFields returns field metadata for a type, including types, descriptions, and optional flags.
func (g *GutsGenerator) ExtractFields(name string) ([]FieldMetadata, error) {
	node, exists := g.tsParser.Node(name)
	if !exists {
		return nil, fmt.Errorf("node %s not found in TypeScript AST", name)
	}

	var fields []FieldMetadata

	switch n := node.(type) {
	case *bindings.Alias:
		// Type alias - extract fields from the aliased type if it's a type literal
		fields = g.extractFieldsFromExpressionType(n.Type)

	case *bindings.Interface:
		// Interface - extract fields from property signatures
		for _, prop := range n.Fields {
			typeStr, err := g.serializeExpressionType(prop.Type)
			if err != nil {
				g.l.Warn("Failed to serialize field type", slog.String("type", name), slog.String("field", prop.Name), slog.String("error", err.Error()))
				return nil, fmt.Errorf("failed to serialize type for field %s in %s: %w", prop.Name, name, err)
			}

			fields = append(fields, FieldMetadata{
				Name:        prop.Name,
				Type:        typeStr,
				Description: g.extractComments(prop.SupportComments),
				Optional:    prop.QuestionToken,
				EnumValues:  g.extractEnumValues(prop.Type),
			})
		}
	}

	g.l.Debug("Extracted fields", slog.String("type", name), slog.Int("count", len(fields)))
	return fields, nil
}

// ExtractTypeDescription extracts the description from a type's comments.
func (g *GutsGenerator) ExtractTypeDescription(name string) (string, error) {
	node, exists := g.tsParser.Node(name)
	if !exists {
		return "", fmt.Errorf("node %s not found in TypeScript AST", name)
	}

	switch n := node.(type) {
	case *bindings.Alias:
		return g.extractComments(n.SupportComments), nil

	case *bindings.Interface:
		return g.extractComments(n.SupportComments), nil

	default:
		return "", fmt.Errorf("node %s is not a supported type (%T)", name, node)
	}
}

// ExtractTypeKind returns a human-readable type classification ("Object", "String Enum", "Union", etc.).
func (g *GutsGenerator) ExtractTypeKind(name string) (string, error) {
	node, exists := g.tsParser.Node(name)
	if !exists {
		return "", fmt.Errorf("node %s not found in TypeScript AST", name)
	}

	switch n := node.(type) {
	case *bindings.Alias:
		kind, err := g.getTypeKindFromExpression(n.Type)
		if err != nil {
			return "", fmt.Errorf("failed to get type kind for alias %s: %w", name, err)
		}
		g.l.Debug("Extracted type kind", slog.String("type", name), slog.String("kind", kind))
		return kind, nil

	case *bindings.Interface:
		g.l.Debug("Extracted type kind", slog.String("type", name), slog.String("kind", "Object"))
		return "Object", nil

	default:
		return "", fmt.Errorf("node %s is not a supported type (%T)", name, node)
	}
}

// ExtractTypeEnumValues returns string literal values if the type is a string enum.
func (g *GutsGenerator) ExtractTypeEnumValues(name string) ([]string, error) {
	node, exists := g.tsParser.Node(name)
	if !exists {
		return nil, fmt.Errorf("node %s not found in TypeScript AST", name)
	}

	switch n := node.(type) {
	case *bindings.Alias:
		// Reuse extractEnumValues logic for the alias type
		values := g.extractEnumValues(n.Type)
		if len(values) > 0 {
			g.l.Debug("Extracted enum values", slog.String("type", name), slog.Int("count", len(values)))
		}
		return values, nil
	default:
		return nil, fmt.Errorf("node %s is not a supported type (%T)", name, node)
	}
}

// getTypeKindFromExpression returns a human-readable type classification from an expression type.
func (g *GutsGenerator) getTypeKindFromExpression(expr bindings.ExpressionType) (string, error) {
	if expr == nil {
		return "", fmt.Errorf("expression type is nil")
	}

	switch e := expr.(type) {
	case *bindings.UnionType:
		// Check if it's a string/number enum (all members are literals of same type)
		allString, allNumber := true, true

		for _, member := range e.Types {
			lit, ok := member.(*bindings.LiteralType)
			if !ok {
				allString, allNumber = false, false
				break
			}

			switch lit.Value.(type) {
			case string:
				allNumber = false
			case int, float64:
				allString = false
			default:
				allString, allNumber = false, false
			}
		}

		switch {
		case allString:
			return "String Enum", nil
		case allNumber:
			return "Number Enum", nil
		default:
			return "Union", nil
		}

	case *bindings.TypeLiteralNode:
		return "Object", nil

	case *bindings.ArrayLiteralType:
		return "Array", nil

	case *bindings.ReferenceType:
		// Try to resolve the reference and get its kind
		refName := e.Name.String()
		refNode, exists := g.tsParser.Node(refName)
		if !exists {
			return "Type Reference", nil
		}

		// Check if it's an alias
		if alias, ok := refNode.(*bindings.Alias); ok {
			return g.getTypeKindFromExpression(alias.Type)
		}

		return "Type Reference", nil

	case *bindings.LiteralKeyword:
		keyword := string(*e)
		switch keyword {
		case "StringKeyword":
			return "String", nil
		case "NumberKeyword":
			return "Number", nil
		case "BooleanKeyword":
			return "Boolean", nil
		case "NullKeyword":
			return "Null", nil
		case "UndefinedKeyword":
			return "Undefined", nil
		case "VoidKeyword":
			return "Void", nil
		default:
			return "Primitive", nil
		}

	default:
		return "Type Alias", nil
	}
}

// extractFieldsFromExpressionType extracts fields from type literals.
// Returns nil if not a type literal. Skips fields that fail serialization with a warning.
func (g *GutsGenerator) extractFieldsFromExpressionType(expr bindings.ExpressionType) []FieldMetadata {
	typeLiteral, ok := expr.(*bindings.TypeLiteralNode)
	if !ok {
		return nil
	}

	var fields []FieldMetadata
	for _, member := range typeLiteral.Members {
		typeStr, err := g.serializeExpressionType(member.Type)
		if err != nil {
			g.l.Warn("Failed to serialize field type in type literal", slog.String("field", member.Name), slog.String("error", err.Error()))
			continue
		}

		fields = append(fields, FieldMetadata{
			Name:        member.Name,
			Type:        typeStr,
			Description: g.extractComments(member.SupportComments),
			Optional:    member.QuestionToken,
			EnumValues:  g.extractEnumValues(member.Type),
		})
	}

	return fields
}

// serializeExpressionType converts an expression type to its TypeScript string representation.
func (g *GutsGenerator) serializeExpressionType(expr bindings.ExpressionType) (string, error) {
	if expr == nil {
		return "", fmt.Errorf("expression type is nil")
	}

	// Convert expression to TypeScript node and serialize
	tsNode, err := g.vm.ToTypescriptExpressionNode(expr)
	if err != nil {
		return "", fmt.Errorf("failed to convert expression to TypeScript node: %w", err)
	}

	serialized, err := g.vm.SerializeToTypescript(tsNode)
	if err != nil {
		return "", fmt.Errorf("failed to serialize TypeScript node: %w", err)
	}

	return serialized, nil
}

// extractComments concatenates all comments into a single space-separated string.
func (g *GutsGenerator) extractComments(sc bindings.SupportComments) string {
	comments := sc.Comments()
	if len(comments) == 0 {
		return ""
	}

	var builder strings.Builder
	for i, comment := range comments {
		if i > 0 {
			builder.WriteString(" ")
		}
		builder.WriteString(strings.TrimSpace(comment.Text))
	}

	return builder.String()
}

// extractEnumValues extracts string literal values from string enum types.
// Handles both direct unions and references to union types.
func (g *GutsGenerator) extractEnumValues(expr bindings.ExpressionType) []string {
	if expr == nil {
		return nil
	}

	// Check if it's a direct union type
	if union, ok := expr.(*bindings.UnionType); ok {
		return g.extractLiteralsFromUnion(union)
	}

	// Check if it's a reference to another type (like EventKind)
	ref, ok := expr.(*bindings.ReferenceType)
	if !ok {
		return nil
	}

	node, exists := g.tsParser.Node(ref.Name.String())
	if !exists {
		return nil
	}

	// Check if the referenced type is an alias to a union
	alias, ok := node.(*bindings.Alias)
	if !ok {
		return nil
	}

	union, ok := alias.Type.(*bindings.UnionType)
	if !ok {
		return nil
	}

	return g.extractLiteralsFromUnion(union)
}

// extractLiteralsFromUnion extracts string literal values from a union, ignoring other types.
func (g *GutsGenerator) extractLiteralsFromUnion(union *bindings.UnionType) []string {
	var values []string
	for _, member := range union.Types {
		lit, ok := member.(*bindings.LiteralType)
		if !ok {
			continue
		}

		// Check if the literal value is a string
		strVal, ok := lit.Value.(string)
		if !ok {
			continue
		}

		values = append(values, strVal)
	}
	return values
}

// collectTypeReferences recursively collects all type references from a node.
func (g *GutsGenerator) collectTypeReferences(node bindings.Node, refs map[string]struct{}) {
	switch n := node.(type) {
	case *bindings.Alias:
		// Type alias: type Foo = Bar
		g.collectExpressionTypeReferences(n.Type, refs)

	case *bindings.Interface:
		// Interface: interface Foo { bar: Bar }
		for _, field := range n.Fields {
			g.collectExpressionTypeReferences(field.Type, refs)
		}
	}
}

// collectExpressionTypeReferences recursively collects all type names referenced by an expression.
// Handles unions, intersections, arrays, type literals, and generic arguments.
func (g *GutsGenerator) collectExpressionTypeReferences(expr bindings.ExpressionType, refs map[string]struct{}) {
	if expr == nil {
		return
	}

	switch e := expr.(type) {
	case *bindings.ReferenceType:
		// Direct reference to another type
		refs[e.Name.String()] = struct{}{}

		// Check generic arguments
		for _, arg := range e.Arguments {
			g.collectExpressionTypeReferences(arg, refs)
		}

	case *bindings.UnionType:
		// Union: A | B
		for _, member := range e.Types {
			g.collectExpressionTypeReferences(member, refs)
		}

	case *bindings.TypeIntersection:
		// Intersection: A & B
		for _, member := range e.Types {
			g.collectExpressionTypeReferences(member, refs)
		}

	case *bindings.ArrayLiteralType:
		// Array: T[]
		for _, elem := range e.Elements {
			g.collectExpressionTypeReferences(elem, refs)
		}

	case *bindings.TypeLiteralNode:
		// Inline object: { foo: Bar }
		for _, member := range e.Members {
			g.collectExpressionTypeReferences(member.Type, refs)
		}

	// Primitive types - no references to collect
	case *bindings.LiteralKeyword:
	case *bindings.LiteralType:
	}
}
