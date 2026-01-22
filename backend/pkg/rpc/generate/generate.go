package generate

import (
	"bytes"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"reflect"
	"slices"
	"sort"
	"strings"
	"time"

	"ws-json-rpc/backend/internal/database/sqlite"
	"ws-json-rpc/backend/pkg/database"
	"ws-json-rpc/backend/pkg/utils"
)

const (
	NULL_TYPE_NAME = "null"
)

// GeneratorImpl is the concrete implementation of the Generator interface.
// It manages type registration and documentation generation.
// Types are registered as methods/events are added during server startup.
type GeneratorImpl struct {
	l                *slog.Logger   // Logger for debugging and error reporting
	d                *Docs          // API documentation structure
	guts             *GutsGenerator // TypeScript AST parser and metadata extractor
	docsFilePath     string         // Output path for API docs JSON
	dbSchemaFilePath string         // Output path for database schema SQL
}

// GeneratorOptions contains all configuration needed to create a Generator.
// All paths must be provided for the generator to function properly.
type GeneratorOptions struct {
	GoTypesDirPath               string      // Path to Go types file for parsing
	DocsFileOutputPath           string      // Path for generated API docs JSON file
	TSTypesOutputPath            string      // Path for generated TypeScript types file
	DatabaseSchemaFileOutputPath string      // Path for generated database schema SQL file
	DocsOptions                  DocsOptions // Docs options
}

// NewGenerator creates a Generator that validates options, initializes the TypeScript parser,
// writes type definitions, and sets up documentation structures.
// Types are registered dynamically as methods/events are added.
func NewGenerator(l *slog.Logger, opts GeneratorOptions) (*GeneratorImpl, error) {
	l.Debug("Creating API documentation generator",
		slog.String("docsOutput", opts.DocsFileOutputPath),
		slog.String("tsOutput", opts.TSTypesOutputPath),
		slog.String("schemaOutput", opts.DatabaseSchemaFileOutputPath))

	if opts.DocsFileOutputPath == "" {
		return nil, fmt.Errorf("docs file path is required")
	}
	if opts.DatabaseSchemaFileOutputPath == "" {
		return nil, fmt.Errorf("schema file path is required")
	}

	gutsGenerator, err := NewGutsGenerator(l, opts.GoTypesDirPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create GutsGenerator: %w", err)
	}

	if err := gutsGenerator.WriteTypescriptASTToFile(gutsGenerator.tsParser, opts.TSTypesOutputPath); err != nil {
		return nil, fmt.Errorf("failed to write TypeScript AST to file: %w", err)
	}

	g := &GeneratorImpl{
		l:                l.With(slog.String("component", "generator")),
		d:                NewDocs(opts.DocsOptions),
		guts:             gutsGenerator,
		docsFilePath:     opts.DocsFileOutputPath,
		dbSchemaFilePath: opts.DatabaseSchemaFileOutputPath,
	}

	l.Info("API documentation generator created successfully")
	return g, nil
}

// GetDatabaseSchema runs migrations on a temporary database and returns the resulting schema.
func (g *GeneratorImpl) GetDatabaseSchema() (string, error) {
	g.l.Debug("Generating database schema from migrations")

	tempDBPath := fmt.Sprintf("%s-%d", os.TempDir(), time.Now().Unix())
	mig, err := database.NewMigrator(g.l, sqlite.GetMigrationsFS(), tempDBPath)
	if err != nil {
		return "", fmt.Errorf("failed to create migrator: %w", err)
	}

	if err := mig.Migrate(); err != nil {
		return "", fmt.Errorf("failed to migrate database: %w", err)
	}

	if err = mig.DumpSchema(g.dbSchemaFilePath); err != nil {
		return "", fmt.Errorf("failed to dump schema: %w", err)
	}

	schemaBytes, err := os.ReadFile(g.dbSchemaFilePath)
	if err != nil {
		return "", fmt.Errorf("failed to read schema file: %w", err)
	}

	g.l.Info("Database schema generated", slog.String("file", g.dbSchemaFilePath))
	return string(bytes.TrimSpace(schemaBytes)), nil
}

// Generate produces the final API documentation JSON file.
// Must be called after all methods and events have been registered.
func (g *GeneratorImpl) Generate() error {
	g.l.Info("Starting API documentation generation",
		slog.Int("methods", len(g.d.Methods)),
		slog.Int("events", len(g.d.Events)),
		slog.Int("types", len(g.d.Types)))

	// Get database schema
	schema, err := g.GetDatabaseSchema()
	if err != nil {
		return fmt.Errorf("failed to get database schema: %w", err)
	}
	g.d.DatabaseSchema = schema

	// Compute back-references for all types
	g.l.Debug("Computing type back-references")
	g.computeBackReferences()

	// Compute usedBy information for all types
	g.l.Debug("Computing type usage information")
	g.computeUsedBy()

	// Write API docs to file
	g.l.Debug("Writing API documentation to file", slog.String("file", g.docsFilePath))
	docsFile, err := os.Create(g.docsFilePath)
	if err != nil {
		return fmt.Errorf("failed to create api docs file: %w", err)
	}
	defer func() {
		if err := docsFile.Close(); err != nil {
			g.l.Error("failed to close api docs file", utils.ErrAttr(err))
		}
	}()

	if err := utils.ToJSONStreamIndent(docsFile, g.d); err != nil {
		return fmt.Errorf("failed to write api docs: %w", err)
	}

	g.l.Info("API documentation generated successfully", slog.String("file", g.docsFilePath))

	return nil
}

// AddEventType registers a WebSocket event with its response type and documentation.
func (g *GeneratorImpl) AddEventType(name string, resp any, docs EventDocs) {
	if _, exists := g.d.Events[name]; exists {
		g.fatalIfErr(errors.New("event already registered: " + name))
	}

	docs.NoNilSlices()
	if err := docs.Validate(); err != nil {
		g.fatalIfErr(fmt.Errorf("failed to validate event docs: %w", err))
	}

	for idx, ex := range docs.Examples {
		docs.Examples[idx].Result = string(utils.MustToJSON(ex.ResultObj))
	}

	docs.Protocols.WS = true
	// Events are only available for WebSocket connections
	docs.Protocols.HTTP = false
	resultTypeName := g.mustGetTypeName(resp)
	docs.ResultType = Ref{Ref: resultTypeName}

	// Register type with JSON instance
	g.registerType(resultTypeName, resp)

	g.d.Events[name] = docs
	g.l.Debug("Event registered", slog.String("event", name), slog.String("resultType", resultTypeName))
}

// AddHandlerType registers an RPC method with its request/response types and documentation.
func (g *GeneratorImpl) AddHandlerType(name string, req any, resp any, docs MethodDocs) {
	if _, exists := g.d.Methods[name]; exists {
		g.fatalIfErr(errors.New("method already registered: " + name))
	}

	docs.NoNilSlices()
	if err := docs.Validate(); err != nil {
		g.fatalIfErr(fmt.Errorf("failed to validate method docs: %w", err))
	}

	for idx, ex := range docs.Examples {
		docs.Examples[idx].Result = string(utils.MustToJSONIndent(ex.ResultObj))
		docs.Examples[idx].Params = string(utils.MustToJSONIndent(ex.ParamsObj))
	}

	docs.Protocols.HTTP = !docs.NoHTTP
	docs.Protocols.WS = true

	resultTypeName := g.mustGetTypeName(resp)
	paramTypeName := g.mustGetTypeName(req)
	docs.ParamType = Ref{Ref: paramTypeName}
	docs.ResultType = Ref{Ref: resultTypeName}

	// Register types with JSON instances
	g.registerType(paramTypeName, req)
	g.registerType(resultTypeName, resp)

	g.d.Methods[name] = docs
	g.l.Debug("Method registered",
		slog.String("method", name),
		slog.String("paramType", paramTypeName),
		slog.String("resultType", resultTypeName),
		slog.Bool("http", docs.Protocols.HTTP))
}

// computeBackReferences builds reverse relationships, allowing navigation from a type
// to all types that reference it.
func (g *GeneratorImpl) computeBackReferences() {
	// First, clear all existing back-references
	for name := range g.d.Types {
		typeDocs := g.d.Types[name]
		typeDocs.ReferencedBy = nil
		g.d.Types[name] = typeDocs
	}

	// Build back-references by iterating through all types
	for typeName, typeDocs := range g.d.Types {
		// For each type this type references, add this type to its ReferencedBy list
		for _, refName := range typeDocs.References {
			refTypeDocs, exists := g.d.Types[refName]
			if !exists {
				continue
			}

			// Skip if already present
			if slices.Contains(refTypeDocs.ReferencedBy, typeName) {
				continue
			}

			refTypeDocs.ReferencedBy = append(refTypeDocs.ReferencedBy, typeName)
			g.d.Types[refName] = refTypeDocs
		}
	}

	// Sort ReferencedBy lists for deterministic output
	totalBackRefs := 0
	for name := range g.d.Types {
		typeDocs := g.d.Types[name]
		if len(typeDocs.ReferencedBy) > 0 {
			sort.Strings(typeDocs.ReferencedBy)
			g.d.Types[name] = typeDocs
			totalBackRefs += len(typeDocs.ReferencedBy)
		}
	}

	g.l.Debug("Computed back-references for all types", slog.Int("totalBackRefs", totalBackRefs))
}

// usedByLess returns a comparison function that sorts by Type, Target, then Role.
func usedByLess(usedBy []UsedBy) func(i, j int) bool {
	return func(i, j int) bool {
		if usedBy[i].Type != usedBy[j].Type {
			return usedBy[i].Type < usedBy[j].Type
		}
		if usedBy[i].Target != usedBy[j].Target {
			return usedBy[i].Target < usedBy[j].Target
		}
		return usedBy[i].Role < usedBy[j].Role
	}
}

// computeUsedBy records which methods and events use each type as a parameter or result.
func (g *GeneratorImpl) computeUsedBy() {
	// First, clear all existing usedBy information
	for name := range g.d.Types {
		typeDocs := g.d.Types[name]
		typeDocs.UsedBy = nil
		g.d.Types[name] = typeDocs
	}

	// Add usedBy information from methods
	for methodName, methodDocs := range g.d.Methods {
		g.addTypeUsage(methodDocs.ParamType.Ref, "method", methodName, "param")
		g.addTypeUsage(methodDocs.ResultType.Ref, "method", methodName, "result")
	}

	// Add usedBy information from events
	for eventName, eventDocs := range g.d.Events {
		g.addTypeUsage(eventDocs.ResultType.Ref, "event", eventName, "result")
	}

	// Sort UsedBy lists for deterministic output
	totalUsages := 0
	for name := range g.d.Types {
		typeDocs := g.d.Types[name]
		if len(typeDocs.UsedBy) > 0 {
			sort.Slice(typeDocs.UsedBy, usedByLess(typeDocs.UsedBy))
			g.d.Types[name] = typeDocs
			totalUsages += len(typeDocs.UsedBy)
		}
	}

	g.l.Debug("Computed usedBy information for all types", slog.Int("totalUsages", totalUsages))
}

// addTypeUsage adds a usage record for a type if it exists and is not null.
func (g *GeneratorImpl) addTypeUsage(typeRef, usageType, target, role string) {
	if typeRef == "" || typeRef == NULL_TYPE_NAME {
		return
	}

	typeDocs, exists := g.d.Types[typeRef]
	if !exists {
		return
	}

	typeDocs.UsedBy = append(typeDocs.UsedBy, UsedBy{
		Type: usageType, Target: target, Role: role,
	})
	g.d.Types[typeRef] = typeDocs
}

// registerType registers a type with optional JSON instance.
// If v is nil, only TypeScript information is registered (for referenced types).
// If v is not nil, also includes JSON representation (for explicitly registered types).
// Recursively registers any types this type references.
func (g *GeneratorImpl) registerType(name string, v any) {
	if name == NULL_TYPE_NAME {
		return
	}

	// Check if type already exists
	if docs, exists := g.d.Types[name]; exists {
		// Type already registered with JSON instance, don't overwrite
		if docs.JsonRepresentation != "" {
			g.l.Debug("Type already registered with JSON instance", slog.String("type", name))
			return
		}
	}

	hasInstance := v != nil

	g.l.Debug("Registering type", slog.String("type", name), slog.Bool("hasInstance", hasInstance))
	var jsonRepresentation string

	if hasInstance {
		// Add JSON representation if we have a Go instance
		jsonRepresentation = string(utils.MustToJSONIndent(v))
	}

	// Extract description from Go comments
	description, err := g.guts.ExtractTypeDescription(name)
	if err != nil {
		g.l.Warn("Failed to extract description from TypeScript AST", slog.String("type", name), slog.String("error", err.Error()))
	}

	// Extract TypeScript type from AST
	tsType, err := g.guts.SerializeNode(name)
	if err != nil {
		g.fatalIfErr(fmt.Errorf("failed to serialize TypeScript AST node: %w", err))
	}

	// Extract all type metadata from TypeScript AST
	metadata := g.extractTypeMetadata(name)

	typeDocs := TypeDocs{
		Description:        strings.TrimSpace(description),
		JsonRepresentation: jsonRepresentation,
		TSType:             tsType,
		Kind:               metadata.kind,
		Fields:             metadata.fields,
		References:         metadata.references,
		EnumValues:         metadata.enumValues,
	}

	g.d.Types[name] = typeDocs

	// Recursively register any referenced types that haven't been registered yet
	for _, refName := range metadata.references {
		if _, exists := g.d.Types[refName]; !exists {
			g.l.Debug("Registering referenced type", slog.String("type", refName), slog.String("referencedBy", name))
			g.registerType(refName, nil)
		}
	}
}

// typeMetadata holds extracted metadata from TypeScript AST.
type typeMetadata struct {
	kind       string
	fields     []FieldMetadata
	references []string
	enumValues []string
}

// extractTypeMetadata extracts all metadata for a type, logging warnings and using defaults on errors.
func (g *GeneratorImpl) extractTypeMetadata(name string) typeMetadata {
	var metadata typeMetadata

	// Extract type kind
	kind, err := g.guts.ExtractTypeKind(name)
	if err != nil {
		g.l.Warn("Failed to extract type kind from TypeScript AST", slog.String("type", name), slog.String("error", err.Error()))
		kind = "Unknown"
	}
	metadata.kind = kind

	// Extract field metadata
	fields, err := g.guts.ExtractFields(name)
	if err != nil {
		g.l.Warn("Failed to extract fields from TypeScript AST", slog.String("type", name), slog.String("error", err.Error()))
		fields = []FieldMetadata{}
	}
	metadata.fields = fields

	// Extract references
	references, err := g.guts.ExtractReferences(name)
	if err != nil {
		g.l.Warn("Failed to extract references from TypeScript AST", slog.String("type", name), slog.String("error", err.Error()))
		references = []string{}
	}
	metadata.references = references

	// Extract type-level enum values
	enumValues, err := g.guts.ExtractTypeEnumValues(name)
	if err != nil {
		g.l.Warn("Failed to extract enum values from TypeScript AST", slog.String("type", name), slog.String("error", err.Error()))
		enumValues = []string{}
	}
	metadata.enumValues = enumValues

	return metadata
}

// fatalIfErr logs the error and exits if err is not nil.
func (g *GeneratorImpl) fatalIfErr(err error) {
	if err == nil {
		return
	}

	g.l.Error("generator error", utils.ErrAttr(err))
	os.Exit(1)
}

// mustGetTypeName extracts the type name from a value, requiring it to be a named struct.
// Returns [NULL_TYPE_NAME] for empty struct{} (representing no params/result).
func (g *GeneratorImpl) mustGetTypeName(v any) string {
	// Handle nil
	if v == nil {
		g.fatalIfErr(errors.New("type must be a named struct, got: nil"))
	}

	// This is cases where there are no params or result
	if v == struct{}{} {
		return NULL_TYPE_NAME
	}

	t := reflect.TypeOf(v)
	// Only named structs are allowed
	if !isNamedStruct(t) {
		g.fatalIfErr(errors.New("type must be a named struct, got: " + t.String()))
	}

	// Handle pointers - get the actual struct name
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}

	return t.Name()
}

// isNamedStruct checks if a type is a named struct (not anonymous).
func isNamedStruct(t reflect.Type) bool {
	// Handle nil
	if t == nil {
		return false
	}

	// Handle pointers - get the underlying type
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}

	// Check if it's a struct
	if t.Kind() != reflect.Struct {
		return false
	}

	// Check if it has a name (named types have PkgPath and Name)
	return t.Name() != ""
}
