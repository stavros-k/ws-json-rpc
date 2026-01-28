package generate

// This file (guts.go) handles Go AST parsing and metadata extraction
// using native Go AST parser to extract type information and generate
// Go source representations with full metadata.

import (
	"bytes"
	"errors"
	"fmt"
	"go/ast"
	"go/format"
	"go/printer"
	"go/token"
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
	"golang.org/x/tools/go/packages"
)

// Sentinel errors for specific cases.
var (
	ErrEmptyConstBlock = errors.New("empty const block")
	ErrMixedTypes      = errors.New("mixed types in const block")
	ErrNoEnumType      = errors.New("no enum type found in const block")
	ErrFieldSkipped    = errors.New("field skipped")
)

// GoParser holds the parsed Go AST and type information.
type GoParser struct {
	fset  *token.FileSet
	files []*ast.File
	pkg   *packages.Package
}

type TSParser struct {
	ts *guts.Typescript
	vm *bindings.Bindings
}

// OpenAPICollector handles Go AST parsing and metadata extraction from Go types.
// It walks the Go AST to extract comprehensive type information in a single pass.
type OpenAPICollector struct {
	goParser            *GoParser
	tsParser            *TSParser
	externalTypeFormats map[string]string
	l                   *slog.Logger

	types             map[string]*TypeInfo             // Extracted type information, keyed by type name
	httpOps           map[string]*RouteInfo            // Registered HTTP operations, keyed by operationID
	mqttPublications  map[string]*MQTTPublicationInfo  // Registered MQTT publications, keyed by operationID
	mqttSubscriptions map[string]*MQTTSubscriptionInfo // Registered MQTT subscriptions, keyed by operationID
	database          Database                         // Database schema and stats

	// AST nodes for generating Go source representations
	typeASTs  map[string]*ast.GenDecl // Type declaration AST nodes, keyed by type name
	constASTs map[string]*ast.GenDecl // Const block AST nodes for enums, keyed by type name

	docsFilePath        string // Path to write documentation JSON file
	openAPISpecFilePath string // Path to write OpenAPI YAML file

	apiInfo     APIInfo
	openapiSpec string
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

	docCollector := &OpenAPICollector{
		l:                 l,
		types:             make(map[string]*TypeInfo),
		httpOps:           make(map[string]*RouteInfo),
		mqttPublications:  make(map[string]*MQTTPublicationInfo),
		mqttSubscriptions: make(map[string]*MQTTSubscriptionInfo),
		typeASTs:          make(map[string]*ast.GenDecl),
		constASTs:         make(map[string]*ast.GenDecl),
		externalTypeFormats: map[string]string{
			"time.Time":                         "date-time",
			"ws-json-rpc/backend/pkg/types.URL": "uri",
		},
		docsFilePath:        opts.DocsFileOutputPath,
		openAPISpecFilePath: opts.OpenAPISpecOutputPath,
		apiInfo:             opts.APIInfo,
	}

	dbSchema, err := docCollector.GenerateDatabaseSchema(opts.DatabaseSchemaFileOutputPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get database schema: %w", err)
	}

	docCollector.database.Schema = dbSchema

	dbStats, err := docCollector.GetDatabaseStats(dbSchema)
	if err != nil {
		return nil, fmt.Errorf("failed to get database stats: %w", err)
	}

	docCollector.database.TableCount = dbStats.TableCount

	goParser, err := docCollector.parseGoTypesDir(goTypesDirPath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Go types directory: %w", err)
	}

	docCollector.goParser = goParser

	tsParser, err := newTSParser(l, goTypesDirPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create TypeScript parser: %w", err)
	}

	docCollector.tsParser = tsParser

	// Walk the AST and extract all type information in one pass
	if err := docCollector.extractAllTypesFromGo(goParser); err != nil {
		return nil, fmt.Errorf("failed to extract types: %w", err)
	}

	l.Info("OpenAPI collector created successfully", slog.Int("types", len(docCollector.types)))

	return docCollector, nil
}

// newTSParser creates a TypeScript parser using guts for the specified Go types directory.
func newTSParser(l *slog.Logger, goTypesDirPath string) (*TSParser, error) {
	l.Debug("Parsing Go types directory", slog.String("path", goTypesDirPath))

	goParser, err := guts.NewGolangParser()
	if err != nil {
		return nil, fmt.Errorf("failed to create guts parser: %w", err)
	}

	goParser.PreserveComments()
	goParser.IncludeCustomDeclaration(map[string]guts.TypeOverride{
		"time.Time": func() bindings.ExpressionType {
			return utils.Ptr(bindings.KeywordString)
		},
		"ws-json-rpc/backend/pkg/types.URL": func() bindings.ExpressionType {
			return utils.Ptr(bindings.KeywordString)
		},
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

	vm, err := bindings.New()
	if err != nil {
		return nil, fmt.Errorf("failed to create bindings: %w", err)
	}

	tsParser := &TSParser{
		ts: ts,
		vm: vm,
	}

	return tsParser, nil
}

func (g *OpenAPICollector) SerializeNode(name string) (string, error) {
	node, exists := g.tsParser.ts.Node(name)
	if !exists {
		return "", fmt.Errorf("type %s not found in TypeScript AST", name)
	}

	tsNode, err := g.tsParser.vm.ToTypescriptNode(node)
	if err != nil {
		return "", fmt.Errorf("failed to convert node to TypeScript node: %w", err)
	}

	serialized, err := g.tsParser.vm.SerializeToTypescript(tsNode)
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

// Generate generates both the OpenAPI spec YAML and the docs JSON file.
func (g *OpenAPICollector) Generate() error {
	// Compute type relationships
	g.computeTypeRelationships()

	// Generate type representations
	if err := g.generateTypesRepresentations(); err != nil {
		return fmt.Errorf("failed to generate types representations: %w", err)
	}

	// Write OpenAPI spec
	if err := g.writeSpecYAML(g.openAPISpecFilePath); err != nil {
		return fmt.Errorf("failed to write OpenAPI spec: %w", err)
	}

	// read the written OpenAPI spec file
	yamlBytes, err := os.ReadFile(g.openAPISpecFilePath)
	if err != nil {
		return fmt.Errorf("failed to read OpenAPI spec file: %w", err)
	}

	g.openapiSpec = string(yamlBytes)

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

// RegisterJSONRepresentation registers the JSON representation of a type value.
// It makes sure to only store the largest representation for the type.
func (g *OpenAPICollector) RegisterJSONRepresentation(value any) error {
	typeName, err := extractTypeNameFromValue(value)
	if err != nil {
		return fmt.Errorf("failed to extract type name: %w", err)
	}

	typeInfo, ok := g.types[typeName]
	if !ok {
		return fmt.Errorf("type %s not found", typeName)
	}

	representation := string(utils.MustToJSONIndent(value))

	// If stored representation is empty or shorter, update it
	if typeInfo.Representations.JSON == "" || len(representation) > len(typeInfo.Representations.JSON) {
		typeInfo.Representations.JSON = representation
	}

	return nil
}

func (g *OpenAPICollector) RegisterRoute(route *RouteInfo) error {
	// Validate operationID is unique
	if _, exists := g.httpOps[route.OperationID]; exists {
		return fmt.Errorf("duplicate operationID: %s", route.OperationID)
	}

	// Extract type names from zero values using reflection, and stringify examples
	if route.Request != nil {
		if reflect.ValueOf(route.Request.TypeValue).IsZero() {
			return fmt.Errorf("Request Type must not be zero value in route [%s]", route.OperationID)
		}

		typeName, err := extractTypeNameFromValue(route.Request.TypeValue)
		if err != nil {
			return fmt.Errorf("failed to extract request type name: %w", err)
		}

		route.Request.TypeName = typeName

		// Mark as used by HTTP (for OpenAPI spec filtering)
		g.markTypeAsHTTP(typeName)

		if err := g.RegisterJSONRepresentation(route.Request.TypeValue); err != nil {
			return fmt.Errorf("failed to register JSON representation for request type: %w", err)
		}

		if err := g.registerExamples(route.Request.Examples); err != nil {
			return fmt.Errorf("failed to register JSON representation for request example: %w", err)
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

		// Mark as used by HTTP (for OpenAPI spec filtering)
		g.markTypeAsHTTP(typeName)

		if err := g.RegisterJSONRepresentation(resp.TypeValue); err != nil {
			return fmt.Errorf("failed to register JSON representation for response type: %w", err)
		}

		if err := g.registerExamples(resp.Examples); err != nil {
			return fmt.Errorf("failed to register JSON representation for response example: %w", err)
		}

		route.Responses[statusCode] = stringifyResponseExamples(resp)
	}

	for i := range route.Parameters {
		typeName, err := extractTypeNameFromValue(route.Parameters[i].TypeValue)
		if err != nil {
			return fmt.Errorf("failed to extract parameter type name: %w", err)
		}

		route.Parameters[i].TypeName = typeName

		// Mark as used by HTTP (for OpenAPI spec filtering)
		g.markTypeAsHTTP(typeName)
	}

	// Add operation keyed by operationID
	g.httpOps[route.OperationID] = route

	return nil
}

// stringifyMQTTExamples converts MQTT examples to stringified JSON.
func stringifyMQTTExamples(examples map[string]any) map[string]string {
	stringified := make(map[string]string)
	for name, example := range examples {
		stringified[name] = string(utils.MustToJSONIndent(example))
	}

	return stringified
}

func (g *OpenAPICollector) RegisterMQTTPublication(pub *MQTTPublicationInfo) error {
	// Validate operationID is unique
	if err := g.validateUniqueOperationID(pub.OperationID); err != nil {
		return err
	}

	if _, exists := g.mqttPublications[pub.OperationID]; exists {
		return fmt.Errorf("duplicate MQTT publication operationID: %s", pub.OperationID)
	}

	// Extract type name from zero value using reflection
	if reflect.ValueOf(pub.TypeValue).IsZero() {
		return fmt.Errorf("MessageType must not be zero value in publication [%s]", pub.OperationID)
	}

	typeName, err := extractTypeNameFromValue(pub.TypeValue)
	if err != nil {
		return fmt.Errorf("failed to extract message type name: %w", err)
	}

	pub.TypeName = typeName

	// Mark as used by MQTT
	g.markTypeAsMQTT(typeName)

	// Register JSON representation
	if err := g.RegisterJSONRepresentation(pub.TypeValue); err != nil {
		return fmt.Errorf("failed to register JSON representation for message type: %w", err)
	}

	// Register examples
	if err := g.registerExamples(pub.Examples); err != nil {
		return fmt.Errorf("failed to register JSON representation for example: %w", err)
	}

	// Stringify examples
	pub.ExamplesStringified = stringifyMQTTExamples(pub.Examples)

	// Store publication
	g.mqttPublications[pub.OperationID] = pub

	return nil
}

func (g *OpenAPICollector) RegisterMQTTSubscription(sub *MQTTSubscriptionInfo) error {
	// Validate operationID is unique
	if err := g.validateUniqueOperationID(sub.OperationID); err != nil {
		return err
	}

	if _, exists := g.mqttSubscriptions[sub.OperationID]; exists {
		return fmt.Errorf("duplicate MQTT subscription operationID: %s", sub.OperationID)
	}

	// Extract type name from zero value using reflection
	if reflect.ValueOf(sub.TypeValue).IsZero() {
		return fmt.Errorf("MessageType must not be zero value in subscription [%s]", sub.OperationID)
	}

	typeName, err := extractTypeNameFromValue(sub.TypeValue)
	if err != nil {
		return fmt.Errorf("failed to extract message type name: %w", err)
	}

	sub.TypeName = typeName

	// Mark as used by MQTT
	g.markTypeAsMQTT(typeName)

	// Register JSON representation
	if err := g.RegisterJSONRepresentation(sub.TypeValue); err != nil {
		return fmt.Errorf("failed to register JSON representation for message type: %w", err)
	}

	// Register examples
	if err := g.registerExamples(sub.Examples); err != nil {
		return fmt.Errorf("failed to register JSON representation for example: %w", err)
	}

	// Stringify examples
	sub.ExamplesStringified = stringifyMQTTExamples(sub.Examples)

	// Store subscription
	g.mqttSubscriptions[sub.OperationID] = sub

	return nil
}

// registerExamples registers JSON representations for a slice of examples.
func (g *OpenAPICollector) registerExamples(examples map[string]any) error {
	for _, ex := range examples {
		if err := g.RegisterJSONRepresentation(ex); err != nil {
			return err
		}
	}

	return nil
}

// validateUniqueOperationID checks that an operationID is not already used.
func (g *OpenAPICollector) validateUniqueOperationID(operationID string) error {
	if _, exists := g.mqttPublications[operationID]; exists {
		return fmt.Errorf("duplicate operationID (MQTT publication exists): %s", operationID)
	}

	if _, exists := g.mqttSubscriptions[operationID]; exists {
		return fmt.Errorf("duplicate operationID (MQTT subscription exists): %s", operationID)
	}

	if _, exists := g.httpOps[operationID]; exists {
		return fmt.Errorf("duplicate operationID (HTTP operation exists): %s", operationID)
	}

	return nil
}

// markTypeAsHTTP recursively marks a type and all its referenced types as used by HTTP.
func (g *OpenAPICollector) markTypeAsHTTP(typeName string) {
	if typeName == "" {
		return
	}

	typeInfo, exists := g.types[typeName]
	if !exists {
		return // Primitive or external type
	}

	// Skip if already marked
	if typeInfo.UsedByHTTP {
		return
	}

	// Mark this type
	typeInfo.UsedByHTTP = true

	// Recursively mark referenced types
	for _, ref := range typeInfo.References {
		g.markTypeAsHTTP(ref)
	}
}

// markTypeAsMQTT recursively marks a type and all its referenced types as used by MQTT.
func (g *OpenAPICollector) markTypeAsMQTT(typeName string) {
	if typeName == "" {
		return
	}

	typeInfo, exists := g.types[typeName]
	if !exists {
		return // Primitive or external type
	}

	// Skip if already marked
	if typeInfo.UsedByMQTT {
		return
	}

	// Mark this type
	typeInfo.UsedByMQTT = true

	// Recursively mark referenced types
	for _, ref := range typeInfo.References {
		g.markTypeAsMQTT(ref)
	}
}

// ExternalTypeInfo holds metadata about external Go types.
type ExternalTypeInfo struct {
	GoType        string // Original Go type (e.g., "time.Time")
	OpenAPIFormat string // OpenAPI format (e.g., "date-time")
}

// parseGoTypesDir parses Go type definitions from a directory using go/packages.
func (g *OpenAPICollector) parseGoTypesDir(goTypesDirPath string) (*GoParser, error) {
	g.l.Debug("Parsing Go types directory", slog.String("path", goTypesDirPath))

	if _, err := os.Stat(goTypesDirPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("go types dir path %s does not exist", goTypesDirPath)
	}

	// Use go/packages to load and type-check the package
	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedFiles | packages.NeedSyntax |
			packages.NeedTypes | packages.NeedTypesInfo,
		Dir: goTypesDirPath,
	}

	pkgs, err := packages.Load(cfg, ".")
	if err != nil {
		return nil, fmt.Errorf("failed to load package: %w", err)
	}

	if len(pkgs) == 0 {
		return nil, fmt.Errorf("no packages found in directory: %s", goTypesDirPath)
	}

	pkg := pkgs[0]

	// Fail if there are any errors in the loaded package
	if len(pkg.Errors) > 0 {
		var errMsgs []string
		for _, e := range pkg.Errors {
			errMsgs = append(errMsgs, e.Error())
		}

		return nil, fmt.Errorf("package loading failed with errors: %s", strings.Join(errMsgs, "; "))
	}

	g.l.Debug("Go types parsed successfully",
		slog.String("package", pkg.Name),
		slog.Int("fileCount", len(pkg.Syntax)))

	return &GoParser{
		fset:  pkg.Fset,
		files: pkg.Syntax,
		pkg:   pkg,
	}, nil
}

// extractAllTypesFromGo walks the Go AST and extracts all type information in one pass.
func (g *OpenAPICollector) extractAllTypesFromGo(goParser *GoParser) error {
	g.l.Debug("Starting type extraction from Go AST")

	var errs []error

	// Walk all files and extract type declarations
	for _, file := range goParser.files {
		// First pass: extract all type declarations
		for _, decl := range file.Decls {
			genDecl, ok := decl.(*ast.GenDecl)
			if !ok || genDecl.Tok != token.TYPE {
				continue
			}

			for _, spec := range genDecl.Specs {
				typeSpec, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}

				typeName := typeSpec.Name.Name

				typeInfo, err := g.extractTypeFromSpec(typeName, typeSpec, genDecl)
				if err != nil {
					errs = append(errs, fmt.Errorf("failed to extract type %s: %w", typeName, err))

					continue
				}

				g.types[typeName] = typeInfo
			}
		}

		// Second pass: extract enums from const blocks
		for _, decl := range file.Decls {
			genDecl, ok := decl.(*ast.GenDecl)
			if !ok || genDecl.Tok != token.CONST {
				continue
			}

			// Try to extract enum from this const block
			// Skip non-enum const blocks silently (they return specific errors we can ignore)
			if err := g.extractEnumsFromConstBlock(genDecl); err != nil {
				// Only skip if it's a known non-enum pattern
				if errors.Is(err, ErrEmptyConstBlock) ||
					errors.Is(err, ErrMixedTypes) ||
					errors.Is(err, ErrNoEnumType) {
					continue
				}
				// All other errors are real problems
				errs = append(errs, fmt.Errorf("failed to process const block: %w", err))
			}
		}
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	g.l.Debug("Completed type extraction", slog.Int("typeCount", len(g.types)))

	return nil
}

// extractTypeFromSpec extracts TypeInfo from a Go type spec.
func (g *OpenAPICollector) extractTypeFromSpec(name string, typeSpec *ast.TypeSpec, genDecl *ast.GenDecl) (*TypeInfo, error) {
	g.l.Debug("Extracting type", slog.String("name", name))

	// Store the AST node for later Go source generation
	g.typeASTs[name] = genDecl

	// Extract comments
	desc := g.extractCommentsFromDoc(genDecl.Doc)

	deprecated, cleanedDesc, err := g.parseDeprecation(desc)
	if err != nil {
		return nil, fmt.Errorf("failed to parse deprecation info for type %s: %w", name, err)
	}

	typeInfo := &TypeInfo{
		Name:        name,
		Description: cleanedDesc,
		Deprecated:  deprecated,
	}

	// Determine type based on the type expression
	switch t := typeSpec.Type.(type) {
	case *ast.StructType:
		return g.extractStructType(name, t, typeInfo)
	case *ast.Ident:
		// Type alias to another type (e.g., type MyString string)
		typeInfo.Kind = TypeKindAlias

		return typeInfo, nil
	case *ast.ArrayType, *ast.MapType:
		// Arrays and maps as top-level types are treated as aliases
		typeInfo.Kind = TypeKindAlias

		return typeInfo, nil
	case *ast.InterfaceType:
		if len(t.Methods.List) > 0 {
			return nil, fmt.Errorf("interface type %s with methods is not supported - use structs for API types", name)
		}

		typeInfo.Kind = TypeKindAlias

		return typeInfo, nil
	case *ast.FuncType:
		return nil, fmt.Errorf("function type %s is not supported for API types", name)
	case *ast.ChanType:
		return nil, fmt.Errorf("channel type %s is not supported for API types", name)
	default:
		return nil, fmt.Errorf("unsupported type %s: %T (please use struct, type alias, or basic types)", name, typeSpec.Type)
	}
}

// extractStructType extracts struct type information.
func (g *OpenAPICollector) extractStructType(name string, structType *ast.StructType, typeInfo *TypeInfo) (*TypeInfo, error) {
	typeInfo.Kind = TypeKindObject
	typeInfo.Fields = []FieldInfo{}
	typeInfo.References = []string{}

	refs := make(map[string]struct{})

	for _, field := range structType.Fields.List {
		if len(field.Names) == 0 {
			return nil, fmt.Errorf("field of struct type %s has no name", name)
		}

		for _, fieldName := range field.Names {
			if !fieldName.IsExported() {
				// Skip unexported fields
				continue
			}

			fieldInfo, fieldRefs, err := g.extractFieldInfo(name, fieldName.Name, field)
			if err != nil {
				// Skip fields that should be ignored (e.g., json:"-" tags)
				if errors.Is(err, ErrFieldSkipped) {
					continue
				}

				return nil, err
			}

			typeInfo.Fields = append(typeInfo.Fields, fieldInfo)

			// Collect references
			for _, ref := range fieldRefs {
				refs[ref] = struct{}{}
			}
		}
	}

	// Convert refs to sorted slice
	for ref := range refs {
		typeInfo.References = append(typeInfo.References, ref)
	}

	sort.Strings(typeInfo.References)

	g.l.Debug("Extracted struct type", slog.String("name", name), slog.Int("fieldCount", len(typeInfo.Fields)))

	return typeInfo, nil
}

// extractFieldInfo extracts field information from a struct field.
func (g *OpenAPICollector) extractFieldInfo(parentName, fieldName string, field *ast.Field) (FieldInfo, []string, error) {
	// Extract JSON tag
	jsonName := fieldName
	omitempty := false
	jsonSkip := false

	if field.Tag != nil {
		tag := strings.Trim(field.Tag.Value, "`")
		// Simple JSON tag parsing
		if strings.Contains(tag, "json:") {
			parts := strings.Split(tag, "json:")
			if len(parts) > 1 {
				jsonTag := strings.Split(parts[1], "\"")[1]

				jsonParts := strings.Split(jsonTag, ",")
				if jsonParts[0] == "-" {
					jsonSkip = true
				} else if jsonParts[0] != "" {
					jsonName = jsonParts[0]
				}

				for _, part := range jsonParts[1:] {
					if part == "omitempty" {
						omitempty = true
					}
				}
			}
		}
	}

	if jsonSkip {
		return FieldInfo{}, nil, ErrFieldSkipped
	}

	// Analyze field type
	fieldType, refs, err := g.analyzeGoType(field.Type)
	if err != nil {
		return FieldInfo{}, nil, fmt.Errorf("failed to analyze field type for %s.%s (type: %T): %w", parentName, fieldName, field.Type, err)
	}

	// Determine if field is required
	// In Go: pointer types (*T) are optional, non-pointer types are required unless omitempty is set
	required := !fieldType.Nullable && !omitempty
	fieldType.Required = required

	// Extract field documentation
	fieldDesc := g.extractCommentsFromDoc(field.Doc)

	fieldDeprecated, cleanedFieldDesc, err := g.parseDeprecation(fieldDesc)
	if err != nil {
		return FieldInfo{}, nil, fmt.Errorf("failed to parse deprecation info for field %s.%s: %w", parentName, fieldName, err)
	}

	fieldInfo := FieldInfo{
		Name:        jsonName,
		DisplayType: generateDisplayType(fieldType),
		TypeInfo:    fieldType,
		Description: cleanedFieldDesc,
		Deprecated:  fieldDeprecated,
	}

	return fieldInfo, refs, nil
}

// extractEnumsFromConstBlock extracts enum values from a const block.
func (g *OpenAPICollector) extractEnumsFromConstBlock(constDecl *ast.GenDecl) error {
	if len(constDecl.Specs) == 0 {
		return ErrEmptyConstBlock
	}

	// Check if this looks like an enum (all values are of the same type)
	var (
		enumTypeName string
		enumValues   []EnumValue
	)

	hasExportedConsts := false

	for _, spec := range constDecl.Specs {
		valueSpec, ok := spec.(*ast.ValueSpec)
		if !ok {
			continue
		}

		// Get the type
		if valueSpec.Type != nil {
			if ident, ok := valueSpec.Type.(*ast.Ident); ok {
				if enumTypeName == "" {
					enumTypeName = ident.Name
				} else if enumTypeName != ident.Name {
					// Mixed types - not an enum
					return ErrMixedTypes
				}
			}
		}

		// Extract values
		for i, name := range valueSpec.Names {
			if !name.IsExported() {
				continue
			}

			hasExportedConsts = true

			// Get the value
			if i < len(valueSpec.Values) {
				basicLit, ok := valueSpec.Values[i].(*ast.BasicLit)
				if !ok {
					// Exported const without a literal value - this is a problem for enums
					if enumTypeName != "" {
						return fmt.Errorf("enum constant %s.%s must have a string literal value, got %T", enumTypeName, name.Name, valueSpec.Values[i])
					}

					continue
				}

				if basicLit.Kind != token.STRING {
					// If we already identified an enum type, non-string values are an error
					if enumTypeName != "" {
						return fmt.Errorf("enum constant %s.%s must be a string, got %v", enumTypeName, name.Name, basicLit.Kind)
					}

					continue
				}

				value := strings.Trim(basicLit.Value, "\"")

				// Extract documentation
				desc := ""
				if valueSpec.Doc != nil {
					desc = g.extractCommentsFromDoc(valueSpec.Doc)
				} else if valueSpec.Comment != nil {
					desc = g.extractCommentsFromDoc(valueSpec.Comment)
				}

				deprecated, cleanedDesc, err := g.parseDeprecation(desc)
				if err != nil {
					return fmt.Errorf("failed to parse deprecation for enum value %s.%s: %w", enumTypeName, name.Name, err)
				}

				enumValues = append(enumValues, EnumValue{
					Value:       value,
					Description: cleanedDesc,
					Deprecated:  deprecated,
				})
			} else if enumTypeName != "" {
				// Exported const in an enum type without a value
				return fmt.Errorf("enum constant %s.%s is missing a value", enumTypeName, name.Name)
			}
		}
	}

	if enumTypeName == "" || len(enumValues) == 0 {
		return ErrNoEnumType
	}

	// Verify we have at least one enum value if we identified an enum type
	if hasExportedConsts && len(enumValues) == 0 && enumTypeName != "" {
		return fmt.Errorf("enum type %s has exported constants but no valid string values", enumTypeName)
	}

	// Store the const block AST node for later Go source generation
	g.constASTs[enumTypeName] = constDecl

	// Check if we already have this type (from type declaration)
	existingType, exists := g.types[enumTypeName]
	if exists {
		// Update existing type to be an enum
		existingType.Kind = TypeKindStringEnum
		existingType.EnumValues = enumValues
		g.l.Debug("Updated type to enum", slog.String("name", enumTypeName), slog.Int("valueCount", len(enumValues)))
	} else {
		// Create new enum type
		g.types[enumTypeName] = &TypeInfo{
			Name:       enumTypeName,
			Kind:       TypeKindStringEnum,
			EnumValues: enumValues,
		}
		g.l.Debug("Created new enum type", slog.String("name", enumTypeName), slog.Int("valueCount", len(enumValues)))
	}

	return nil
}

// analyzeGoType analyzes a Go type expression and returns FieldType and referenced types.
func (g *OpenAPICollector) analyzeGoType(expr ast.Expr) (FieldType, []string, error) {
	refs := []string{}

	switch t := expr.(type) {
	case *ast.Ident:
		// Simple type reference (string, int, MyType, etc.)
		typeName := t.Name

		// Check for primitives
		switch typeName {
		case "string", "bool", "int", "int8", "int16", "int32", "int64",
			"uint", "uint8", "uint16", "uint32", "uint64",
			"float32", "float64", "byte", "rune":
			return FieldType{
				Kind: FieldKindPrimitive,
				Type: typeName,
			}, refs, nil
		case "any", "interface{}":
			g.l.Warn("Using 'any' or 'interface{} is discouraged", slog.String("type", typeName))

			return FieldType{
				Kind: FieldKindObject,
				Type: "object",
			}, refs, nil
		}

		// Check if it's a defined type in our types map (will be populated after first pass)
		// For now, treat as reference
		refs = append(refs, typeName)

		return FieldType{
			Kind: FieldKindReference,
			Type: typeName,
		}, refs, nil

	case *ast.StarExpr:
		// Pointer type (*T) - nullable
		inner, innerRefs, err := g.analyzeGoType(t.X)
		if err != nil {
			return FieldType{}, nil, err
		}

		inner.Nullable = true

		return inner, innerRefs, nil

	case *ast.ArrayType:
		// Array or slice ([]T)
		elemType, elemRefs, err := g.analyzeGoType(t.Elt)
		if err != nil {
			return FieldType{}, nil, err
		}

		return FieldType{
			Kind:      FieldKindArray,
			Type:      "array",
			ItemsType: &elemType,
		}, elemRefs, nil

	case *ast.MapType:
		// Map type (map[K]V)
		valueType, valueRefs, err := g.analyzeGoType(t.Value)
		if err != nil {
			return FieldType{}, nil, err
		}

		return FieldType{
			Kind:                 FieldKindObject,
			Type:                 "object",
			AdditionalProperties: &valueType,
		}, valueRefs, nil

	case *ast.SelectorExpr:
		// External type (e.g., time.Time, types.URL)
		pkgIdent, ok := t.X.(*ast.Ident)
		if !ok {
			return FieldType{}, nil, fmt.Errorf("unsupported selector expression with base type %T - expected package.Type format", t.X)
		}

		fullType := pkgIdent.Name + "." + t.Sel.Name

		// Check for known external types
		format, exists := g.externalTypeFormats[fullType]
		if !exists {
			// Try with full package path
			fullTypePath := "ws-json-rpc/backend/pkg/" + pkgIdent.Name + "." + t.Sel.Name
			format, exists = g.externalTypeFormats[fullTypePath]

			if !exists {
				return FieldType{}, nil, fmt.Errorf("unknown external type %s - please add it to externalTypeFormats map in NewOpenAPICollector", fullType)
			}
		}

		return FieldType{
			Kind:   FieldKindPrimitive,
			Type:   "string",
			Format: format,
		}, refs, nil

	case *ast.InterfaceType:
		// Interface type (interface{} or any)
		if len(t.Methods.List) > 0 {
			return FieldType{}, nil, errors.New("interface types with methods are not supported - please use concrete types or any/interface{}")
		}

		return FieldType{
			Kind: FieldKindObject,
			Type: "object",
		}, refs, nil

	default:
		return FieldType{}, nil, fmt.Errorf("unsupported type expression: %T", expr)
	}
}

// extractCommentsFromDoc extracts text from a comment group.
func (g *OpenAPICollector) extractCommentsFromDoc(doc *ast.CommentGroup) string {
	if doc == nil {
		return ""
	}

	var builder strings.Builder

	for i, comment := range doc.List {
		if i > 0 {
			builder.WriteString(" ")
		}
		// Remove // or /* */ prefix/suffix
		text := strings.TrimPrefix(comment.Text, "//")
		text = strings.TrimPrefix(text, "/*")
		text = strings.TrimSuffix(text, "*/")
		builder.WriteString(strings.TrimSpace(text))
	}

	return builder.String()
}

func (g *OpenAPICollector) getDocumentation() *APIDocumentation {
	return &APIDocumentation{
		Types:             g.types,
		HTTPOperations:    g.httpOps,
		MQTTPublications:  g.mqttPublications,
		MQTTSubscriptions: g.mqttSubscriptions,
		Database:          g.database,
		Info:              g.apiInfo,
		OpenAPISpec:       g.openapiSpec,
	}
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

// generateGoRepresentations generates Go source representations for all types as a post-processing step.
func (g *OpenAPICollector) generateTypesRepresentations() error {
	g.l.Debug("Generating all representations for all types", slog.Int("typeCount", len(g.types)))

	for name, typeInfo := range g.types {
		// Go Representation
		goSource, err := g.generateGoSource(typeInfo)
		if err != nil {
			return fmt.Errorf("failed to generate Go representation for %s: %w", name, err)
		}

		typeInfo.Representations.Go = goSource

		// TypeScript Representation
		tsSource, err := g.SerializeNode(name)
		if err != nil {
			return fmt.Errorf("failed to serialize TS representation for %s: %w", name, err)
		}

		typeInfo.Representations.TS = tsSource

		// JSON Schema Representation
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

	g.l.Debug("All representations generated successfully")

	return nil
}

// printASTNode prints an AST node to a buffer using go/printer.
func (g *OpenAPICollector) printASTNode(buf *bytes.Buffer, node *ast.GenDecl, typeName, nodeType string) error {
	if node == nil {
		return nil
	}

	if err := printer.Fprint(buf, g.goParser.fset, node); err != nil {
		return fmt.Errorf("failed to print %s for %s: %w", nodeType, typeName, err)
	}

	return nil
}

// generateGoSource generates Go source code for a type using the parsed AST.
func (g *OpenAPICollector) generateGoSource(typeInfo *TypeInfo) (string, error) {
	var buf bytes.Buffer

	// Look up the AST nodes by type name
	typeDecl, exists := g.typeASTs[typeInfo.Name]
	if !exists {
		return "", fmt.Errorf("no type declaration AST found for type %s", typeInfo.Name)
	}

	if err := g.printASTNode(&buf, typeDecl, typeInfo.Name, "type declaration"); err != nil {
		return "", err
	}

	buf.WriteString("\n")

	if typeInfo.Kind == TypeKindStringEnum {
		constDecl, exists := g.constASTs[typeInfo.Name]
		if !exists {
			return "", fmt.Errorf("no const declaration AST found for enum type %s", typeInfo.Name)
		}

		if err := g.printASTNode(&buf, constDecl, typeInfo.Name, "const declaration"); err != nil {
			return "", err
		}

		buf.WriteString("\n")
	}

	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		return "", fmt.Errorf("failed to format Go source: %w", err)
	}

	return string(formatted), nil
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
	// Track HTTP operations
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

	// Track MQTT publications
	for _, pub := range g.mqttPublications {
		g.addUsage(pub.TypeName, pub.OperationID, "mqtt_publication")
	}

	// Track MQTT subscriptions
	for _, sub := range g.mqttSubscriptions {
		g.addUsage(sub.TypeName, sub.OperationID, "mqtt_subscription")
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
