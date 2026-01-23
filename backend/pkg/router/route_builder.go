package router

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"slices"
	"strconv"
	"strings"
	"ws-json-rpc/backend/pkg/utils"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3gen"
	"github.com/go-chi/chi/v5"
	"github.com/oasdiff/yaml"
)

// RouteBuilder is a chi router with OpenAPI support
type RouteBuilder struct {
	spec           *openapi3.T
	router         chi.Router
	gen            *openapi3gen.Generator
	l              *slog.Logger
	prefix         string
	gutsCustomizer *GutsSchemaCustomizer
}

// RouteBuilderOptions contains configuration for creating a RouteBuilder
type RouteBuilderOptions struct {
	Title          string
	Version        string
	Description    string
	TypesDirectory string // Path to Go types directory for guts metadata extraction
}

// NewRouteBuilder creates a new RouteBuilder
func NewRouteBuilder(l *slog.Logger, opts RouteBuilderOptions) (*RouteBuilder, error) {
	spec := &openapi3.T{
		OpenAPI: "3.0.3",
		Info: &openapi3.Info{
			Title:       opts.Title,
			Version:     opts.Version,
			Description: opts.Description,
		},
		Paths:      openapi3.NewPaths(),
		Components: &openapi3.Components{Schemas: make(openapi3.Schemas)},
	}

	// Create guts customizer for metadata extraction
	gutsCustomizer, err := NewGutsSchemaCustomizer(l, opts.TypesDirectory)
	if err != nil {
		return nil, fmt.Errorf("failed to create guts customizer: %w", err)
	}

	// Create OpenAPI generator
	gen := openapi3gen.NewGenerator(
		// Use guts customizer to extract metadata from Go types
		gutsCustomizer.AsOpenAPIOption(),
		openapi3gen.ThrowErrorOnCycle(),
		// Enable component schemas
		openapi3gen.CreateComponentSchemas(openapi3gen.ExportComponentSchemasOptions{
			ExportComponentSchemas: true,
			ExportTopLevelSchema:   true,
		}),
	)

	return &RouteBuilder{
		spec:           spec,
		router:         chi.NewRouter(),
		gen:            gen,
		l:              l.With(slog.String("component", "route-builder")),
		gutsCustomizer: gutsCustomizer,
	}, nil
}

// Must exits the program if an error occurs
func (rb *RouteBuilder) Must(err error) {
	if err != nil {
		rb.l.Error("Fatal error", slog.Any("error", err))
		os.Exit(1)
	}
}

// Route adds a new route group to the router
func (rb *RouteBuilder) Route(path string, fn func(rb *RouteBuilder)) *RouteBuilder {
	oldPrefix := rb.prefix
	rb.prefix += path

	// Isolate sub-router
	rb.router.Group(func(r chi.Router) {
		subRB := &RouteBuilder{
			spec:           rb.spec,
			router:         r,
			gen:            rb.gen,
			prefix:         rb.prefix,
			l:              rb.l.With(slog.String("prefix", rb.prefix)),
			gutsCustomizer: rb.gutsCustomizer,
		}
		fn(subRB)
	})

	rb.prefix = oldPrefix
	return rb
}

// Use adds middlewares to the router
func (rb *RouteBuilder) Use(middlewares ...func(http.Handler) http.Handler) *RouteBuilder {
	rb.router.Use(middlewares...)
	return rb
}

// WithServer adds a server to the OpenAPI spec
func (rb *RouteBuilder) WithServer(url, description string) *RouteBuilder {
	rb.spec.Servers = append(rb.spec.Servers, &openapi3.Server{
		URL:         url,
		Description: description,
	})
	return rb
}

// RouteSpec defines a specific route
type RouteSpec struct {
	OperationID string           // OperationID is the unique identifier for the route
	Handler     http.HandlerFunc // Handler is the function that will handle the route
	Summary     string           // Summary is a short summary of the route
	Description string           // Description is a longer description of the route
	Tags        []string         // Tags are used to group routes in the OpenAPI spec
	Deprecated  bool             // Deprecated indicates if the route is deprecated

	RequestType *RequestBodySpec     // RequestType is the type of the request body, or nil if no body
	Responses   map[int]ResponseSpec // Responses is a map of status code to response spec

	Parameters map[string]ParameterSpec // Parameters (ie query, path, etc) is a map of parameter name to parameter spec

	// Internal fields
	localPath string // localPath is the path without the prefix
	fullPath  string // fullPath is the full path with the prefix
	method    string // method is the HTTP method (e.g., GET, POST, etc.)
}

type ParameterIn string

const (
	ParameterInPath   ParameterIn = "path"
	ParameterInQuery  ParameterIn = "query"
	ParameterInHeader ParameterIn = "header"
)

// ParameterSpec defines a parameter for a route
type ParameterSpec struct {
	In          ParameterIn
	Description string
	Required    bool
	Type        any // The Go type - validation comes from interfaces
}

type RequestBodySpec struct {
	Type     any
	Examples map[string]any
}

type ResponseSpec struct {
	Description string
	Type        any
	Examples    map[string]any
}

// generateResponses generates the OpenAPI responses for a route
func (rb *RouteBuilder) generateResponses(op *openapi3.Operation, spec RouteSpec) error {
	// Regular HTTP responses
	responseCodes := map[int]struct{}{}
	op.Deprecated = spec.Deprecated

	// Generate the schemas for each response
	for statusCode, respSpec := range spec.Responses {
		if statusCode < 100 || statusCode > 599 {
			return fmt.Errorf("invalid status code %d for %s %s", statusCode, spec.method, spec.fullPath)
		}
		if _, exists := responseCodes[statusCode]; exists {
			return fmt.Errorf("duplicate status code %d for %s %s", statusCode, spec.method, spec.fullPath)
		}
		if respSpec.Description == "" {
			return fmt.Errorf("description required for status %d", statusCode)
		}

		// Keep track of response codes
		responseCodes[statusCode] = struct{}{}
		statusStr := strconv.Itoa(statusCode)

		// Response without body
		if respSpec.Type == nil {
			// No body (e.g., 204 No Content)
			op.Responses.Set(statusStr, &openapi3.ResponseRef{
				Value: &openapi3.Response{Description: ptr(respSpec.Description)},
			})
			continue
		}

		// Response with body
		respSchema, err := rb.gen.NewSchemaRefForValue(respSpec.Type, rb.spec.Components.Schemas)
		if err != nil {
			return fmt.Errorf("failed to generate response schema: %w", err)
		}

		op.Responses.Set(statusStr, &openapi3.ResponseRef{
			Value: &openapi3.Response{
				Description: ptr(respSpec.Description),
				Content:     jsonContent(respSchema, respSpec.Examples),
			},
		})
	}

	return nil
}

func examplesToOpenAPIExamples(examples map[string]any) openapi3.Examples {
	openAPIExamples := openapi3.Examples{}
	for name, value := range examples {
		openAPIExamples[name] = &openapi3.ExampleRef{
			Value: &openapi3.Example{Value: value},
		}
	}
	return openAPIExamples
}

// sanitizePath removes double slashes and trailing slashes from a path
func sanitizePath(path string) string {
	cleanPath := path
	for strings.Contains(cleanPath, "//") {
		cleanPath = strings.ReplaceAll(cleanPath, "//", "/")
	}
	cleanPath = strings.TrimSuffix(cleanPath, "/")
	if cleanPath == "" {
		cleanPath = "/"
	}
	return cleanPath
}

// validateRouteSpec validates a RouteSpec
func validateRouteSpec(spec RouteSpec) error {
	if spec.OperationID == "" {
		return fmt.Errorf("field OperationID required")
	}
	if spec.Summary == "" {
		return fmt.Errorf("field Summary required")
	}
	if spec.Description == "" {
		return fmt.Errorf("field Description required")
	}
	if len(spec.Tags) == 0 {
		return fmt.Errorf("field Tags requires at least one tag")
	}

	return nil
}

// add adds a new route to the router
func (rb *RouteBuilder) add(path string, spec RouteSpec) error {
	spec.localPath = path
	cleanPath := rb.prefix + spec.localPath
	cleanPath = sanitizePath(cleanPath)
	spec.fullPath = cleanPath

	// Validation
	if err := validateRouteSpec(spec); err != nil {
		return fmt.Errorf("invalid route spec: %w", err)
	}

	// 1. Register route with chi
	rb.router.Method(spec.method, spec.fullPath, spec.Handler)

	// Rest of the steps are pure OpenAPI spec generation
	// We can probably skip this step when starting in production mode
	// TODO: add a mode to skip OpenAPI generation for production

	// 2. Build OpenAPI operation
	op := &openapi3.Operation{
		OperationID: spec.OperationID,
		Summary:     spec.Summary,
		Description: spec.Description,
		Tags:        spec.Tags,
		Responses:   &openapi3.Responses{},
	}

	// 3. Add parameters
	documentedPathParams := map[string]struct{}{}
	paramsInPath := map[string]struct{}{}

	// Extract param name from path
	for section := range strings.SplitSeq(spec.fullPath, "/") {
		paramsName := extractParamName(section)
		if len(paramsName) == 0 {
			continue
		}
		for _, paramName := range paramsName {
			paramsInPath[paramName] = struct{}{}
		}
	}

	for name, paramSpec := range spec.Parameters {
		if name == "" {
			return fmt.Errorf("parameter name required for %s %s", spec.method, spec.fullPath)
		}
		if paramSpec.Description == "" {
			return fmt.Errorf("parameter Description required for %s %s", spec.method, spec.fullPath)
		}
		if paramSpec.Type == nil {
			return fmt.Errorf("parameter Type required for %s %s", spec.method, spec.fullPath)
		}

		validInValues := []ParameterIn{ParameterInPath, ParameterInQuery, ParameterInHeader}
		if !slices.Contains(validInValues, paramSpec.In) {
			return fmt.Errorf("parameter In must be one of %v for %s %s", validInValues, spec.method, spec.fullPath)
		}

		paramSchema, err := rb.gen.NewSchemaRefForValue(paramSpec.Type, rb.spec.Components.Schemas)
		if err != nil {
			return fmt.Errorf("failed to generate schema for parameter %s: %w", name, err)
		}

		param := &openapi3.Parameter{
			Name:        name,
			In:          string(paramSpec.In),
			Required:    paramSpec.Required,
			Description: paramSpec.Description,
			Schema:      paramSchema,
		}

		op.Parameters = append(op.Parameters, &openapi3.ParameterRef{Value: param})

		if paramSpec.In == "path" {
			if _, exists := paramsInPath[name]; !exists {
				return fmt.Errorf("documented path parameter %s not found in path", name)
			}
			if !paramSpec.Required {
				return fmt.Errorf("path parameter %s must be required", name)
			}
			documentedPathParams[name] = struct{}{}
		}
	}

	// Validate that all path params are documented
	for name := range paramsInPath {
		if _, exists := documentedPathParams[name]; !exists {
			return fmt.Errorf("path parameter %s not documented", name)
		}
	}

	// 4. Generate request schema if provided
	if spec.RequestType != nil {
		if spec.RequestType.Type == nil {
			return fmt.Errorf("request type is nil")
		}
		reqSchema, err := rb.gen.NewSchemaRefForValue(spec.RequestType.Type, rb.spec.Components.Schemas)
		if err != nil {
			return fmt.Errorf("failed to generate request schema: %w", err)
		}
		op.RequestBody = &openapi3.RequestBodyRef{
			Value: &openapi3.RequestBody{
				Required: true,
				Content:  jsonContent(reqSchema, spec.RequestType.Examples),
			},
		}
	}

	// 5. Generate responses
	if err := rb.generateResponses(op, spec); err != nil {
		return fmt.Errorf("failed to generate responses: %w", err)
	}

	// 6. Add to OpenAPI paths
	pathItem := rb.spec.Paths.Find(spec.fullPath)
	if pathItem == nil {
		pathItem = &openapi3.PathItem{}
		rb.spec.Paths.Set(spec.fullPath, pathItem)
	}

	// Add the operation to the OpenAPI path item
	switch spec.method {
	case http.MethodGet:
		pathItem.Get = op
	case http.MethodPost:
		pathItem.Post = op
	case http.MethodPut:
		pathItem.Put = op
	case http.MethodPatch:
		pathItem.Patch = op
	case http.MethodDelete:
		pathItem.Delete = op
	default:
		return fmt.Errorf("unsupported HTTP method: %s", spec.method)
	}

	rb.l.Info("Registered route", slog.String("method", spec.method), slog.String("path", spec.fullPath), slog.String("operationID", spec.OperationID))

	return nil
}

// Get adds a GET route to the router
func (rb *RouteBuilder) Get(path string, spec RouteSpec) error {
	spec.method = http.MethodGet
	return rb.add(path, spec)
}

// Post adds a POST route to the router
func (rb *RouteBuilder) Post(path string, spec RouteSpec) error {
	spec.method = http.MethodPost
	return rb.add(path, spec)
}

// Put adds a PUT route to the router
func (rb *RouteBuilder) Put(path string, spec RouteSpec) error {
	spec.method = http.MethodPut
	return rb.add(path, spec)
}

// Patch adds a PATCH route to the router
func (rb *RouteBuilder) Patch(path string, spec RouteSpec) error {
	spec.method = http.MethodPatch
	return rb.add(path, spec)
}

// Delete adds a DELETE route to the router
func (rb *RouteBuilder) Delete(path string, spec RouteSpec) error {
	spec.method = http.MethodDelete
	return rb.add(path, spec)
}

// Router returns the underlying chi.Router
func (rb *RouteBuilder) Router() chi.Router {
	return rb.router
}

// Spec returns the OpenAPI specification
func (rb *RouteBuilder) Spec() *openapi3.T {
	return rb.spec
}

// SpecBytes returns the OpenAPI specification as JSON bytes
func (rb *RouteBuilder) SpecBytes() ([]byte, error) {
	bytes, err := rb.spec.MarshalJSON()
	if err != nil {
		return nil, err
	}
	return bytes, nil
}

// WriteSpecYAML writes the OpenAPI specification to a YAML file
func (rb *RouteBuilder) WriteSpecYAML(filename string) error {
	yamlData, err := yaml.Marshal(rb.spec)
	if err != nil {
		return err
	}
	return os.WriteFile(filename, yamlData, 0644)
}

// WriteSpecJSON writes the OpenAPI specification to a JSON file
func (rb *RouteBuilder) WriteSpecJSON(filename string) error {
	f, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}

	if err := utils.ToJSONStreamIndent(f, rb.spec); err != nil {
		return fmt.Errorf("failed to write JSON: %w", err)
	}
	return nil
}
