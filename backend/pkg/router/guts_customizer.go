package router

import (
	"fmt"
	"log/slog"
	"os"
	"reflect"
	"strings"

	"github.com/coder/guts"
	"github.com/coder/guts/bindings"
	"github.com/coder/guts/config"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3gen"
)

// GutsSchemaCustomizer creates an OpenAPI schema customizer that uses guts
// to extract metadata from Go types (comments, enum values, required fields, etc.)
type GutsSchemaCustomizer struct {
	tsParser   *guts.Typescript
	vm         *bindings.Bindings
	l          *slog.Logger
	components openapi3.Schemas // Reference to OpenAPI components for creating enum refs
}

// NewGutsSchemaCustomizer creates a new guts-based schema customizer
// by parsing Go types from the specified directory
func NewGutsSchemaCustomizer(l *slog.Logger, goTypesDirPath string) (*GutsSchemaCustomizer, error) {
	l = l.With(slog.String("component", "guts-generator"))

	// Prepend "./" to the path if needed for local package resolution
	goTypesDirPath = strings.TrimPrefix(goTypesDirPath, "./")
	goTypesDirPath = strings.TrimPrefix(goTypesDirPath, "/")
	goTypesDirPath = "./" + goTypesDirPath

	// Create bindings VM for TypeScript serialization
	vm, err := bindings.New()
	if err != nil {
		return nil, fmt.Errorf("failed to create bindings VM: %w", err)
	}

	// Parse Go types and generate TypeScript AST
	tsParser, err := parseGoTypesToTS(l, goTypesDirPath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Go types: %w", err)
	}

	return &GutsSchemaCustomizer{
		tsParser:   tsParser,
		vm:         vm,
		l:          l,
		components: nil, // Will be set later via SetComponents
	}, nil
}

// SetComponents sets the OpenAPI components schemas map for creating enum references
func (g *GutsSchemaCustomizer) SetComponents(components openapi3.Schemas) {
	g.components = components
}

// parseGoTypesToTS parses Go types from a directory and generates TypeScript AST
func parseGoTypesToTS(l *slog.Logger, goTypesDirPath string) (*guts.Typescript, error) {
	l.Debug("Parsing Go types directory", slog.String("path", goTypesDirPath))

	goParser, err := guts.NewGolangParser()
	if err != nil {
		return nil, fmt.Errorf("failed to create guts parser: %w", err)
	}

	// Preserve comments so we can extract descriptions
	goParser.PreserveComments()

	if _, err := os.Stat(goTypesDirPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("go types dir path %s does not exist", goTypesDirPath)
	}

	if err := goParser.IncludeGenerate(goTypesDirPath); err != nil {
		return nil, fmt.Errorf("failed to include go types dir for parsing: %w", err)
	}

	// Check for parsing errors
	hasErrors := false
	for _, pkg := range goParser.Pkgs {
		if len(pkg.Errors) > 0 {
			hasErrors = true
		}
		for _, e := range pkg.Errors {
			l.Error("failed to parse go types", slog.String("pkg", pkg.PkgPath), slog.String("error", e.Error()))
		}
	}
	if hasErrors {
		return nil, fmt.Errorf("failed to parse go types: see logs for details")
	}

	l.Debug("Generating TypeScript AST from Go types")

	// Generate TypeScript AST
	ts, err := goParser.ToTypescript()
	if err != nil {
		return nil, fmt.Errorf("failed to generate TypeScript AST: %w", err)
	}

	// Apply transformations for better TypeScript output
	ts.ApplyMutations(
		config.EnumLists,
		config.ExportTypes,
		config.InterfaceToType,
	)
	l.Debug("TypeScript AST generated successfully")

	return ts, nil
}

// AsOpenAPIOption returns an openapi3gen.Option that can be used with the generator
func (g *GutsSchemaCustomizer) AsOpenAPIOption() openapi3gen.Option {
	return openapi3gen.SchemaCustomizer(func(name string, ft reflect.Type, tag reflect.StructTag, schema *openapi3.Schema) error {
		// Get the type name - handle pointers
		typeName := ft.Name()
		if ft.Kind() == reflect.Pointer {
			typeName = ft.Elem().Name()
		}

		// Skip unnamed types (primitives, slices, maps, etc.)
		if typeName == "" {
			return nil
		}

		// Try to find this type in the TypeScript AST
		node, exists := g.tsParser.Node(typeName)
		if !exists {
			// Type not found in AST - might be external or builtin
			return nil
		}

		// Extract and apply metadata
		if err := g.applyMetadata(typeName, node, schema); err != nil {
			return fmt.Errorf("failed to apply metadata for %s: %w", typeName, err)
		}

		return nil
	})
}

// applyMetadata extracts metadata from the TypeScript AST node and applies it to the OpenAPI schema
func (g *GutsSchemaCustomizer) applyMetadata(typeName string, node bindings.Node, schema *openapi3.Schema) error {
	// Check if this is an enum type first
	enumValues := g.extractEnumValues(typeName, node)
	isEnum := len(enumValues) > 0

	// Extract and apply description from comments
	desc := g.extractComments(node)
	schema.Description = desc
	if !isEnum && desc == "" {
		return fmt.Errorf("type %s has no documentation", typeName)
	}

	// Handle enums - extract values with descriptions
	if isEnum {
		g.applyEnumValues(typeName, schema, enumValues)
	}
	// Handle object types - extract field metadata and determine required fields
	if fields := g.extractFields(node); len(fields) > 0 {
		g.applyFieldMetadata(schema, fields)
	}

	return nil
}

// extractComments extracts documentation from a node's comments
func (g *GutsSchemaCustomizer) extractComments(node bindings.Node) string {
	var sc bindings.SupportComments

	switch n := node.(type) {
	case *bindings.Alias:
		sc = n.SupportComments
	case *bindings.Interface:
		sc = n.SupportComments

	default:
		return ""
	}

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

// enumValueWithDesc represents an enum value with its description
type enumValueWithDesc struct {
	Value       string
	Description string
}

// extractEnumValues extracts enum values with descriptions from a type
func (g *GutsSchemaCustomizer) extractEnumValues(typeName string, node bindings.Node) []enumValueWithDesc {
	switch n := node.(type) {
	case *bindings.Enum:
		return g.extractEnumMemberValues(n)
	case *bindings.Alias:
		return g.extractEnumFromAlias(n)
	default:
		return nil
	}
}

// extractEnumMemberValues extracts values from Go enum members (const declarations)
func (g *GutsSchemaCustomizer) extractEnumMemberValues(enum *bindings.Enum) []enumValueWithDesc {
	var values []enumValueWithDesc

	for _, member := range enum.Members {
		// Serialize the enum member value
		valueStr, err := g.serializeExpressionType(member.Value)
		if err != nil {
			continue
		}

		// Remove quotes from string literals
		valueStr = strings.Trim(valueStr, "\"'")

		values = append(values, enumValueWithDesc{
			Value:       valueStr,
			Description: g.extractMemberComments(member.SupportComments),
		})
	}

	return values
}

// extractEnumFromAlias extracts enum values from a type alias to a union of literals
func (g *GutsSchemaCustomizer) extractEnumFromAlias(alias *bindings.Alias) []enumValueWithDesc {
	union, ok := alias.Type.(*bindings.UnionType)
	if !ok {
		return nil
	}

	return g.extractLiteralsFromUnion(union)
}

// extractLiteralsFromUnion extracts string literal values from a union type
func (g *GutsSchemaCustomizer) extractLiteralsFromUnion(union *bindings.UnionType) []enumValueWithDesc {
	var values []enumValueWithDesc

	for _, member := range union.Types {
		lit, ok := member.(*bindings.LiteralType)
		if !ok {
			continue
		}

		// Only handle string literals
		strVal, ok := lit.Value.(string)
		if !ok {
			continue
		}

		values = append(values, enumValueWithDesc{
			Value:       strVal,
			Description: "", // Union literals don't have individual comments
		})
	}

	return values
}

// applyEnumValues applies enum values and descriptions to the schema
// If components are available, it creates a component schema and uses a $ref
func (g *GutsSchemaCustomizer) applyEnumValues(typeName string, schema *openapi3.Schema, values []enumValueWithDesc) {
	// Build description with enum value documentation
	enumDescription := schema.Description
	hasDescriptions := false
	for _, val := range values {
		if val.Description != "" {
			hasDescriptions = true
			break
		}
	}

	if hasDescriptions {
		if enumDescription != "" {
			enumDescription += "\n\nPossible values:\n"
		} else {
			enumDescription = "Possible values:\n"
		}

		for _, val := range values {
			if val.Description != "" {
				enumDescription += fmt.Sprintf("- `%s`: %s\n", val.Value, val.Description)
			} else {
				enumDescription += fmt.Sprintf("- `%s`\n", val.Value)
			}
		}
	}

	// If we have access to components, create a component schema and use a ref
	if g.components != nil && typeName != "" {
		ref, exists := g.components[typeName]
		if !exists {
			// Create the enum component schema
			enumSchema := &openapi3.Schema{
				Type:        &openapi3.Types{"string"},
				Description: enumDescription,
				Enum:        make([]any, len(values)),
			}

			for i, val := range values {
				enumSchema.Enum[i] = val.Value
			}
			enumSchemaRef := &openapi3.SchemaRef{Value: enumSchema}
			ref = enumSchemaRef
			g.components[typeName] = ref
		}

		// Replace current schema with a reference
		// We can't directly make this a $ref, so we use AllOf with a single reference
		// This is valid OpenAPI and effectively makes it a reference
		*schema = openapi3.Schema{
			Extensions: map[string]any{ExtensionReplaceWithRef: true},
			AllOf: []*openapi3.SchemaRef{
				{Ref: "#/components/schemas/" + typeName},
			},
		}
	} else {
		// Fallback: inline the enum values (old behavior)
		schema.Enum = make([]any, len(values))
		for i, val := range values {
			schema.Enum[i] = val.Value
		}
		schema.Description = enumDescription
	}
}

// fieldMetadata represents metadata for a single field
type fieldMetadata struct {
	Name        string
	Description string
	Optional    bool
}

// extractFields extracts field metadata from an object type
func (g *GutsSchemaCustomizer) extractFields(node bindings.Node) []fieldMetadata {
	switch n := node.(type) {
	case *bindings.Interface:
		return g.extractInterfaceFields(n)
	case *bindings.Alias:
		return g.extractAliasFields(n)
	default:
		return nil
	}
}

// extractInterfaceFields extracts fields from an interface
func (g *GutsSchemaCustomizer) extractInterfaceFields(iface *bindings.Interface) []fieldMetadata {
	var fields []fieldMetadata

	for _, prop := range iface.Fields {
		fields = append(fields, fieldMetadata{
			Name:        prop.Name,
			Description: g.extractMemberComments(prop.SupportComments),
			Optional:    prop.QuestionToken,
		})
	}

	return fields
}

// extractAliasFields extracts fields from a type alias to a type literal
func (g *GutsSchemaCustomizer) extractAliasFields(alias *bindings.Alias) []fieldMetadata {
	typeLiteral, ok := alias.Type.(*bindings.TypeLiteralNode)
	if !ok {
		return nil
	}

	var fields []fieldMetadata

	for _, member := range typeLiteral.Members {
		fields = append(fields, fieldMetadata{
			Name:        member.Name,
			Description: g.extractMemberComments(member.SupportComments),
			Optional:    member.QuestionToken,
		})
	}

	return fields
}

// applyFieldMetadata applies field descriptions and determines required fields
func (g *GutsSchemaCustomizer) applyFieldMetadata(schema *openapi3.Schema, fields []fieldMetadata) {
	// Apply field descriptions
	for _, field := range fields {
		if propSchema, exists := schema.Properties[field.Name]; exists && propSchema.Value != nil {
			if field.Description != "" {
				propSchema.Value.Description = field.Description
			}
		}
	}

	// Determine required fields based on optional flags
	var required []string
	for _, field := range fields {
		if !field.Optional {
			required = append(required, field.Name)
		}
	}

	if len(required) > 0 {
		schema.Required = required
	}
}

// extractMemberComments extracts comments from a member (field or enum value)
func (g *GutsSchemaCustomizer) extractMemberComments(sc bindings.SupportComments) string {
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

// serializeExpressionType converts an expression type to its TypeScript string representation
func (g *GutsSchemaCustomizer) serializeExpressionType(expr bindings.ExpressionType) (string, error) {
	if expr == nil {
		return "", fmt.Errorf("expression type is nil")
	}

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
