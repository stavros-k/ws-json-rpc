package generate

import "github.com/getkin/kin-openapi/openapi3"

// RouteMetadataCollector is the interface that RouteBuilder uses to collect route metadata
type RouteMetadataCollector interface {
	RegisterRoute(route *RouteInfo)
	GenerateOpenAPISpec() (*openapi3.T, error)
	Spec() (*openapi3.T, error)
	WriteSpecYAML(filename string) error
	Generate() error
}

// NoopCollector is a no-op implementation of RouteMetadataCollector
type NoopCollector struct{}

func (n *NoopCollector) RegisterRoute(route *RouteInfo)            {}
func (n *NoopCollector) GenerateOpenAPISpec() (*openapi3.T, error) { return nil, nil }
func (n *NoopCollector) Spec() (*openapi3.T, error)                { return nil, nil }
func (n *NoopCollector) WriteSpecYAML(filename string) error       { return nil }
func (n *NoopCollector) Generate() error                           { return nil }
