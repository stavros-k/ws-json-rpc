package generate

// This file (guts.go) handles TypeScript AST parsing and metadata extraction
// using the github.com/coder/guts library to parse Go structs and generate
// TypeScript type definitions with full metadata.

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"reflect"
	"sort"
	"strings"
	"ws-json-rpc/backend/pkg/utils"

	"github.com/coder/guts"
	"github.com/coder/guts/bindings"
	"github.com/coder/guts/config"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/oasdiff/yaml"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// OpenAPICollector handles TypeScript AST parsing and metadata extraction from Go types.
// It walks the TypeScript AST to extract comprehensive type information in a single pass.
type OpenAPICollector struct {
	tsParser *guts.Typescript
	vm       *bindings.Bindings
	l        *slog.Logger

	types         map[string]*TypeInfo                           // Extracted type information, keyed by type name
	httpOps       map[string]*RouteInfo                          // Registered HTTP operations, keyed by operationID
	database      Database                                       // Database schema and stats
	externalTypes map[*bindings.LiteralKeyword]*ExternalTypeInfo // External type metadata, keyed by keyword pointer

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
		httpOps:             make(map[string]*RouteInfo),
		externalTypes:       make(map[*bindings.LiteralKeyword]*ExternalTypeInfo),
		docsFilePath:        opts.DocsFileOutputPath,
		openAPISpecFilePath: opts.OpenAPISpecOutputPath,
		apiInfo:             opts.APIInfo,
	}

	dbSchema, err := gutsGenerator.GenerateDatabaseSchema(opts.DatabaseSchemaFileOutputPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get database schema: %w", err)
	}

	gutsGenerator.database.Schema = dbSchema

	dbStats, err := gutsGenerator.GetDatabaseStats(dbSchema)
	if err != nil {
		return nil, fmt.Errorf("failed to get database stats: %w", err)
	}

	gutsGenerator.database.TableCount = dbStats.TableCount

	gutsGenerator.vm, err = bindings.New()
	if err != nil {
		return nil, fmt.Errorf("failed to create bindings VM: %w", err)
	}

	gutsGenerator.tsParser, err = gutsGenerator.newTypescriptASTFromGoTypesDir(l, goTypesDirPath)
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

// Generate generates both the OpenAPI spec YAML and the docs JSON file.
func (g *OpenAPICollector) Generate() error {
	// Compute type relationships
	g.computeTypeRelationships()

	// Generate JSON schemas for all types
	if err := g.generateJSONSchemaRepresentations(); err != nil {
		return fmt.Errorf("failed to generate JSON schema representations: %w", err)
	}

	if err := g.generateTSRepresentations(); err != nil {
		return fmt.Errorf("failed to generate TypeScript representations: %w", err)
	}

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

// stringifyResponseExamples converts response examples to stringified JSON.
func stringifyResponseExamples(r ResponseInfo) ResponseInfo {
	r.ExamplesStringified = make(map[string]string)
	for name, example := range r.Examples {
		r.ExamplesStringified[name] = string(utils.MustToJSONIndent(example))
	}

	return r
}

// stringifyRequestExamples converts request examples to stringified JSON.
func stringifyRequestExamples(r *RequestInfo) *RequestInfo {
	r.ExamplesStringified = make(map[string]string)
	for name, example := range r.Examples {
		r.ExamplesStringified[name] = string(utils.MustToJSONIndent(example))
	}

	return r
}

func (g *OpenAPICollector) RegisterRoute(route *RouteInfo) error {
	// Validate operationID is unique
	if _, exists := g.httpOps[route.OperationID]; exists {
		return fmt.Errorf("duplicate operationID: %s", route.OperationID)
	}

	// Extract type names from zero values using reflection, and stringify examples
	if route.Request != nil {
		typeName, err := extractTypeNameFromValue(route.Request.TypeValue)
		if err != nil {
			return fmt.Errorf("failed to extract request type name: %w", err)
		}

		route.Request.TypeName = typeName

		if typeInfo, ok := g.types[typeName]; ok {
			if typeInfo.Representations.JSON == "" {
				typeInfo.Representations.JSON = string(utils.MustToJSONIndent(route.Request.TypeValue))
			}
		}

		route.Request = stringifyRequestExamples(route.Request)
	}

	for statusCode, response := range route.Responses {
		resp := response

		typeName, err := extractTypeNameFromValue(resp.TypeValue)
		if err != nil {
			return fmt.Errorf("failed to extract response type name: %w", err)
		}

		resp.TypeName = typeName

		if typeInfo, ok := g.types[typeName]; ok {
			if typeInfo.Representations.JSON == "" {
				typeInfo.Representations.JSON = string(utils.MustToJSONIndent(resp.TypeValue))
			}
		}

		route.Responses[statusCode] = stringifyResponseExamples(resp)
	}

	for i := range route.Parameters {
		typeName, err := extractTypeNameFromValue(route.Parameters[i].TypeValue)
		if err != nil {
			return fmt.Errorf("failed to extract parameter type name: %w", err)
		}

		route.Parameters[i].TypeName = typeName
	}

	// Add operation keyed by operationID
	g.httpOps[route.OperationID] = route

	return nil
}

// ExternalTypeInfo holds metadata about external Go types.
type ExternalTypeInfo struct {
	GoType        string // Original Go type (e.g., "time.Time")
	OpenAPIFormat string // OpenAPI format (e.g., "date-time")
}

// createTimeTypeKeyword creates a LiteralKeyword for time.Time and registers it as an external type.
// Each call creates a new keyword pointer and registers it in the external types map.
//
//nolint:ireturn
func (g *OpenAPICollector) createTimeTypeKeyword() bindings.ExpressionType {
	keywordPtr := utils.Ptr(bindings.KeywordString)
	g.externalTypes[keywordPtr] = &ExternalTypeInfo{
		GoType: "time.Time", OpenAPIFormat: "date-time",
	}

	return keywordPtr
}

// createURLTypeKeyword creates a LiteralKeyword for types.URL and registers it as an external type.
// Each call creates a new keyword pointer and registers it in the external types map.
//
//nolint:ireturn
func (g *OpenAPICollector) createURLTypeKeyword() bindings.ExpressionType {
	keywordPtr := utils.Ptr(bindings.KeywordString)
	g.externalTypes[keywordPtr] = &ExternalTypeInfo{
		GoType: "ws-json-rpc/backend/pkg/types.URL", OpenAPIFormat: "uri",
	}

	return keywordPtr
}

// newTypescriptASTFromGoTypesDir creates a TypeScript AST from Go type definitions,
// preserving comments and applying transformations for TypeScript compatibility.
func (g *OpenAPICollector) newTypescriptASTFromGoTypesDir(l *slog.Logger, goTypesDirPath string) (*guts.Typescript, error) {
	l.Debug("Parsing Go types directory", slog.String("path", goTypesDirPath))

	goParser, err := guts.NewGolangParser()
	if err != nil {
		return nil, fmt.Errorf("failed to create guts parser: %w", err)
	}

	goParser.PreserveComments()
	goParser.IncludeCustomDeclaration(map[string]guts.TypeOverride{
		"time.Time":                         g.createTimeTypeKeyword,
		"ws-json-rpc/backend/pkg/types.URL": g.createURLTypeKeyword,
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
		config.InterfaceToType,
		config.SimplifyOptional,
		config.NotNullMaps,
	)

	l.Debug("TypeScript AST generated successfully")

	return ts, nil
}

// extractAllTypes walks the TypeScript AST and extracts all type information in one pass.
func (g *OpenAPICollector) extractAllTypes() error {
	g.l.Debug("Starting type extraction from TypeScript AST")

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

	g.l.Debug("Completed type extraction", slog.Int("typeCount", len(g.types)))

	return nil
}

// extractTypeFromNode extracts TypeInfo from a TypeScript AST node.
func (g *OpenAPICollector) extractTypeFromNode(name string, node bindings.Node) (*TypeInfo, error) {
	g.l.Debug("Extracting type", slog.String("name", name), slog.String("nodeType", fmt.Sprintf("%T", node)))

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

// processFieldMember processes a field member and returns FieldInfo.
// This helper eliminates duplication between alias type members and interface fields.
func (g *OpenAPICollector) processFieldMember(parentName, fieldName string, fieldType bindings.ExpressionType, hasQuestionToken bool, comments bindings.SupportComments) (FieldInfo, error) {
	_, err := g.serializeExpressionType(fieldType)
	if err != nil {
		return FieldInfo{}, fmt.Errorf("failed to serialize field type for %s: %w", parentName, err)
	}

	// Extract structured type information
	fieldInfo, err := g.analyzeFieldType(fieldType)
	if err != nil {
		return FieldInfo{}, fmt.Errorf("failed to analyze field type for %s.%s: %w", parentName, fieldName, err)
	}

	// A field is required only if it has no question token AND is not nullable
	// In Go: *string becomes "string | null" (nullable, no question token) → should be optional
	fieldInfo.Required = !hasQuestionToken && !fieldInfo.Nullable

	fieldDesc := g.extractComments(comments)

	fieldDeprecated, cleanedFieldDesc, err := g.parseDeprecation(fieldDesc)
	if err != nil {
		return FieldInfo{}, fmt.Errorf("failed to parse deprecation info for field %s.%s: %w", parentName, fieldName, err)
	}

	field := FieldInfo{
		Name:        fieldName,
		DisplayType: generateDisplayType(fieldInfo),
		TypeInfo:    fieldInfo,
		Description: cleanedFieldDesc,
		Deprecated:  fieldDeprecated,
	}

	// Check if it's an external type and capture metadata
	if extInfo, exists := g.getExternalTypeInfo(fieldType); exists {
		field.GoType = extInfo.GoType
	}

	return field, nil
}

// extractAliasType extracts type information from an alias node.
func (g *OpenAPICollector) extractAliasType(name string, alias *bindings.Alias) (*TypeInfo, error) {
	desc := g.extractComments(alias.SupportComments)

	deprecated, cleanedDesc, err := g.parseDeprecation(desc)
	if err != nil {
		return nil, fmt.Errorf("failed to parse deprecation info for alias %s: %w", name, err)
	}

	// Assume it's a simple alias by default
	typeInfo := &TypeInfo{
		Name:        name,
		Kind:        TypeKindAlias,
		Description: cleanedDesc,
		Deprecated:  deprecated,
	}

	switch alias := alias.Type.(type) {
	case *bindings.UnionType:
		// Union types in Go->TS should only be nullable types (T | null)
		// Enums use bindings.Enum, not union literals
		g.l.Debug("Analyzing alias type kind", slog.String("type", name), slog.String("kind", "union"))

		// Check if it's nullable
		isNullable, _ := g.isNullableUnion(alias)
		if !isNullable {
			// Non-nullable unions shouldn't exist
			return nil, fmt.Errorf("unexpected non-nullable union type for alias %s - use Go enums instead", name)
		}
		// This is just a nullable alias - treat as object and field analysis will handle it
		typeInfo.Kind = TypeKindAlias
	case *bindings.TypeLiteralNode:
		g.l.Debug("Analyzing alias type kind", slog.String("type", name), slog.String("kind", "object"))
		// Check if it's an alias to a type literal (object)
		typeInfo.Fields = make([]FieldInfo, 0, len(alias.Members))
		typeInfo.Kind = TypeKindObject

		for _, member := range alias.Members {
			field, err := g.processFieldMember(name, member.Name, member.Type, member.QuestionToken, member.SupportComments)
			if err != nil {
				return nil, err
			}

			typeInfo.Fields = append(typeInfo.Fields, field)
		}

		// Collect references by walking raw AST (captures generics, intersections, tuples, inline objects)
		typeInfo.References = g.collectReferencesFromMembers(alias.Members)
	}

	g.l.Debug("Extracted alias type", slog.String("name", name), slog.String("kind", typeInfo.Kind), slog.Int("fieldCount", len(typeInfo.Fields)))

	return typeInfo, nil
}

// extractInterfaceType extracts type information from an interface node.
func (g *OpenAPICollector) extractInterfaceType(name string, iface *bindings.Interface) (*TypeInfo, error) {
	desc := g.extractComments(iface.SupportComments)

	deprecated, cleanedDesc, err := g.parseDeprecation(desc)
	if err != nil {
		return nil, fmt.Errorf("failed to parse deprecation info for interface %s: %w", name, err)
	}

	typeInfo := &TypeInfo{
		Name:        name,
		Kind:        TypeKindObject,
		Description: cleanedDesc,
		Deprecated:  deprecated,
		Fields:      []FieldInfo{},
		References:  g.collectReferencesFromMembers(iface.Fields),
	}

	for _, field := range iface.Fields {
		fieldInfo, err := g.processFieldMember(name, field.Name, field.Type, field.QuestionToken, field.SupportComments)
		if err != nil {
			return nil, err
		}

		typeInfo.Fields = append(typeInfo.Fields, fieldInfo)
	}

	g.l.Debug("Extracted interface type", slog.String("name", name), slog.Int("fieldCount", len(typeInfo.Fields)), slog.Bool("deprecated", deprecated != ""))

	return typeInfo, nil
}

// extractEnumType extracts type information from an enum node.
func (g *OpenAPICollector) extractEnumType(name string, enum *bindings.Enum) (*TypeInfo, error) {
	desc := g.extractComments(enum.SupportComments)

	enumVals, err := g.extractEnumMemberValues(enum)
	if err != nil {
		return nil, fmt.Errorf("failed to extract enum values for %s: %w", name, err)
	}

	g.l.Debug("Extracted enum type", slog.String("name", name), slog.Int("enumValueCount", len(enumVals)))

	deprecated, cleanedDesc, err := g.parseDeprecation(desc)
	if err != nil {
		return nil, fmt.Errorf("failed to parse deprecation info for enum %s: %w", name, err)
	}

	return &TypeInfo{
		Name:        name,
		Kind:        TypeKindStringEnum,
		Description: cleanedDesc,
		Deprecated:  deprecated,
		EnumValues:  enumVals,
	}, nil
}

func (g *OpenAPICollector) getDocumentation() *APIDocumentation {
	return &APIDocumentation{
		Types:          g.types,
		HTTPOperations: g.httpOps,
		Database:       g.database,
		Info:           g.apiInfo,
	}
}

// collectReferencesFromMembers collects direct type references by walking the raw AST
// Only collects named types - rejects generics, inline objects, tuples, and intersections.
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
// (References are already computed during type extraction).
func (g *OpenAPICollector) computeTypeRelationships() {
	g.l.Debug("Computing type relationships", slog.Int("typeCount", len(g.types)), slog.Int("operationCount", len(g.httpOps)))

	// Build ReferencedBy from References
	g.buildReferencedBy()
	g.l.Debug("Built ReferencedBy relationships")

	// Build UsedBy from routes
	g.buildUsedBy()
	g.l.Debug("Computed UsedBy relationships")
}

// generateJSONSchemaRepresentations generates JSON Schema for all types as a post-processing step.
func (g *OpenAPICollector) generateJSONSchemaRepresentations() error {
	g.l.Debug("Generating JSON schemas for all types", slog.Int("typeCount", len(g.types)))

	for name, typeInfo := range g.types {
		schema, err := toOpenAPISchema(typeInfo)
		if err != nil {
			return fmt.Errorf("failed to generate JSON schema for type %s: %w", name, err)
		}

		jsonSchema, err := schemaToJSONString(schema)
		if err != nil {
			return fmt.Errorf("failed to serialize JSON schema for type %s: %w", name, err)
		}

		typeInfo.Representations.JSONSchema = jsonSchema
	}

	g.l.Debug("JSON schemas generated successfully")

	return nil
}

// generateTSRepresentations generates TypeScript representations for all types as a post-processing step.
func (g *OpenAPICollector) generateTSRepresentations() error {
	g.l.Debug("Generating TypeScript representations for all types", slog.Int("typeCount", len(g.types)))

	for name, typeInfo := range g.types {
		node, exists := g.tsParser.Node(name)
		if !exists {
			return fmt.Errorf("type %s not found in TypeScript AST", name)
		}

		tsRepresentation, err := g.serializeNode(node)
		if err != nil {
			return fmt.Errorf("failed to generate TypeScript representation for type %s: %w", name, err)
		}

		typeInfo.Representations.TS = tsRepresentation
	}

	g.l.Debug("TypeScript representations generated successfully")

	return nil
}

// buildReferencedBy builds the inverse of References for all types.
func (g *OpenAPICollector) buildReferencedBy() {
	for typeName, typeInfo := range g.types {
		for _, ref := range typeInfo.References {
			if refType, exists := g.types[ref]; exists {
				refType.ReferencedBy = append(refType.ReferencedBy, typeName)
			}
		}
	}

	// Sort for deterministic output
	for _, typeInfo := range g.types {
		sort.Strings(typeInfo.ReferencedBy)
	}
}

// buildUsedBy tracks which operations use each type.
func (g *OpenAPICollector) buildUsedBy() {
	for _, route := range g.httpOps {
		// Track request type
		if route.Request != nil {
			g.addUsage(route.Request.TypeName, route.OperationID, "request")
		}

		// Track response types
		for _, resp := range route.Responses {
			g.addUsage(resp.TypeName, route.OperationID, "response")
		}

		// Track parameter types
		for _, param := range route.Parameters {
			g.addUsage(param.TypeName, route.OperationID, "parameter")
		}
	}
}

// addUsage adds a UsageInfo entry to a type if it exists.
func (g *OpenAPICollector) addUsage(typeName, operationID, role string) {
	if typeName == "" {
		return
	}

	if typeInfo, exists := g.types[typeName]; exists {
		typeInfo.UsedBy = append(typeInfo.UsedBy, UsageInfo{
			OperationID: operationID,
			Role:        role,
		})
	}
}

// collectExpressionTypeReferences collects direct type references from an expression
// Only collects named types - rejects generics and inline objects (which should error during analysis).
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

	// Primitive types - no references to collect
	case *bindings.LiteralKeyword:
	case *bindings.LiteralType:
	case *bindings.TypeLiteralNode:
		panic("inline object found during reference collection - should have been rejected earlier")
	case *bindings.TypeIntersection:
		panic("intersection type found during reference collection - should have been rejected earlier")
	}
}

// generateOpenAPISpec generates a complete OpenAPI specification from all collected metadata.
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

func (g *OpenAPICollector) serializeNode(node bindings.Node) (string, error) {
	tsNode, err := g.vm.ToTypescriptNode(node)
	if err != nil {
		return "", fmt.Errorf("failed to convert node to TypeScript node: %w", err)
	}

	serialized, err := g.vm.SerializeToTypescript(tsNode)
	if err != nil {
		return "", fmt.Errorf("failed to serialize TypeScript node: %w", err)
	}

	var str strings.Builder

	for line := range strings.SplitSeq(serialized, "\n") {
		if strings.HasPrefix(line, "// From") {
			continue
		}

		str.WriteString(line + "\n")
	}

	return str.String(), nil
}

// serializeExpressionType converts an expression type to its TypeScript string representation.
func (g *OpenAPICollector) serializeExpressionType(expr bindings.ExpressionType) (string, error) {
	if expr == nil {
		return "", errors.New("expression type is nil")
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

const DEPRECATED_PREFIX = "deprecated:"

// parseDeprecation extracts deprecation info from comments and returns cleaned description.
// It looks for "Deprecated:" anywhere in the text (case-insensitive) and captures the message.
// Returns (deprecationInfo, cleanedDescription, error).
func (g *OpenAPICollector) parseDeprecation(comments string) (string, string, error) {
	if comments == "" {
		return "", "", nil
	}

	// Look for "Deprecated:" anywhere in the text (case-insensitive)
	lowerComments := strings.ToLower(comments)
	idx := strings.Index(lowerComments, DEPRECATED_PREFIX)

	if idx == -1 {
		return "", comments, nil
	}

	// Extract the message after "Deprecated:"
	// Start from the original string to preserve casing
	message := strings.TrimSpace(comments[idx+len(DEPRECATED_PREFIX):])

	// Clean the description by removing the deprecation text
	cleanedDesc := strings.TrimSpace(comments[:idx])

	if message == "" {
		return "", cleanedDesc, errors.New("deprecation message is empty")
	}

	return message, cleanedDesc, nil
}

// isNullableUnion checks if a union type represents a nullable pattern (T | null or T | undefined).
// Returns true and the non-null type if it's nullable, false and nil otherwise.
//
//nolint:ireturn
func (g *OpenAPICollector) isNullableUnion(union *bindings.UnionType) (bool, bindings.ExpressionType) {
	if len(union.Types) != 2 {
		return false, nil
	}

	var nonNullType bindings.ExpressionType

	hasNull := false

	for _, t := range union.Types {
		serialized, err := g.serializeExpressionType(t)
		if err == nil && (serialized == "null" || serialized == "undefined") {
			hasNull = true

			continue
		}

		nonNullType = t
	}

	if hasNull && nonNullType != nil {
		g.l.Debug("Detected nullable union", slog.String("nonNullType", fmt.Sprintf("%T", nonNullType)))

		return true, nonNullType
	}

	return false, nil
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

// extractEnumMemberValues extracts string literal values with comments from enum members.
func (g *OpenAPICollector) extractEnumMemberValues(enum *bindings.Enum) ([]EnumValue, error) {
	var values []EnumValue

	for _, member := range enum.Members {
		// Serialize the enum member value to get its string representation
		valueStr, err := g.serializeExpressionType(member.Value)
		if err != nil {
			return nil, fmt.Errorf("failed to serialize enum member value for %s.%s: %w", enum.Name.String(), member.Name, err)
		}

		// Remove quotes from string literals
		valueStr = strings.Trim(valueStr, "\"'")

		memberDesc := g.extractComments(member.SupportComments)

		deprecated, cleanedMemberDesc, err := g.parseDeprecation(memberDesc)
		if err != nil {
			return nil, fmt.Errorf("failed to parse deprecation info for enum member %s.%s: %w", enum.Name.String(), member.Name, err)
		}

		values = append(values, EnumValue{
			Value:       valueStr,
			Description: cleanedMemberDesc,
			Deprecated:  deprecated,
		})
	}

	return values, nil
}

// generateDisplayType creates a human-readable type string from FieldType.
func generateDisplayType(ft FieldType) string {
	switch ft.Kind {
	case FieldKindReference, FieldKindEnum:
		return ft.Type
	case FieldKindPrimitive:
		caser := cases.Title(language.English)

		return caser.String(ft.Type)

	case FieldKindArray:
		if ft.ItemsType != nil {
			itemDisplay := generateDisplayType(*ft.ItemsType)

			return itemDisplay + "[]"
		}

		return "Array"

	case FieldKindObject:
		return "Object"

	default:
		panic(fmt.Sprintf("unexpected field kind: %s, should have been caught by type analysis", ft.Kind))
	}
}

func (g *OpenAPICollector) analyzeFieldType(expr bindings.ExpressionType) (FieldType, error) {
	if expr == nil {
		return FieldType{Kind: FieldKindUnknown, Type: "unknown"}, errors.New("cannot analyze nil expression type")
	}

	switch e := expr.(type) {
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

		// Special case: Record<K, V> types (Go maps)
		if refName == "Record" && len(e.Arguments) == 2 {
			// Analyze the value type (second argument)
			valueType, err := g.analyzeFieldType(e.Arguments[1])
			if err != nil {
				return FieldType{}, fmt.Errorf("failed to analyze Record value type: %w", err)
			}

			// Return object type with additionalProperties
			return FieldType{
				Kind:                 FieldKindObject,
				Type:                 "object",
				AdditionalProperties: &valueType,
			}, nil
		}

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
		g.l.Debug("Analyzing union type", slog.Int("numTypes", len(e.Types)))

		// Check for nullable pattern FIRST: T | null or null | T or T | undefined
		isNullable, nonNullType := g.isNullableUnion(e)
		if !isNullable {
			// Unwrap single-element unions and analyze the inner type
			if len(e.Types) == 1 {
				return g.analyzeFieldType(e.Types[0])
			}

			// Log the union types for debugging
			var typeStrings []string
			for _, t := range e.Types {
				serialized, err := g.serializeExpressionType(t)
				if err != nil {
					typeStrings = append(typeStrings, fmt.Sprintf("%T (error: %v)", t, err))
				} else {
					typeStrings = append(typeStrings, serialized)
				}
			}
			g.l.Error("Non-nullable union type detected", slog.String("types", strings.Join(typeStrings, " | ")))
			// Non-nullable unions shouldn't exist in Go->TS serialization
			// (Go enums use bindings.Enum, not union literals)
			return FieldType{}, errors.New("unexpected non-nullable union type - Go->TS serialization should only have nullable unions (T | null)")
		}

		result, err := g.analyzeFieldType(nonNullType)
		if err != nil {
			return FieldType{}, fmt.Errorf("failed to analyze nullable type: %w", err)
		}

		result.Nullable = true
		g.l.Debug("Successfully analyzed nullable union", slog.String("baseType", result.Type))

		return result, nil

	case *bindings.LiteralKeyword:
		// Literal keywords (primitives)
		// Serialize to get the actual TypeScript type name
		typeStr, err := g.serializeExpressionType(e)
		if err != nil {
			return FieldType{}, fmt.Errorf("failed to serialize keyword type: %w", err)
		}

		var format string
		// Check if this is an external type (e.g., time.Time -> string with date-time format)
		if extInfo, exists := g.getExternalTypeInfo(e); exists {
			format = extInfo.OpenAPIFormat
		}

		return FieldType{
			Kind:   FieldKindPrimitive,
			Type:   typeStr,
			Format: format,
		}, nil
	case *bindings.ArrayLiteralType:
		// Tuple types are not supported in Go->TS serialization
		return FieldType{}, errors.New("tuple types are not supported - Go does not have tuple types")

	case *bindings.TypeLiteralNode:
		// Inline objects are not allowed - require named types
		return FieldType{}, errors.New("inline object types are not supported - please create a named type instead")

	case *bindings.TypeIntersection:
		// Type intersections are not supported in Go->TS serialization
		return FieldType{}, errors.New("intersection types are not supported")

	default:
		return FieldType{}, fmt.Errorf("unsupported expression type: %T", expr)
	}
}

// writeSpecYAML writes the OpenAPI specification to a YAML file.
func (g *OpenAPICollector) writeSpecYAML(filename string) error {
	spec, err := g.generateOpenAPISpec()
	if err != nil {
		return fmt.Errorf("failed to generate spec: %w", err)
	}

	yamlData, err := yaml.Marshal(spec)
	if err != nil {
		return err
	}

	return os.WriteFile(filename, yamlData, 0600)
}

// writeDocsJSON writes the complete API documentation to a JSON file.
func (g *OpenAPICollector) writeDocsJSON() error {
	if g.docsFilePath == "" {
		return nil // Skip if no path configured
	}

	doc := g.getDocumentation()

	// Use GenerateAPIDocs for sorted, deterministic output
	if err := GenerateAPIDocs(g.l, doc, g.docsFilePath); err != nil {
		return fmt.Errorf("failed to write docs JSON: %w", err)
	}

	g.l.Info("API documentation written", slog.String("file", g.docsFilePath))

	return nil
}

// getExternalTypeInfo returns the external type metadata for a given expression.
// Returns nil if the expression is not an external type.
// Unwraps nullable unions (T | null) to check the underlying type.
func (g *OpenAPICollector) getExternalTypeInfo(expr bindings.ExpressionType) (*ExternalTypeInfo, bool) {
	switch e := expr.(type) {
	case *bindings.LiteralKeyword:
		info, exists := g.externalTypes[e]

		return info, exists
	case *bindings.UnionType:
		if isNullable, nonNullType := g.isNullableUnion(e); isNullable {
			// Recursively check the non-null type
			return g.getExternalTypeInfo(nonNullType)
		}
	}

	return nil, false
}

// extractTypeNameFromValue extracts the type name from a Go value using reflection.
// If the value is nil, typeName is set to empty string and no error is returned.
func extractTypeNameFromValue(value any) (string, error) {
	if value == nil {
		return "", nil
	}

	rt := reflect.TypeOf(value)
	if rt == nil {
		return "", nil
	}

	// Handle pointers
	for rt.Kind() == reflect.Pointer {
		rt = rt.Elem()
	}

	name := rt.Name()
	if name == "" {
		return "", errors.New("anonymous type not supported")
	}

	return name, nil
}
