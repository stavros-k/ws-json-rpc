// Package generate provides API documentation generation from Go type definitions.
// It extracts type metadata from Go structs, generates TypeScript definitions,
// and produces comprehensive JSON documentation including methods, events, types,
// and database schema information.
package generate

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

type MockGenerator struct{}

func (g *MockGenerator) Generate() error                                                { return nil }
func (g *MockGenerator) AddEventType(name string, resp any, docs EventDocs)             {}
func (g *MockGenerator) AddHandlerType(name string, req any, resp any, docs MethodDocs) {}
