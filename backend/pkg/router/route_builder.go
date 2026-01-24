package router

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"slices"
	"strings"
	"ws-json-rpc/backend/pkg/router/generate"

	"github.com/go-chi/chi/v5"
)

// RouteBuilder is a chi router that collects metadata for OpenAPI generation.
type RouteBuilder struct {
	router    chi.Router
	collector generate.RouteMetadataCollector
	l         *slog.Logger
	prefix    string
}

// NewRouteBuilder creates a new RouteBuilder.
func NewRouteBuilder(l *slog.Logger, collector generate.RouteMetadataCollector) (*RouteBuilder, error) {
	return &RouteBuilder{
		router:    chi.NewRouter(),
		collector: collector,
		l:         l.With(slog.String("component", "route-builder")),
	}, nil
}

// Must exits the program if an error occurs.
func (rb *RouteBuilder) Must(err error) {
	if err != nil {
		rb.l.Error("Fatal error", slog.Any("error", err))
		os.Exit(1)
	}
}

// Route adds a new route group to the router.
func (rb *RouteBuilder) Route(path string, fn func(rb *RouteBuilder)) *RouteBuilder {
	oldPrefix := rb.prefix
	rb.prefix += path

	// Isolate sub-router
	rb.router.Group(func(r chi.Router) {
		subRB := &RouteBuilder{
			router:    r,
			collector: rb.collector,
			prefix:    rb.prefix,
			l:         rb.l.With(slog.String("prefix", rb.prefix)),
		}
		fn(subRB)
	})

	rb.prefix = oldPrefix

	return rb
}

// Use adds middlewares to the router.
func (rb *RouteBuilder) Use(middlewares ...func(http.Handler) http.Handler) *RouteBuilder {
	rb.router.Use(middlewares...)

	return rb
}

// RouteSpec defines a specific route.
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

// ParameterSpec defines a parameter for a route.
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

// add adds a new route to the router and collects metadata.
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

	// 2. Validate path parameters and collect metadata
	documentedPathParams := map[string]struct{}{}
	paramsInPath := map[string]struct{}{}

	// Extract param names from path
	for section := range strings.SplitSeq(spec.fullPath, "/") {
		paramsName := extractParamName(section)
		if len(paramsName) == 0 {
			continue
		}

		for _, paramName := range paramsName {
			paramsInPath[paramName] = struct{}{}
		}
	}

	// 4. Collect parameters metadata
	var parameters []generate.ParameterInfo

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

		parameters = append(parameters, generate.ParameterInfo{
			Name:        name,
			In:          string(paramSpec.In),
			TypeValue:   paramSpec.Type,
			Description: paramSpec.Description,
			Required:    paramSpec.Required,
		})

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

	// 5. Collect request metadata
	var requestInfo *generate.RequestInfo

	if spec.RequestType != nil {
		if spec.RequestType.Type == nil {
			return errors.New("request type is nil")
		}

		requestInfo = &generate.RequestInfo{
			TypeValue: spec.RequestType.Type,
			Examples:  spec.RequestType.Examples,
		}
	}

	// 6. Collect responses metadata
	responses := make(map[int]generate.ResponseInfo)

	for statusCode, respSpec := range spec.Responses {
		responseInfo := generate.ResponseInfo{
			StatusCode:  statusCode,
			TypeValue:   respSpec.Type,
			Description: respSpec.Description,
			Examples:    respSpec.Examples,
		}

		responses[statusCode] = responseInfo
	}

	// 7. Register route with collector
	if err := rb.collector.RegisterRoute(&generate.RouteInfo{
		OperationID: spec.OperationID,
		Method:      spec.method,
		Path:        spec.fullPath,
		Summary:     spec.Summary,
		Description: spec.Description,
		Tags:        spec.Tags,
		Deprecated:  spec.Deprecated,
		Request:     requestInfo,
		Parameters:  parameters,
		Responses:   responses,
	}); err != nil {
		return fmt.Errorf("failed to register route: %w", err)
	}

	rb.l.Info("Registered route", slog.String("method", spec.method), slog.String("path", spec.fullPath), slog.String("operationID", spec.OperationID))

	return nil
}

// Get adds a GET route to the router.
func (rb *RouteBuilder) Get(path string, spec RouteSpec) error {
	spec.method = http.MethodGet

	return rb.add(path, spec)
}

// Post adds a POST route to the router.
func (rb *RouteBuilder) Post(path string, spec RouteSpec) error {
	spec.method = http.MethodPost

	return rb.add(path, spec)
}

// Put adds a PUT route to the router.
func (rb *RouteBuilder) Put(path string, spec RouteSpec) error {
	spec.method = http.MethodPut

	return rb.add(path, spec)
}

// Patch adds a PATCH route to the router.
func (rb *RouteBuilder) Patch(path string, spec RouteSpec) error {
	spec.method = http.MethodPatch

	return rb.add(path, spec)
}

// Delete adds a DELETE route to the router.
func (rb *RouteBuilder) Delete(path string, spec RouteSpec) error {
	spec.method = http.MethodDelete

	return rb.add(path, spec)
}

// Router returns the underlying chi.Router.
//
//nolint:ireturn
func (rb *RouteBuilder) Router() chi.Router {
	return rb.router
}
