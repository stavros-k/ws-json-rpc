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

	"github.com/coder/guts"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3gen"

	"ws-json-rpc/internal/database/sqlite"
	"ws-json-rpc/pkg/database"
	"ws-json-rpc/pkg/utils"
)

// Generator is the main interface for generating API documentation and type definitions.
// It orchestrates schema parsing, type registration, and documentation generation.
type Generator interface {
	// Generate produces the final API documentation file and database schema.
	Generate() error
	// AddEventType registers a WebSocket event with its response type and documentation.
	AddEventType(name string, resp any, docs EventDocs)
	// AddHandlerType registers an RPC method with its request/response types and documentation.
	AddHandlerType(name string, req any, resp any, docs MethodDocs)
}

// GeneratorImpl is the concrete implementation of the Generator interface.
// It manages type registration and documentation generation.
// Types are registered as methods/events are added during server startup.
type GeneratorImpl struct {
	l                *slog.Logger                   // Logger for debugging and error reporting
	d                *Docs                          // API documentation structure
	schemaGen        *openapi3gen.Generator         // OpenAPI schema generator for JSON schemas
	componentSchemas openapi3.Schemas               // Shared component schemas for all types
	schemaRegistry   map[string]*openapi3.SchemaRef // Registry of all generated schemas for reference extraction
	tsParser         *guts.Typescript               // TypeScript AST parser
	docsFilePath     string                         // Output path for API docs JSON
	dbSchemaFilePath string                         // Output path for database schema SQL
}

// GeneratorOptions contains all configuration needed to create a Generator.
// All paths must be provided for the generator to function properly.
type GeneratorOptions struct {
	GoTypesDirPath               string      // Path to Go types file for parsing
	DocsFileOutputPath           string      // Path for generated API docs JSON file
	DatabaseSchemaFileOutputPath string      // Path for generated database schema SQL file
	DocsOptions                  DocsOptions // Docs options
}

// NewGenerator creates a new Generator instance with the given options.
// It performs the following initialization steps:
// 1. Validates all required options are provided
// 2. Creates the docs structure
//
// Types are registered dynamically when methods/events are added.
func NewGenerator(l *slog.Logger, opts GeneratorOptions) (Generator, error) {
	if opts.DocsFileOutputPath == "" {
		return nil, fmt.Errorf("docs file path is required")
	}
	if opts.DatabaseSchemaFileOutputPath == "" {
		return nil, fmt.Errorf("schema file path is required")
	}

	ts, err := newTypescriptASTFromGoTypesDir(opts.GoTypesDirPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create TypeScript AST from go types dir: %w", err)
	}

	// TODO: make the output path configurable
	if err := writeTypescriptASTToFile(ts, "test.ts"); err != nil {
		return nil, fmt.Errorf("failed to write TypeScript AST to file: %w", err)
	}

	g := &GeneratorImpl{
		l:                l.With(slog.String("component", "generator")),
		d:                NewDocs(opts.DocsOptions),
		schemaGen:        newOpenAPISchemaGenerator(),
		componentSchemas: make(openapi3.Schemas),
		schemaRegistry:   make(map[string]*openapi3.SchemaRef),
		tsParser:         ts,
		docsFilePath:     opts.DocsFileOutputPath,
		dbSchemaFilePath: opts.DatabaseSchemaFileOutputPath,
	}

	return g, nil
}

// GetDatabaseSchema generates the database schema by running migrations on a temporary database.
// It creates a temporary SQLite database, runs all migrations, and dumps the resulting schema.
// Returns the schema as a string for inclusion in API documentation.
func (g *GeneratorImpl) GetDatabaseSchema() (string, error) {
	mig, err := database.NewMigrator(
		sqlite.GetMigrationsFS(),
		fmt.Sprintf("%s-%d", os.TempDir(), time.Now().Unix()),
		g.l,
	)
	if err != nil {
		return "", fmt.Errorf("failed to create migrator: %w", err)
	}
	if err := mig.Migrate(); err != nil {
		return "", fmt.Errorf("failed to migrate database: %w", err)
	}

	err = mig.DumpSchema(g.dbSchemaFilePath)
	if err != nil {
		return "", fmt.Errorf("failed to dump schema: %w", err)
	}
	schemaBytes, err := os.ReadFile(g.dbSchemaFilePath)
	if err != nil {
		return "", fmt.Errorf("failed to read schema file: %w", err)
	}

	return string(bytes.TrimSpace(schemaBytes)), nil
}

// Generate produces the final API documentation JSON file.
// It performs the following steps:
// 1. Generates database schema from migrations
// 2. Writes complete API documentation (types, methods, events, database schema) to JSON file
//
// This should be called after all methods and events have been registered via
// AddHandlerType and AddEventType.
func (g *GeneratorImpl) Generate() error {
	// Get database schema
	schema, err := g.GetDatabaseSchema()
	if err != nil {
		return fmt.Errorf("failed to get database schema: %w", err)
	}
	g.d.DatabaseSchema = schema

	// Compute back-references for all types
	g.computeBackReferences()

	// Write API docs to file
	docsFile, err := os.Create(g.docsFilePath)
	if err != nil {
		return fmt.Errorf("failed to create api docs file: %w", err)
	}
	defer docsFile.Close()

	if err := utils.ToJSONStreamIndent(docsFile, g.d); err != nil {
		return fmt.Errorf("failed to write api docs: %w", err)
	}
	g.l.Info("API docs generated", slog.String("file", "api_docs.json"))

	return nil
}

// computeBackReferences computes which types are referenced by other types.
// For each type A that references type B, adds A to B's ReferencedBy list.
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
	for name := range g.d.Types {
		typeDocs := g.d.Types[name]
		if len(typeDocs.ReferencedBy) > 0 {
			sort.Strings(typeDocs.ReferencedBy)
			g.d.Types[name] = typeDocs
		}
	}

	g.l.Debug("Computed back-references for all types")
}

// AddEventType registers a WebSocket event with its response type and documentation.
// Events are unidirectional server-to-client messages sent over WebSocket connections.
// The response type must be a named struct.
//
// This method:
// 1. Validates the event hasn't been registered already
// 2. Validates the event documentation
// 3. Converts example objects to JSON strings
// 4. Sets the result type reference
// 5. Registers the type with its JSON instance
func (g *GeneratorImpl) AddEventType(name string, resp any, docs EventDocs) {
	if _, exists := g.d.Events[name]; exists {
		g.fatalIfErr(errors.New("event already registered: " + name))
	}
	if err := docs.Validate(); err != nil {
		g.fatalIfErr(err)
	}

	for idx, ex := range docs.Examples {
		docs.Examples[idx].Result = string(utils.MustToJSON(ex.ResultObj))
	}

	docs.NoNilSlices()

	docs.Protocols.WS = true
	// Events are only available for WebSocket connections
	docs.Protocols.HTTP = false
	resultTypeName := g.mustGetTypeName(resp)
	docs.ResultType = Ref{Ref: resultTypeName}

	// Register type with JSON instance
	g.registerType(resultTypeName, resp)

	g.d.Events[name] = docs
}

// AddHandlerType registers an RPC method with its request/response types and documentation.
// Methods are bidirectional request-response calls available over WebSocket and optionally HTTP.
//
// This method:
// 1. Validates the method hasn't been registered already
// 2. Validates the method documentation
// 3. Converts example objects to JSON strings
// 4. Sets parameter and result type references
// 5. Registers both types with their JSON instances
// 6. Configures protocol availability (WS always enabled, HTTP based on docs.NoHTTP)
func (g *GeneratorImpl) AddHandlerType(name string, req any, resp any, docs MethodDocs) {
	if _, exists := g.d.Methods[name]; exists {
		g.fatalIfErr(errors.New("method already registered: " + name))
	}

	if err := docs.Validate(); err != nil {
		g.fatalIfErr(err)
	}

	for idx, ex := range docs.Examples {
		docs.Examples[idx].Result = string(utils.MustToJSONIndent(ex.ResultObj))
		docs.Examples[idx].Params = string(utils.MustToJSONIndent(ex.ParamsObj))
	}

	docs.NoNilSlices()

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
}

// registerType registers a type with its JSON instance and JSON schema.
// Creates a new type entry if it doesn't exist, or updates an existing one.
// Parses the Go type to extract AST information for documentation.
func (g *GeneratorImpl) registerType(name string, v any) {
	if name == "null" {
		return
	}

	// Check if type already exists
	if docs, exists := g.d.Types[name]; exists {
		// Type already registered, just verify we don't overwrite
		if docs.JsonRepresentation != "" {
			g.l.Debug("Type already registered with JSON instance", slog.String("type", name))
			return
		}
	}

	g.l.Debug("Registering type", slog.String("type", name))
	_, exists := g.tsParser.Node(name)
	if !exists {
		g.fatalIfErr(fmt.Errorf("type %s not found in TypeScript AST", name))
	}
	// FIXME: find a way to serialize a single node

	// Generate JSON schema from Go type
	jsonSchema, schemaRef, err := g.getJsonSchema(name, v)
	g.fatalIfErr(err)

	// Store schema in registry for later reference extraction
	g.schemaRegistry[name] = schemaRef

	// Extract references from schema
	references := g.extractReferencesFromSchema(schemaRef)

	// Create new type docs with JSON instance and schema
	typeDocs := TypeDocs{
		Description:        schemaRef.Value.Description,
		JsonRepresentation: string(utils.MustToJSONIndent(v)),
		JsonSchema:         jsonSchema,
		References:         references,
		// TSType:             tsType, // Insert TS type here
	}

	g.d.Types[name] = typeDocs
}

// extractReferencesFromSchema finds all type references in a schema.
// Returns a list of referenced type names.
func (g *GeneratorImpl) extractReferencesFromSchema(schemaRef *openapi3.SchemaRef) []string {
	if schemaRef == nil || schemaRef.Value == nil {
		return nil
	}

	refs := make(map[string]struct{})
	g.collectReferences(schemaRef, refs)

	// Convert to sorted slice
	refList := make([]string, 0, len(refs))
	for ref := range refs {
		refList = append(refList, ref)
	}
	sort.Strings(refList)

	return refList
}

// collectReferences recursively collects all $ref entries in a schema.
func (g *GeneratorImpl) collectReferences(schemaRef *openapi3.SchemaRef, refs map[string]struct{}) {
	if schemaRef == nil {
		return
	}

	// Check if this is a reference
	if schemaRef.Ref != "" {
		typeName := extractTypeNameFromRef(schemaRef.Ref)
		refs[typeName] = struct{}{}
		return
	}

	// No value, nothing to recurse
	if schemaRef.Value == nil {
		return
	}

	schema := schemaRef.Value

	// Collect from properties
	for _, propSchemaRef := range schema.Properties {
		g.collectReferences(propSchemaRef, refs)
	}

	// Collect from array items
	if schema.Items != nil {
		g.collectReferences(schema.Items, refs)
	}

	for _, item := range []openapi3.SchemaRefs{schema.OneOf, schema.AnyOf, schema.AllOf} {
		for _, s := range item {
			g.collectReferences(s, refs)
		}
	}
}

// extractTypeNameFromRef extracts the type name from a $ref like "#/components/schemas/UserType".
func extractTypeNameFromRef(ref string) string {
	parts := strings.Split(ref, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return ref
}

// fatalIfErr logs the error and exits the program if err is not nil.
// This is used for unrecoverable errors during generator setup.
func (g *GeneratorImpl) fatalIfErr(err error) {
	if err == nil {
		return
	}

	g.l.Error("generator error", utils.ErrAttr(err))
	os.Exit(1)
}

// mustGetTypeName extracts the type name from a value.
// It validates that the value is a named struct (not an anonymous struct or primitive type).
// Returns "null" for empty struct{} values (representing no params/result).
// Panics via fatalIfErr if the value is not a valid named struct.
func (g *GeneratorImpl) mustGetTypeName(v any) string {
	// Handle nil
	if v == nil {
		g.fatalIfErr(errors.New("type must be a named struct, got: nil"))
	}

	// This is cases where there are no params or result
	if v == struct{}{} {
		return "null"
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

// isNamedStruct checks if a type is a named struct (has a type name).
// Returns true for types like "type User struct { ... }" but false for anonymous structs.
// Handles pointer types by checking the underlying element type.
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
