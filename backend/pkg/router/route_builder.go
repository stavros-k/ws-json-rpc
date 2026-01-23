package router

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"slices"
	"strconv"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3gen"
	"github.com/go-chi/chi/v5"
	"github.com/oasdiff/yaml"
)

type RouteBuilder struct {
	spec   *openapi3.T
	router chi.Router
	gen    *openapi3gen.Generator
	l      *slog.Logger
	prefix string
}

func NewRouteBuilder(l *slog.Logger, title, version string) *RouteBuilder {
	spec := &openapi3.T{
		OpenAPI: "3.0.3",
		Info: &openapi3.Info{
			Title:   title,
			Version: version,
		},
		Paths: openapi3.NewPaths(),
		Components: &openapi3.Components{
			Schemas: make(openapi3.Schemas),
		},
	}

	gen := openapi3gen.NewGenerator(
		openapi3gen.ThrowErrorOnCycle(),
		// Customizer that takes into account interfaces for customizing schemas
		SmartCustomizer(),
		// Enable component schemas
		openapi3gen.CreateComponentSchemas(openapi3gen.ExportComponentSchemasOptions{
			ExportComponentSchemas: true,
			ExportTopLevelSchema:   true,
		}),
	)

	return &RouteBuilder{
		spec:   spec,
		router: chi.NewRouter(),
		gen:    gen,
		l:      l.With(slog.String("component", "route-builder")),
	}
}

func (rb *RouteBuilder) Must(err error) {
	if err != nil {
		panic(err)
	}
}

func (rb *RouteBuilder) Route(path string, fn func(rb *RouteBuilder)) *RouteBuilder {
	oldPrefix := rb.prefix
	rb.prefix += path

	// Isolate sub-router
	rb.router.Group(func(r chi.Router) {
		subRB := &RouteBuilder{
			spec:   rb.spec,
			router: r,
			gen:    rb.gen,
			prefix: rb.prefix,
			l:      rb.l.With(slog.String("prefix", rb.prefix)),
		}
		fn(subRB)
	})

	rb.prefix = oldPrefix
	return rb
}

func (rb *RouteBuilder) Use(middlewares ...func(http.Handler) http.Handler) *RouteBuilder {
	rb.router.Use(middlewares...)
	return rb
}

func (rb *RouteBuilder) WithDescription(desc string) *RouteBuilder {
	rb.spec.Info.Description = desc
	return rb
}

func (rb *RouteBuilder) WithServer(url, description string) *RouteBuilder {
	rb.spec.Servers = append(rb.spec.Servers, &openapi3.Server{
		URL:         url,
		Description: description,
	})
	return rb
}

type RouteSpec struct {
	OperationID string
	Handler     http.HandlerFunc
	Summary     string
	Description string
	Tags        []string

	// Regular HTTP route fields
	RequestType any
	Responses   map[int]ResponseSpec

	// Common fields
	Parameters map[string]ParameterSpec

	// Internal fields
	localPath string
	fullPath  string
	method    string
}

type ParameterSpec struct {
	In          string // "path", "query", "header", "cookie"
	Description string
	Required    bool
	Type        any // The Go type - validation comes from interfaces
}

type ResponseSpec struct {
	Description string
	Type        any
}

func (rb *RouteBuilder) generateResponses(op *openapi3.Operation, spec RouteSpec) error {
	// Regular HTTP responses
	responseCodes := map[int]struct{}{}

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

		responseCodes[statusCode] = struct{}{}
		statusStr := strconv.Itoa(statusCode)

		if respSpec.Type != nil {
			// Response with body
			respSchema, err := rb.gen.NewSchemaRefForValue(respSpec.Type, rb.spec.Components.Schemas)
			if err != nil {
				return fmt.Errorf("failed to generate response schema: %w", err)
			}
			op.Responses.Set(statusStr, &openapi3.ResponseRef{
				Value: &openapi3.Response{
					Description: Ptr(respSpec.Description),
					Content:     jsonContent(respSchema),
				},
			})
		} else {
			// No body (e.g., 204 No Content)
			op.Responses.Set(statusStr, &openapi3.ResponseRef{
				Value: &openapi3.Response{
					Description: Ptr(respSpec.Description),
				},
			})
		}
	}

	return nil
}

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

		validInValues := []string{"path", "query", "header", "cookie"}
		if !slices.Contains(validInValues, paramSpec.In) {
			return fmt.Errorf("parameter In must be one of %v for %s %s", validInValues, spec.method, spec.fullPath)
		}

		paramSchema, err := rb.gen.NewSchemaRefForValue(paramSpec.Type, rb.spec.Components.Schemas)
		if err != nil {
			return fmt.Errorf("failed to generate schema for parameter %s: %w", name, err)
		}

		param := &openapi3.Parameter{
			Name:        name,
			In:          paramSpec.In,
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
		reqSchema, err := rb.gen.NewSchemaRefForValue(spec.RequestType, rb.spec.Components.Schemas)
		if err != nil {
			return fmt.Errorf("failed to generate request schema: %w", err)
		}
		op.RequestBody = &openapi3.RequestBodyRef{
			Value: &openapi3.RequestBody{
				Required: true,
				Content:  jsonContent(reqSchema),
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

	logArgs := []any{
		slog.String("method", spec.method),
		slog.String("path", spec.fullPath),
		slog.String("operationID", spec.OperationID),
	}
	rb.l.Info("Registered route", logArgs...)

	return nil
}

func (rb *RouteBuilder) Get(path string, spec RouteSpec) error {
	spec.method = http.MethodGet
	return rb.add(path, spec)
}

func (rb *RouteBuilder) Post(path string, spec RouteSpec) error {
	spec.method = http.MethodPost
	return rb.add(path, spec)
}

func (rb *RouteBuilder) Put(path string, spec RouteSpec) error {
	spec.method = http.MethodPut
	return rb.add(path, spec)
}

func (rb *RouteBuilder) Patch(path string, spec RouteSpec) error {
	spec.method = http.MethodPatch
	return rb.add(path, spec)
}

func (rb *RouteBuilder) Delete(path string, spec RouteSpec) error {
	spec.method = http.MethodDelete
	return rb.add(path, spec)
}

func (rb *RouteBuilder) Router() chi.Router {
	return rb.router
}

func (rb *RouteBuilder) Spec() *openapi3.T {
	return rb.spec
}

func (rb *RouteBuilder) SpecBytes() ([]byte, error) {
	bytes, err := rb.spec.MarshalJSON()
	if err != nil {
		return nil, err
	}
	return bytes, nil
}

func (rb *RouteBuilder) WriteSpecYAML(filename string) error {
	yamlData, err := yaml.Marshal(rb.spec)
	if err != nil {
		return err
	}
	return os.WriteFile(filename, yamlData, 0644)
}

func (rb *RouteBuilder) WriteSpecJSON(filename string) error {
	jsonData, err := json.MarshalIndent(rb.spec, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filename, jsonData, 0644)
}

func Ptr[T any](v T) *T {
	return &v
}

func jsonContent(schema *openapi3.SchemaRef) openapi3.Content {
	return openapi3.Content{
		"application/json": &openapi3.MediaType{Schema: schema},
	}
}

func getSchemaName(schema *openapi3.SchemaRef) string {
	const prefix = "#/components/schemas/"
	if schema.Ref != "" {
		return strings.TrimPrefix(schema.Ref, prefix)
	}
	return ""
}

func (rb *RouteBuilder) getSchemaValue(schema *openapi3.SchemaRef) (*openapi3.Schema, error) {
	if schema.Ref != "" {
		schemaName := getSchemaName(schema)
		if compSchema, ok := rb.spec.Components.Schemas[schemaName]; ok {
			if compSchema.Value == nil {
				return nil, fmt.Errorf("component schema %q has no value", schemaName)
			}
			return compSchema.Value, nil
		}
	}
	if schema.Value == nil {
		return nil, fmt.Errorf("schema has no value")
	}
	return schema.Value, nil
}

func extractParamName(path string) []string {
	dirtyParams := []string{}
	cleanParams := []string{}

	// Find the content between '{' and '}'
	// Examples:
	// - {userID} -> userID
	// - {userID:[0-9]+} -> userID:[0-9]+
	start := -1
	for i, ch := range path {
		if ch == '{' {
			start = i + 1
		} else if ch == '}' && start >= 0 {
			dirtyParams = append(dirtyParams, path[start:i])
			start = -1
		}
	}

	// Now split on ':' to remove any regex matchers
	// Examples:
	// - userID -> userID
	// - userID:[0-9]+ -> userID
	for _, param := range dirtyParams {
		parts := strings.Split(param, ":")
		param = parts[0]
		if param != "" {
			cleanParams = append(cleanParams, param)
		}
	}

	return cleanParams
}
