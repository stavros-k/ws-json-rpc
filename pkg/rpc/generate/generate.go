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
	"time"
	"ws-json-rpc/internal/database/sqlite"
	"ws-json-rpc/pkg/database"
	"ws-json-rpc/pkg/rpc/generate/typesystem"
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
// It manages schema parsing, type registration, and documentation generation.
// All schema types are registered during construction via NewGenerator.
type GeneratorImpl struct {
	l                *slog.Logger           // Logger for debugging and error reporting
	d                *Docs                  // API documentation structure
	parser           *typesystem.TypeParser // Type system parser
	goOptions        GoOptions              // Go code generation options
	tsOptions        TSOptions              // TypeScript code generation options
	csharpOptions    CSharpOptions          // C# code generation options
	docsFilePath     string                 // Output path for API docs JSON
	dbSchemaFilePath string                 // Output path for database schema SQL
	schemasDirectory string                 // Directory containing .type.json files
}

// GeneratorOptions contains all configuration needed to create a Generator.
// All paths must be provided for the generator to function properly.
type GeneratorOptions struct {
	DocsFileOutputPath    string        // Path for generated API docs JSON file
	SchemaFileOutputPath  string        // Path for generated database schema SQL file
	SchemasInputDirectory string        // Directory containing JSON schema files
	CSharpOptions         CSharpOptions // C# code generation options
	GoOptions             GoOptions     // Go code generation options
	TSOptions             TSOptions     // TypeScript code generation options
	DocsOptions           DocsOptions   // Docs options
}

// NewGenerator creates a new Generator instance with the given options.
// It performs the following initialization steps:
// 1. Validates all required options are provided
// 2. Creates a schema parser
// 3. Parses all JSON schema files
// 4. Registers all types from schemas (generating Go/TS/C# code, but not JSON instances)
//
// JSON instance representations are added later when methods/events are registered.
func NewGenerator(l *slog.Logger, opts GeneratorOptions) (Generator, error) {
	if opts.DocsFileOutputPath == "" {
		return nil, fmt.Errorf("docs file path is required")
	}
	if opts.SchemaFileOutputPath == "" {
		return nil, fmt.Errorf("schema file path is required")
	}
	if opts.SchemasInputDirectory == "" {
		return nil, fmt.Errorf("schemas directory is required")
	}

	parser := typesystem.NewTypeParser(l)

	g := &GeneratorImpl{
		l:                l.With(slog.String("component", "generator")),
		d:                NewDocs(opts.DocsOptions),
		parser:           parser,
		goOptions:        opts.GoOptions,
		tsOptions:        opts.TSOptions,
		csharpOptions:    opts.CSharpOptions,
		docsFilePath:     opts.DocsFileOutputPath,
		dbSchemaFilePath: opts.SchemaFileOutputPath,
		schemasDirectory: opts.SchemasInputDirectory,
	}

	// Parse all .type.json files and register types immediately
	if err := parser.ParseDirectory(opts.SchemasInputDirectory); err != nil {
		return nil, fmt.Errorf("failed to parse schemas: %w", err)
	}

	// Register all parsed types from definitions
	allTypes := parser.GetRegistry().GetAll()
	for name := range allTypes {
		if err := g.registerTypeFromDefinition(name); err != nil {
			return nil, fmt.Errorf("failed to register type %q from definition: %w", name, err)
		}
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

	// Generate and write type files for all languages
	if err := g.generateTypeFiles(); err != nil {
		return fmt.Errorf("failed to generate type files: %w", err)
	}

	// Generate and append TypeScript API mappings to the TypeScript output file
	if err := g.generateTypeScriptAPIMappings(); err != nil {
		return fmt.Errorf("failed to generate typescript API mappings: %w", err)
	}

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

// generateTypeFiles generates and writes complete type files for Go, TypeScript, and C#.
// This uses the parser's complete file generation methods to create properly formatted
// type definition files with all necessary imports and scaffolding.
func (g *GeneratorImpl) generateTypeFiles() error {
	// Generate Go types if output file is configured
	if g.goOptions.OutputFile != "" {
		goCode, err := g.parser.GenerateCompleteGo(g.goOptions.PackageName)
		if err != nil {
			return fmt.Errorf("failed to generate Go types: %w", err)
		}

		if err := os.WriteFile(g.goOptions.OutputFile, []byte(goCode), 0644); err != nil {
			return fmt.Errorf("failed to write Go types file: %w", err)
		}

		g.l.Info("Go types generated", slog.String("file", g.goOptions.OutputFile))
	} else {
		g.l.Info("Skipping Go type generation (no output file configured)")
	}

	// Generate TypeScript types if output file is configured
	if g.tsOptions.OutputFile != "" {
		tsCode, err := g.parser.GenerateCompleteTypeScript()
		if err != nil {
			return fmt.Errorf("failed to generate TypeScript types: %w", err)
		}

		if err := os.WriteFile(g.tsOptions.OutputFile, []byte(tsCode), 0644); err != nil {
			return fmt.Errorf("failed to write TypeScript types file: %w", err)
		}

		g.l.Info("TypeScript types generated", slog.String("file", g.tsOptions.OutputFile))
	} else {
		g.l.Info("Skipping TypeScript type generation (no output file configured)")
	}

	// Generate C# types if output file is configured
	if g.csharpOptions.OutputFile != "" {
		csharpCode, err := g.parser.GenerateCompleteCSharp(g.csharpOptions.Namespace)
		if err != nil {
			return fmt.Errorf("failed to generate C# types: %w", err)
		}

		if err := os.WriteFile(g.csharpOptions.OutputFile, []byte(csharpCode), 0644); err != nil {
			return fmt.Errorf("failed to write C# types file: %w", err)
		}

		g.l.Info("C# types generated", slog.String("file", g.csharpOptions.OutputFile))
	} else {
		g.l.Info("Skipping C# type generation (no output file configured)")
	}

	return nil
}

// generateTypeScriptAPIMappings generates and appends TypeScript API mappings to the output file.
// This creates ApiMethods and ApiEvents types for fully-typed API clients.
func (g *GeneratorImpl) generateTypeScriptAPIMappings() error {
	// Skip if no TypeScript output file is configured
	if g.tsOptions.OutputFile == "" {
		return nil
	}

	// Collect method mappings
	methods := make([]MethodMapping, 0, len(g.d.Methods))
	for name, method := range g.d.Methods {
		methods = append(methods, MethodMapping{
			Name:       name,
			ParamType:  method.ParamType.Ref,
			ResultType: method.ResultType.Ref,
		})
	}

	// Collect event mappings
	events := make([]EventMapping, 0, len(g.d.Events))
	for name, event := range g.d.Events {
		events = append(events, EventMapping{
			Name:       name,
			ResultType: event.ResultType.Ref,
		})
	}

	// Generate API mappings
	mappings := GenerateTypeScriptAPIMappings(methods, events)

	// Append to TypeScript output file
	f, err := os.OpenFile(g.tsOptions.OutputFile, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open typescript file for appending: %w", err)
	}
	defer f.Close()

	if _, err := f.WriteString("\n" + mappings); err != nil {
		return fmt.Errorf("failed to append API mappings to typescript file: %w", err)
	}

	g.l.Info("TypeScript API mappings generated", slog.String("file", g.tsOptions.OutputFile))

	return nil
}

// AddEventType registers a WebSocket event with its response type and documentation.
// Events are unidirectional server-to-client messages sent over WebSocket connections.
// The response type must be a named struct with a corresponding JSON schema file.
//
// This method:
// 1. Validates the event hasn't been registered already
// 2. Validates the event documentation
// 3. Converts example objects to JSON strings
// 4. Sets the result type reference
// 5. Sets the JSON instance representation for the result type
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

	// Set JSON instance representation (type must already be registered via Generate)
	if err := g.setTypeJsonInstance(resultTypeName, resp); err != nil {
		g.fatalIfErr(err)
	}

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
// 5. Sets JSON instance representations for both types
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

	// Set JSON instance representations (types must already be registered via Generate)
	if err := g.setTypeJsonInstance(paramTypeName, req); err != nil {
		g.fatalIfErr(err)
	}
	if err := g.setTypeJsonInstance(resultTypeName, resp); err != nil {
		g.fatalIfErr(err)
	}

	g.d.Methods[name] = docs
}

// registerTypeFromDefinition registers a type from the type system.
// Generates Go/TS/C# representations but not JSON instance representation.
// Returns an error if the type is already registered.
func (g *GeneratorImpl) registerTypeFromDefinition(name string) error {
	if name == "null" {
		return nil
	}

	if _, exists := g.d.Types[name]; exists {
		return fmt.Errorf("type %q is already registered", name)
	}

	g.l.Debug("Registering type from definition", slog.String("type", name))
	node := g.parser.GetRegistry().Get(name)
	if node == nil {
		return fmt.Errorf("type %q not found in parsed schemas", name)
	}

	// Generate standalone code with package/namespace wrappers
	tsStr, err := typesystem.ToStandaloneTypeScript(node)
	if err != nil {
		return fmt.Errorf("failed to generate typescript for type %q: %w", name, err)
	}

	goStr, err := typesystem.ToStandaloneGo(node, "rpcapi")
	if err != nil {
		return fmt.Errorf("failed to generate go for type %q: %w", name, err)
	}

	csharpStr, err := typesystem.ToStandaloneCSharp(node, "rpcapi")
	if err != nil {
		return fmt.Errorf("failed to generate csharp for type %q: %w", name, err)
	}

	// Extract metadata from type node
	typeDocs := TypeDocs{
		Description:          node.GetDescription(),
		Kind:                 string(node.GetKind()),
		JsonRepresentation:   "", // Set later via setTypeJsonInstance
		TypeDefinition:       node.GetRawDefinition(),
		GoRepresentation:     goStr,
		TsRepresentation:     tsStr,
		CsharpRepresentation: csharpStr,
	}

	// Add type-specific metadata based on node type
	switch n := node.(type) {
	case *typesystem.EnumNode:
		// Add enum values
		values := n.GetValues()
		typeDocs.EnumValues = make([]EnumValue, len(values))
		for i, val := range values {
			typeDocs.EnumValues[i] = EnumValue{
				Value:       val.Value,
				Description: val.Description,
			}
		}
		// Enums have no type references
		typeDocs.References = []string{}

	case *typesystem.AliasNode:
		// Add alias target
		targetType := n.GetTargetType()
		typeDocs.AliasTarget = targetType
		// Alias references the target type if it's a reference
		if n.IsTargetRef() {
			typeDocs.References = []string{targetType}
		} else {
			typeDocs.References = []string{}
		}

	case *typesystem.MapNode:
		// Add map metadata
		typeDocs.MapValueType = n.GetValueType()
		typeDocs.MapValueIsRef = n.IsValueRef()
		typeDocs.References = n.GetReferences()

	case *typesystem.ObjectNode:
		// Add object field metadata
		tsFields := n.GetFields()
		typeDocs.Fields = make([]FieldMetadata, len(tsFields))
		for i, field := range tsFields {
			// Determine the base type for the field
			var baseType string
			if field.Type.Ref != "" {
				// Reference to another type
				baseType = field.Type.Ref
			} else if field.Type.Primitive != "" {
				// Primitive type (string, number, integer, boolean)
				baseType = string(field.Type.Primitive)
			} else {
				// Fallback: use ToGoType() for complex types (arrays, maps)
				baseType = field.Type.ToGoType()
			}

			typeDocs.Fields[i] = FieldMetadata{
				Name:        field.Name,
				Description: field.Description,
				Type:        baseType,
				Format:      string(field.Type.Format),
				Optional:    field.Optional,
				Nullable:    field.Nullable,
				IsRef:       field.Type.Ref != "",
				RefTypeName: field.Type.Ref,
			}
		}
		typeDocs.References = n.GetReferences()
	}

	g.d.Types[name] = typeDocs

	return nil
}

// setTypeJsonInstance sets the JSON instance representation for an already-registered type.
// The type must have been registered via registerTypeFromDefinition first.
func (g *GeneratorImpl) setTypeJsonInstance(name string, v any) error {
	if name == "null" {
		return nil
	}

	docs, exists := g.d.Types[name]
	if !exists {
		return fmt.Errorf("type %q not registered, call registerTypeFromDefinition first", name)
	}

	if docs.JsonRepresentation != "" {
		return fmt.Errorf("type %q already has JSON instance representation", name)
	}

	g.l.Debug("Setting JSON instance for type", slog.String("type", name))
	docs.JsonRepresentation = string(utils.MustToJSONIndent(v))
	g.d.Types[name] = docs

	return nil
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

// FIXME: Implement this generator
// During server startup we register the types with the generator
// Then generator will parse the json schema(s) and generate:
// 1. Types for Go, TypeScript, and C#
// 2. Combine the docs for the types (ie from descriptions etc) plus the docs defined during method/event registration
// 3. Generates a json file that will be consumed by a website to display the API docs

// Example of api docs structure
// Title: Local API
// Description: This is the API documentation for the Local API.
// Version: 1.0.0
//
// Methods: {
//   "Ping": {
//     "name": "Ping",
//     "tags": ["Utility"],
//     "description": "A simple ping method to test connectivity",
//     "params": "$ref:PingParams",
//     "result": "$ref:PingResult",
//   }
// },
// Events: {
//   "DataCreated": {
//     "name": "DataCreated",
//     "tags": ["Data"],
//     "description": "Triggered when a new data is created",
//     "result": "$ref:DataCreatedResult",
//   },
// },
// Types: {
//   "PingParams": {
//     "description": "Parameters for the Ping method",
//     "goRepresentation": "struct { }", # The string representation of the Go struct
//     "tsRepresentation": "interface PingParams { }", # The string representation of the TypeScript interface
//     "csharpRepresentation": "class PingParams { }", # The string representation of the C# class
//     "jsonRepresentation": "{ }", # The string representation of the JSON object
//     "jsonSchemaRepresentation": "{ }", # The string representation of the JSON schema
//     "fields": [ # List of fields with their types and descriptions
//       {
//         "name": "field1",
//         "type": "string",
//         "description": "Description of field1"
//       },
//     ],
//   },
// },

// We might need to keep track of nested types, for example if PingResult has a field of type Status
// Then the document for the webpage should know that it should also include the Status type in the "types" section

// We can start doing most of the work, and finally when we use our own generator to generate the types, we can then
// populate the "goRepresentation", "tsRepresentation", "csharpRepresentation" fields.
// jsonRepresentation and jsonSchemaRepresentation can be populated now since we have the schema files.

// We probably shouldn't allow non-$ref types for params and result, to make it easier to generate the types and docs.
