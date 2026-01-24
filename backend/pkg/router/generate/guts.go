package generate

// This file (guts.go) handles TypeScript AST parsing and metadata extraction
// using the github.com/coder/guts library to parse Go structs and generate
// TypeScript type definitions with full metadata.

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"sort"
	"strings"

	"github.com/coder/guts"
	"github.com/coder/guts/bindings"
	"github.com/coder/guts/config"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/oasdiff/yaml"
)

// ExternalType represents an external Go type with metadata for OpenAPI generation
type ExternalType struct {
	bindings.LiteralKeyword
	GoType         string // Original Go type (e.g., "time.Time")
	TypeScriptType string // TypeScript representation (e.g., "string")
	OpenAPIFormat  string // OpenAPI format (e.g., "date-time")
}

// OpenAPICollector handles TypeScript AST parsing and metadata extraction from Go types.
// It walks the TypeScript AST to extract comprehensive type information in a single pass.
type OpenAPICollector struct {
	tsParser *guts.Typescript
	vm       *bindings.Bindings
	l        *slog.Logger
	types    map[string]*TypeInfo   // Extracted type information
	routes   map[string]*PathRoutes // Registered routes, keyed by path
	dbSchema string                 // Database schema SQL
	servers  []*openapi3.Server     // OpenAPI servers
}

type OpenAPICollectorOptions struct {
	GoTypesDirPath               string // Path to Go types file for parsing
	DocsFileOutputPath           string // Path for generated API docs JSON file
	DatabaseSchemaFileOutputPath string // Path for generated DB schema SQL file
}

// NewOpenAPICollector parses the Go types directory and generates a TypeScript AST for metadata extraction.
func NewOpenAPICollector(l *slog.Logger, opts OpenAPICollectorOptions) (*OpenAPICollector, error) {
	var err error

	l = l.With(slog.String("component", "openapi-collector"))

	if opts.GoTypesDirPath == "" {
		return nil, errors.New("go types dir path is required")
	}
	if opts.DatabaseSchemaFileOutputPath == "" {
		return nil, errors.New("database schema file path is required")
	}
	if opts.DocsFileOutputPath == "" {
		return nil, errors.New("docs file path is required")
	}

	// Prepend "./" to the path if it's not already there, this is
	// to make the package parser to know that it's a local package
	// and not a standard library package
	goTypesDirPath := strings.TrimPrefix(opts.GoTypesDirPath, "./")
	goTypesDirPath = strings.TrimPrefix(goTypesDirPath, "/")
	goTypesDirPath = "./" + goTypesDirPath

	l.Debug("Creating guts generator", slog.String("goTypesDirPath", goTypesDirPath))

	gutsGenerator := &OpenAPICollector{
		l:      l,
		types:  make(map[string]*TypeInfo),
		routes: make(map[string]*PathRoutes),
	}

	dbSchema, err := gutsGenerator.GenerateDatabaseSchema(l, opts.DatabaseSchemaFileOutputPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get database schema: %w", err)
	}
	gutsGenerator.dbSchema = dbSchema

	gutsGenerator.vm, err = bindings.New()
	if err != nil {
		return nil, fmt.Errorf("failed to create bindings VM: %w", err)
	}

	gutsGenerator.tsParser, err = newTypescriptASTFromGoTypesDir(l, goTypesDirPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create TypeScript AST from go types dir: %w", err)
	}

	// Walk the AST and extract all type information in one pass
	if err := gutsGenerator.extractAllTypes(); err != nil {
		return nil, fmt.Errorf("failed to extract types: %w", err)
	}

	l.Info("OpenAPI collector created successfully", slog.Int("types", len(gutsGenerator.types)))

	return gutsGenerator, nil
}

// customTypeOverrides returns custom type mappings for external Go types
func customTypeOverrides() map[string]guts.TypeOverride {
	return map[string]guts.TypeOverride{
		"time.Time": func() bindings.ExpressionType {
			return &ExternalType{
				GoType:         "time.Time",
				TypeScriptType: "string",
				OpenAPIFormat:  "date-time",
			}
		},
	}
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
	goParser.IncludeCustomDeclaration(customTypeOverrides())

	if _, err := os.Stat(goTypesDirPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("go types dir path %s does not exist", goTypesDirPath)
	}

	if err := goParser.IncludeGenerate(goTypesDirPath); err != nil {
		return nil, fmt.Errorf("failed to include go types dir for parsing: %w", err)
	}

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
		return nil, errors.New("failed to parse go types")
	}

	l.Debug("Generating TypeScript AST from Go types")

	ts, err := goParser.ToTypescript()
	if err != nil {
		return nil, fmt.Errorf("failed to generate TypeScript AST: %w", err)
	}

	ts.ApplyMutations(
		// config.EnumAsTypes,
		config.EnumLists,
		config.ExportTypes,
		config.InterfaceToType,
		config.SimplifyOptional,
	)

	l.Debug("TypeScript AST generated successfully")

	return ts, nil
}

// extractAllTypes walks the TypeScript AST and extracts all type information in one pass
func (g *OpenAPICollector) extractAllTypes() error {
	var err error
	g.tsParser.ForEach(func(name string, node bindings.Node) {
		typeInfo, err := g.extractTypeFromNode(name, node)
		if err != nil {
			err = fmt.Errorf("failed to extract type %s: %w", name, err)
			return
		}
		g.types[name] = typeInfo
	})
	return err
}

// extractTypeFromNode extracts TypeInfo from a TypeScript AST node
func (g *OpenAPICollector) extractTypeFromNode(name string, node bindings.Node) (*TypeInfo, error) {
	switch n := node.(type) {
	case *bindings.Alias:
		return g.extractAliasType(name, n)
	case *bindings.Interface:
		return g.extractInterfaceType(name, n)
	case *bindings.Enum:
		return g.extractEnumType(name, n)
	default:
		return nil, fmt.Errorf("unsupported node type: %T", node)
	}
}

// extractAliasType extracts type information from an alias node
func (g *OpenAPICollector) extractAliasType(name string, alias *bindings.Alias) (*TypeInfo, error) {
	desc := g.extractComments(alias.SupportComments)

	// Check if it's a union type (enum)
	if union, ok := alias.Type.(*bindings.UnionType); ok {
		// Check if it's a string enum
		enumVals := g.extractLiteralsFromUnion(union)
		if len(enumVals) > 0 {
			return &TypeInfo{
				Name:        name,
				Kind:        TypeKindStringEnum,
				Description: desc,
				EnumValues:  enumVals,
				UsedBy:      []UsageInfo{},
			}, nil
		}
	}

	// Check if it's an alias to a type literal (object)
	if typeLiteral, ok := alias.Type.(*bindings.TypeLiteralNode); ok {
		fields := make([]FieldInfo, 0, len(typeLiteral.Members))

		for _, member := range typeLiteral.Members {
			fieldType, err := g.serializeExpressionType(member.Type)
			if err != nil {
				g.l.Warn("Failed to serialize field type",
					slog.String("type", name),
					slog.String("field", member.Name),
					slog.String("error", err.Error()))
				continue
			}

			// Extract structured type information
			typeInfo := g.analyzeFieldType(member.Type)

			fieldInfo := FieldInfo{
				Name:        member.Name,
				Type:        fieldType,
				TypeInfo:    typeInfo,
				Description: g.extractComments(member.SupportComments),
				Required:    !member.QuestionToken,
				EnumValues:  g.extractEnumValues(member.Type),
			}

			// Check if it's an external type and capture metadata
			if ext, ok := member.Type.(*ExternalType); ok {
				fieldInfo.GoType = ext.GoType
			}

			fields = append(fields, fieldInfo)
		}

		return &TypeInfo{
			Name:        name,
			Kind:        TypeKindObject,
			Description: desc,
			Fields:      fields,
			UsedBy:      []UsageInfo{},
		}, nil
	}

	// Otherwise it's a type alias
	return &TypeInfo{
		Name:        name,
		Kind:        TypeKindAlias,
		Description: desc,
		UsedBy:      []UsageInfo{},
	}, nil
}

// extractInterfaceType extracts type information from an interface node
func (g *OpenAPICollector) extractInterfaceType(name string, iface *bindings.Interface) (*TypeInfo, error) {
	desc := g.extractComments(iface.SupportComments)
	fields := make([]FieldInfo, 0, len(iface.Fields))

	for _, field := range iface.Fields {
		fieldType, err := g.serializeExpressionType(field.Type)
		if err != nil {
			g.l.Warn("Failed to serialize field type",
				slog.String("type", name),
				slog.String("field", field.Name),
				slog.String("error", err.Error()))
			continue
		}

		// Extract structured type information
		typeInfo := g.analyzeFieldType(field.Type)

		fieldInfo := FieldInfo{
			Name:        field.Name,
			Type:        fieldType,
			TypeInfo:    typeInfo,
			Description: g.extractComments(field.SupportComments),
			Required:    !field.QuestionToken,
			EnumValues:  g.extractEnumValues(field.Type),
		}

		// Check if it's an external type and capture metadata
		if ext, ok := field.Type.(*ExternalType); ok {
			fieldInfo.GoType = ext.GoType
		}

		fields = append(fields, fieldInfo)
	}

	return &TypeInfo{
		Name:        name,
		Kind:        TypeKindObject,
		Description: desc,
		Fields:      fields,
		UsedBy:      []UsageInfo{},
	}, nil
}

// extractEnumType extracts type information from an enum node
func (g *OpenAPICollector) extractEnumType(name string, enum *bindings.Enum) (*TypeInfo, error) {
	// Note: bindings.Enum doesn't have SupportComments, so description will be empty
	enumVals := g.extractEnumMemberValues(enum)

	return &TypeInfo{
		Name:        name,
		Kind:        TypeKindStringEnum,
		Description: "",
		EnumValues:  enumVals,
		UsedBy:      []UsageInfo{},
	}, nil
}

// Get methods for accessing extracted data
func (g *OpenAPICollector) GetType(name string) (*TypeInfo, bool) {
	t, ok := g.types[name]
	return t, ok
}

func (g *OpenAPICollector) GetAllTypes() map[string]*TypeInfo {
	return g.types
}

func (g *OpenAPICollector) RegisterRoute(route *RouteInfo) {
	// Get or create PathRoutes for this path
	pathRoutes, exists := g.routes[route.Path]
	if !exists {
		pathRoutes = &PathRoutes{
			Path:   route.Path,
			Routes: make(map[string]*RouteInfo),
		}
		g.routes[route.Path] = pathRoutes
	}

	// Add route under its HTTP method
	pathRoutes.Routes[route.Method] = route

	// TODO: Track usage in types
}

func (g *OpenAPICollector) GetDocumentation() *APIDocumentation {
	return &APIDocumentation{
		Types:          g.types,
		Routes:         g.routes,
		DatabaseSchema: g.dbSchema,
	}
}

// GenerateOpenAPISpec generates a complete OpenAPI specification from all collected metadata
func (g *OpenAPICollector) GenerateOpenAPISpec() (*openapi3.T, error) {
	doc := g.GetDocumentation()
	spec, err := generateOpenAPISpec(doc)
	if err != nil {
		return nil, err
	}

	// Add servers if any were configured
	if len(g.servers) > 0 {
		spec.Servers = g.servers
	}

	return spec, nil
}

// WriteTypescriptASTToFile serializes and writes TypeScript type definitions to a file.
func (g *OpenAPICollector) WriteTypescriptASTToFile(ts *guts.Typescript, filePath string) error {
	g.l.Debug("Serializing TypeScript AST", slog.String("file", filePath))

	str, err := ts.Serialize()
	if err != nil {
		return fmt.Errorf("failed to serialize TypeScript AST: %w", err)
	}

	err = os.WriteFile(filePath, []byte(str), 0600)
	if err != nil {
		return fmt.Errorf("failed to write TypeScript AST to file: %w", err)
	}

	g.l.Info("TypeScript types written", slog.String("file", filePath))

	return nil
}

// SerializeNode converts a type name to its TypeScript string representation.
func (g *OpenAPICollector) SerializeNode(name string) (string, error) {
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
func (g *OpenAPICollector) ExtractReferences(name string) ([]string, error) {
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
func (g *OpenAPICollector) ExtractFields(name string) ([]FieldMetadata, error) {
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
func (g *OpenAPICollector) ExtractTypeDescription(name string) (string, error) {
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
func (g *OpenAPICollector) ExtractTypeKind(name string) (string, error) {
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
func (g *OpenAPICollector) ExtractTypeEnumValues(name string) ([]EnumValue, error) {
	node, exists := g.tsParser.Node(name)
	if !exists {
		return nil, fmt.Errorf("node %s not found in TypeScript AST", name)
	}

	switch n := node.(type) {
	case *bindings.Enum:
		// Extract enum values with comments from enum members
		values := g.extractEnumMemberValues(n)
		if len(values) > 0 {
			g.l.Debug("Extracted enum values from enum", slog.String("type", name), slog.Int("count", len(values)))
		}

		return values, nil

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
//
//nolint:funlen
func (g *OpenAPICollector) getTypeKindFromExpression(expr bindings.ExpressionType) (string, error) {
	if expr == nil {
		return "", errors.New("expression type is nil")
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
func (g *OpenAPICollector) extractFieldsFromExpressionType(expr bindings.ExpressionType) []FieldMetadata {
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
func (g *OpenAPICollector) serializeExpressionType(expr bindings.ExpressionType) (string, error) {
	if expr == nil {
		return "", errors.New("expression type is nil")
	}

	// Handle our custom ExternalType
	if ext, ok := expr.(*ExternalType); ok {
		return ext.TypeScriptType, nil
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
func (g *OpenAPICollector) extractComments(sc bindings.SupportComments) string {
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
func (g *OpenAPICollector) extractEnumValues(expr bindings.ExpressionType) []EnumValue {
	if expr == nil {
		return nil
	}

	// Skip external types
	if _, ok := expr.(*ExternalType); ok {
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

// extractEnumMemberValues extracts string literal values with comments from enum members.
func (g *OpenAPICollector) extractEnumMemberValues(enum *bindings.Enum) []EnumValue {
	var values []EnumValue

	for _, member := range enum.Members {
		// Serialize the enum member value to get its string representation
		valueStr, err := g.serializeExpressionType(member.Value)
		if err != nil {
			g.l.Warn("Failed to serialize enum member value",
				slog.String("enum", enum.Name.String()),
				slog.String("member", member.Name),
				slog.String("error", err.Error()))

			continue
		}

		// Remove quotes from string literals
		valueStr = strings.Trim(valueStr, "\"'")

		values = append(values, EnumValue{
			Value:       valueStr,
			Description: g.extractComments(member.SupportComments),
		})
	}

	return values
}

// extractLiteralsFromUnion extracts string literal values from a union, ignoring other types.
// Note: Union literals don't have comments, so Description will be empty.
// For enums with comments, use extractEnumMemberValues instead.
func (g *OpenAPICollector) extractLiteralsFromUnion(union *bindings.UnionType) []EnumValue {
	var values []EnumValue

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

		values = append(values, EnumValue{Value: strVal})
	}

	return values
}

// collectTypeReferences recursively collects all type references from a node.
func (g *OpenAPICollector) collectTypeReferences(node bindings.Node, refs map[string]struct{}) {
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

// analyzeFieldType analyzes an expression type and returns structured type information
func (g *OpenAPICollector) analyzeFieldType(expr bindings.ExpressionType) FieldType {
	if expr == nil {
		return FieldType{Kind: FieldKindPrimitive, Type: "any"}
	}

	// Handle external types
	if ext, ok := expr.(*ExternalType); ok {
		return FieldType{
			Kind:   FieldKindPrimitive,
			Type:   ext.TypeScriptType,
			Format: ext.OpenAPIFormat,
		}
	}

	// Handle regular arrays (e.g., User[])
	if arr, ok := expr.(*bindings.ArrayType); ok {
		itemType := g.analyzeFieldType(arr.Node)
		return FieldType{
			Kind:      FieldKindArray,
			Type:      "array",
			ItemsType: &itemType,
		}
	}

	// Handle tuple arrays (e.g., [string, number])
	if arr, ok := expr.(*bindings.ArrayLiteralType); ok {
		if len(arr.Elements) > 0 {
			itemType := g.analyzeFieldType(arr.Elements[0])
			return FieldType{
				Kind:      FieldKindArray,
				Type:      "array",
				ItemsType: &itemType,
			}
		}
		return FieldType{Kind: FieldKindArray, Type: "array"}
	}

	// Handle reference types
	if ref, ok := expr.(*bindings.ReferenceType); ok {
		refName := ref.Name.String()
		// Check if it's an enum
		if refNode, exists := g.tsParser.Node(refName); exists {
			if alias, ok := refNode.(*bindings.Alias); ok {
				if _, ok := alias.Type.(*bindings.UnionType); ok {
					return FieldType{Kind: FieldKindEnum, Type: refName}
				}
			}
		}
		return FieldType{Kind: FieldKindReference, Type: refName}
	}

	// Handle union types (could be enums)
	if union, ok := expr.(*bindings.UnionType); ok {
		// Check if it's a string enum
		if len(g.extractLiteralsFromUnion(union)) > 0 {
			return FieldType{Kind: FieldKindEnum, Type: "string"}
		}
		return FieldType{Kind: FieldKindUnion, Type: "union"}
	}

	// Handle literal keywords (primitives)
	if kw, ok := expr.(*bindings.LiteralKeyword); ok {
		switch *kw {
		case bindings.KeywordString:
			return FieldType{Kind: FieldKindPrimitive, Type: "string"}
		case bindings.KeywordNumber:
			return FieldType{Kind: FieldKindPrimitive, Type: "number"}
		case bindings.KeywordBoolean:
			return FieldType{Kind: FieldKindPrimitive, Type: "boolean"}
		default:
			return FieldType{Kind: FieldKindPrimitive, Type: string(*kw)}
		}
	}

	// Handle type literals (inline objects)
	if _, ok := expr.(*bindings.TypeLiteralNode); ok {
		return FieldType{Kind: FieldKindObject, Type: "object"}
	}

	// Default
	return FieldType{Kind: FieldKindUnknown, Type: "any"}
}

// collectExpressionTypeReferences recursively collects all type names referenced by an expression.
// Handles unions, intersections, arrays, type literals, and generic arguments.
func (g *OpenAPICollector) collectExpressionTypeReferences(expr bindings.ExpressionType, refs map[string]struct{}) {
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

// Spec generates and returns the OpenAPI specification
func (g *OpenAPICollector) Spec() (*openapi3.T, error) {
	return g.GenerateOpenAPISpec()
}

// WriteSpecYAML writes the OpenAPI specification to a YAML file
func (g *OpenAPICollector) WriteSpecYAML(filename string) error {
	spec, err := g.Spec()
	if err != nil {
		return fmt.Errorf("failed to generate spec: %w", err)
	}
	yamlData, err := yaml.Marshal(spec)
	if err != nil {
		return err
	}
	return os.WriteFile(filename, yamlData, 0644)
}

// WithServer adds a server to the OpenAPI spec
func (g *OpenAPICollector) WithServer(url, description string) {
	g.servers = append(g.servers, &openapi3.Server{
		URL:         url,
		Description: description,
	})
}
