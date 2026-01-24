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

// OpenAPICollector handles TypeScript AST parsing and metadata extraction from Go types.
// It walks the TypeScript AST to extract comprehensive type information in a single pass.
type OpenAPICollector struct {
	tsParser *guts.Typescript
	vm       *bindings.Bindings
	l        *slog.Logger

	types    map[string]*TypeInfo   // Extracted type information
	routes   map[string]*PathRoutes // Registered routes, keyed by path
	dbSchema string                 // Database schema SQL

	docsFilePath        string // Path to write documentation JSON file
	openAPISpecFilePath string // Path to write OpenAPI YAML file

	apiInfo APIInfo
}

type OpenAPICollectorOptions struct {
	GoTypesDirPath               string // Path to Go types file for parsing
	DocsFileOutputPath           string // Path for generated API docs JSON file
	DatabaseSchemaFileOutputPath string // Path for generated DB schema SQL file
	OpenAPISpecOutputPath        string // Path for generated OpenAPI YAML file
	APIInfo                      APIInfo
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
		l:                   l,
		types:               make(map[string]*TypeInfo),
		routes:              make(map[string]*PathRoutes),
		docsFilePath:        opts.DocsFileOutputPath,
		openAPISpecFilePath: opts.OpenAPISpecOutputPath,
		apiInfo:             opts.APIInfo,
	}

	dbSchema, err := gutsGenerator.GenerateDatabaseSchema(l, opts.DatabaseSchemaFileOutputPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get database schema: %w", err)
	}
	l.Info("Database schema generated", slog.String("file", opts.DatabaseSchemaFileOutputPath))
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

// Generate generates both the OpenAPI spec YAML and the docs JSON file
func (g *OpenAPICollector) Generate() error {
	// Compute type relationships
	g.computeTypeRelationships()

	// Write OpenAPI spec
	if err := g.writeSpecYAML(g.openAPISpecFilePath); err != nil {
		return fmt.Errorf("failed to write OpenAPI spec: %w", err)
	}
	g.l.Info("OpenAPI spec written", slog.String("file", g.openAPISpecFilePath))

	// Write docs JSON
	if err := g.writeDocsJSON(); err != nil {
		return fmt.Errorf("failed to write docs JSON: %w", err)
	}

	return nil
}

func (g *OpenAPICollector) RegisterRoute(route *RouteInfo) {
	// Get or create PathRoutes for this path
	pathRoutes, exists := g.routes[route.Path]
	if !exists {
		pathRoutes = &PathRoutes{Routes: make(map[string]*RouteInfo)}
		g.routes[route.Path] = pathRoutes
	}

	// Add route under its HTTP method
	pathRoutes.Routes[route.Method] = route
}

// timeTypeOverride returns a TypeOverride for time.Time
func timeTypeOverride() bindings.ExpressionType {
	return &ExternalType{
		GoType:         "time.Time",
		TypeScriptType: "string",
		OpenAPIFormat:  "date-time",
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
	goParser.IncludeCustomDeclaration(map[string]guts.TypeOverride{
		"time.Time": timeTypeOverride,
	})

	if _, err := os.Stat(goTypesDirPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("go types dir path %s does not exist", goTypesDirPath)
	}

	if err := goParser.IncludeGenerate(goTypesDirPath); err != nil {
		return nil, fmt.Errorf("failed to include go types dir for parsing: %w", err)
	}

	var errs []error
	for _, pkg := range goParser.Pkgs {
		for _, e := range pkg.Errors {
			errs = append(errs, fmt.Errorf("failed to parse go types in %s: %w", pkg.PkgPath, e))
		}
	}

	if len(errs) > 0 {
		return nil, errors.Join(errs...)
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
	var errs []error

	g.tsParser.ForEach(func(name string, node bindings.Node) {
		typeInfo, err := g.extractTypeFromNode(name, node)
		if err != nil {
			errs = append(errs, fmt.Errorf("failed to extract type %s: %w", name, err))
			return
		}
		g.types[name] = typeInfo
	})

	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
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

	// Assume it's a simple alias by default
	typeInfo := &TypeInfo{
		Name:        name,
		Kind:        TypeKindAlias,
		Description: desc,
		UsedBy:      []UsageInfo{},
	}

	switch alias := alias.Type.(type) {
	case *bindings.UnionType:
		// Check if it's a union type (enum)
		g.l.Debug("Analyzing alias type kind", slog.String("type", name), slog.String("kind", "union"))
		// Check if it's a string enum
		enumVals := g.extractLiteralsFromUnion(alias)
		if len(enumVals) > 0 {
			typeInfo.Kind = TypeKindStringEnum
			typeInfo.EnumValues = enumVals
		}
	case *bindings.TypeLiteralNode:
		g.l.Debug("Analyzing alias type kind", slog.String("type", name), slog.String("kind", "object"))
		// Check if it's an alias to a type literal (object)
		typeInfo.Fields = make([]FieldInfo, 0, len(alias.Members))
		typeInfo.Kind = TypeKindObject

		for _, member := range alias.Members {
			_, err := g.serializeExpressionType(member.Type)
			if err != nil {
				return nil, fmt.Errorf("failed to serialize field type for %s: %w", name, err)
			}

			// Extract structured type information
			fieldInfo, err := g.analyzeFieldType(member.Type)
			if err != nil {
				return nil, fmt.Errorf("failed to analyze field type for %s.%s: %w", name, member.Name, err)
			}
			fieldInfo.Required = !member.QuestionToken
			fieldInfo.EnumValues = g.extractEnumValues(member.Type)

			field := FieldInfo{
				Name:        member.Name,
				DisplayType: generateDisplayType(fieldInfo),
				TypeInfo:    fieldInfo,
				Description: g.extractComments(member.SupportComments),
			}

			// Check if it's an external type and capture metadata
			if ext, ok := member.Type.(*ExternalType); ok {
				field.GoType = ext.GoType
			}

			typeInfo.Fields = append(typeInfo.Fields, field)
		}

		// Collect references by walking raw AST (captures generics, intersections, tuples, inline objects)
		typeInfo.References = g.collectReferencesFromMembers(alias.Members)
	}

	return typeInfo, nil
}

// extractInterfaceType extracts type information from an interface node
func (g *OpenAPICollector) extractInterfaceType(name string, iface *bindings.Interface) (*TypeInfo, error) {
	desc := g.extractComments(iface.SupportComments)
	fields := make([]FieldInfo, 0, len(iface.Fields))

	for _, field := range iface.Fields {
		_, err := g.serializeExpressionType(field.Type)
		if err != nil {
			return nil, fmt.Errorf("failed to serialize field type for %s: %w", name, err)
		}

		// Extract structured type information
		typeInfo, err := g.analyzeFieldType(field.Type)
		if err != nil {
			return nil, fmt.Errorf("failed to analyze field type for %s.%s: %w", name, field.Name, err)
		}
		typeInfo.Required = !field.QuestionToken
		typeInfo.EnumValues = g.extractEnumValues(field.Type)

		fieldInfo := FieldInfo{
			Name:        field.Name,
			DisplayType: generateDisplayType(typeInfo),
			TypeInfo:    typeInfo,
			Description: g.extractComments(field.SupportComments),
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
		References:  g.collectReferencesFromMembers(iface.Fields),
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

func (g *OpenAPICollector) getDocumentation() *APIDocumentation {

	return &APIDocumentation{
		Types:          g.types,
		Routes:         g.routes,
		DatabaseSchema: g.dbSchema,
		Info:           g.apiInfo,
	}
}

// collectReferencesFromMembers collects direct type references by walking the raw AST
// Only collects named types - rejects generics, inline objects, tuples, and intersections
func (g *OpenAPICollector) collectReferencesFromMembers(members []*bindings.PropertySignature) []string {
	refs := make(map[string]struct{})
	for _, member := range members {
		g.collectExpressionTypeReferences(member.Type, refs)
	}

	// Convert to sorted slice
	refList := make([]string, 0, len(refs))
	for ref := range refs {
		refList = append(refList, ref)
	}
	sort.Strings(refList)
	return refList
}

// computeTypeRelationships computes ReferencedBy and UsedBy for all types
// (References are already computed during type extraction)
func (g *OpenAPICollector) computeTypeRelationships() {
	// First pass: build ReferencedBy from References
	for typeName, typeInfo := range g.types {
		for _, ref := range typeInfo.References {
			if refType, exists := g.types[ref]; exists {
				if refType.ReferencedBy == nil {
					refType.ReferencedBy = []string{}
				}
				refType.ReferencedBy = append(refType.ReferencedBy, typeName)
			}
		}
	}

	// Sort ReferencedBy lists
	for _, typeInfo := range g.types {
		sort.Strings(typeInfo.ReferencedBy)
	}

	// Second pass: compute UsedBy from routes
	for _, pathRoutes := range g.routes {
		for _, route := range pathRoutes.Routes {
			// Track request type usage
			if route.Request != nil && route.Request.Type != "" {
				if typeInfo, exists := g.types[route.Request.Type]; exists {
					typeInfo.UsedBy = append(typeInfo.UsedBy, UsageInfo{
						OperationID: route.OperationID,
						Role:        "request",
					})
				}
			}

			// Track response type usage
			for _, resp := range route.Responses {
				if resp.Type != "" {
					if typeInfo, exists := g.types[resp.Type]; exists {
						typeInfo.UsedBy = append(typeInfo.UsedBy, UsageInfo{
							OperationID: route.OperationID,
							Role:        "response",
						})
					}
				}
			}

			// Track parameter type usage
			for _, param := range route.Parameters {
				if param.Type != "" {
					if typeInfo, exists := g.types[param.Type]; exists {
						typeInfo.UsedBy = append(typeInfo.UsedBy, UsageInfo{
							OperationID: route.OperationID,
							Role:        "parameter",
						})
					}
				}
			}
		}
	}
}

// collectExpressionTypeReferences collects direct type references from an expression
// Only collects named types - rejects generics and inline objects (which should error during analysis)
func (g *OpenAPICollector) collectExpressionTypeReferences(expr bindings.ExpressionType, refs map[string]struct{}) {
	if expr == nil {
		return
	}

	switch e := expr.(type) {
	case *bindings.ReferenceType:
		// Direct reference to a named type
		refs[e.Name.String()] = struct{}{}

	case *bindings.UnionType:
		// Union: A | B (used for enums and nullable types)
		for _, member := range e.Types {
			g.collectExpressionTypeReferences(member, refs)
		}

	case *bindings.ArrayType:
		// Array: T[] - recurse to get the element type
		g.collectExpressionTypeReferences(e.Node, refs)

	// Primitive types and external types - no references to collect
	case *bindings.LiteralKeyword:
	case *bindings.LiteralType:
	case *ExternalType:

	case *bindings.TypeLiteralNode:
		panic("inline object found during reference collection - should have been rejected earlier")
	case *bindings.TypeIntersection:
		panic("intersection type found during reference collection - should have been rejected earlier")
	}
}

// generateOpenAPISpec generates a complete OpenAPI specification from all collected metadata
func (g *OpenAPICollector) generateOpenAPISpec() (*openapi3.T, error) {
	doc := g.getDocumentation()

	spec, err := generateOpenAPISpec(doc)
	if err != nil {
		return nil, err
	}

	// Set API metadata
	spec.Info.Title = g.apiInfo.Title
	spec.Info.Version = g.apiInfo.Version
	spec.Info.Description = g.apiInfo.Description

	for _, server := range g.apiInfo.Servers {
		spec.Servers = append(spec.Servers, &openapi3.Server{
			URL:         server.URL,
			Description: server.Description,
		})
	}

	return spec, nil
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

	switch e := expr.(type) {
	case *ExternalType:
		// Skip external types
		return nil
	case *bindings.UnionType:
		// Check if it's a direct union type
		return g.extractLiteralsFromUnion(e)
	}

	// Check if it's a reference to another type
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
			// Is this expected? Or should we error?
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
// Note: Union literals don't have comments, so Description will be empty. // FIXME: Check if we can improve this upstream.
// For enums with comments, use extractEnumMemberValues instead.
func (g *OpenAPICollector) extractLiteralsFromUnion(union *bindings.UnionType) []EnumValue {
	var values []EnumValue

	for _, member := range union.Types {
		lit, ok := member.(*bindings.LiteralType)
		if !ok {
			// TODO: Is this expected? Or should we error?
			g.l.Warn("Skipping non-literal union member", slog.String("memberType", fmt.Sprintf("%T", member)))
			continue
		}

		// Check if the literal value is a string
		strVal, ok := lit.Value.(string)
		if !ok {
			g.l.Warn("Skipping non-string union member", slog.String("memberType", fmt.Sprintf("%T", member)))
			continue
		}

		values = append(values, EnumValue{Value: strVal})
	}

	return values
}

// generateDisplayType creates a human-readable type string from FieldType
func generateDisplayType(ft FieldType) string {
	switch ft.Kind {
	case FieldKindPrimitive:
		if ft.Nullable {
			return ft.Type + " | null"
		}
		return ft.Type

	case FieldKindArray:
		if ft.ItemsType != nil {
			itemDisplay := generateDisplayType(*ft.ItemsType)
			return itemDisplay + "[]"
		}
		return "array"

	case FieldKindReference, FieldKindEnum:
		if ft.Nullable {
			return ft.Type + " | null"
		}
		return ft.Type

	case FieldKindUnion:
		if len(ft.UnionTypes) > 0 {
			types := make([]string, len(ft.UnionTypes))
			for i, t := range ft.UnionTypes {
				types[i] = generateDisplayType(t)
			}
			return strings.Join(types, " | ")
		}
		return "union"

	case FieldKindObject:
		return "object"

	default:
		return "any"
	}
}

func (g *OpenAPICollector) analyzeFieldType(expr bindings.ExpressionType) (FieldType, error) {
	if expr == nil {
		return FieldType{Kind: FieldKindUnknown, Type: "unknown"}, fmt.Errorf("cannot analyze nil expression type")
	}

	switch e := expr.(type) {
	case *ExternalType:
		// External types (e.g., time.Time)
		return FieldType{
			Kind:   FieldKindPrimitive,
			Type:   e.TypeScriptType,
			Format: e.OpenAPIFormat,
		}, nil

	case *bindings.ArrayType:
		// Regular arrays (e.g., User[])
		itemType, err := g.analyzeFieldType(e.Node)
		if err != nil {
			return FieldType{}, fmt.Errorf("failed to analyze array type: %w", err)
		}
		return FieldType{
			Kind:      FieldKindArray,
			Type:      "array",
			ItemsType: &itemType,
		}, nil

	case *bindings.ReferenceType:
		// Reference types
		refName := e.Name.String()

		// Reject generic types (we don't support them in Go->TS serialization)
		if len(e.Arguments) > 0 {
			return FieldType{}, fmt.Errorf("generic types are not supported: %s<%d type arguments>", refName, len(e.Arguments))
		}

		// Check if it's an enum
		if refNode, exists := g.tsParser.Node(refName); exists {
			if alias, ok := refNode.(*bindings.Alias); ok {
				if _, ok := alias.Type.(*bindings.UnionType); ok {
					return FieldType{Kind: FieldKindEnum, Type: refName}, nil
				}
			}
		}
		return FieldType{Kind: FieldKindReference, Type: refName}, nil

	case *bindings.UnionType:
		// Union types (could be enums or nullable types)

		// Check if it's a string enum
		if len(g.extractLiteralsFromUnion(e)) > 0 {
			return FieldType{Kind: FieldKindEnum, Type: "string"}, nil
		}

		// Check for nullable pattern: T | null or null | T or T | undefined
		if len(e.Types) == 2 {
			var nonNullType bindings.ExpressionType
			hasNull := false

			for _, t := range e.Types {
				// Serialize the type to check if it's null/undefined
				serialized, err := g.serializeExpressionType(t)
				if err == nil && (serialized == "null" || serialized == "undefined") {
					hasNull = true
					continue
				}
				nonNullType = t
			}

			// If we found exactly one null and one non-null type, it's nullable
			if hasNull && nonNullType != nil {
				result, err := g.analyzeFieldType(nonNullType)
				if err != nil {
					return FieldType{}, fmt.Errorf("failed to analyze nullable type: %w", err)
				}
				result.Nullable = true
				return result, nil
			}
		}

		// For complex unions, analyze each member
		var unionMembers []FieldType
		for _, t := range e.Types {
			memberType, err := g.analyzeFieldType(t)
			if err != nil {
				return FieldType{}, fmt.Errorf("failed to analyze union member: %w", err)
			}
			unionMembers = append(unionMembers, memberType)
		}
		return FieldType{
			Kind:       FieldKindUnion,
			Type:       "union",
			UnionTypes: unionMembers,
		}, nil

	case *bindings.LiteralKeyword:
		// Literal keywords (primitives)
		switch *e {
		case bindings.KeywordString:
			return FieldType{Kind: FieldKindPrimitive, Type: "string"}, nil
		case bindings.KeywordNumber:
			return FieldType{Kind: FieldKindPrimitive, Type: "number"}, nil
		case bindings.KeywordBoolean:
			return FieldType{Kind: FieldKindPrimitive, Type: "boolean"}, nil
		default:
			return FieldType{Kind: FieldKindPrimitive, Type: string(*e)}, nil
		}
	case *bindings.ArrayLiteralType:
		// Tuple types are not supported in Go->TS serialization
		return FieldType{}, fmt.Errorf("tuple types are not supported - Go does not have tuple types")

	case *bindings.TypeLiteralNode:
		// Inline objects are not allowed - require named types
		return FieldType{}, fmt.Errorf("inline object types are not supported - please create a named type instead")

	case *bindings.TypeIntersection:
		// Type intersections are not supported in Go->TS serialization
		return FieldType{}, fmt.Errorf("intersection types are not supported")

	default:
		return FieldType{}, fmt.Errorf("unsupported expression type: %T", expr)
	}
}

// writeSpecYAML writes the OpenAPI specification to a YAML file
func (g *OpenAPICollector) writeSpecYAML(filename string) error {
	spec, err := g.generateOpenAPISpec()
	if err != nil {
		return fmt.Errorf("failed to generate spec: %w", err)
	}
	yamlData, err := yaml.Marshal(spec)
	if err != nil {
		return err
	}
	return os.WriteFile(filename, yamlData, 0644)
}

// writeDocsJSON writes the complete API documentation to a JSON file
func (g *OpenAPICollector) writeDocsJSON() error {
	if g.docsFilePath == "" {
		return nil // Skip if no path configured
	}

	doc := g.getDocumentation()

	// Use GenerateAPIDocs for sorted, deterministic output
	if err := GenerateAPIDocs(doc, g.docsFilePath); err != nil {
		return fmt.Errorf("failed to write docs JSON: %w", err)
	}

	g.l.Info("API documentation written", slog.String("file", g.docsFilePath))
	return nil
}
