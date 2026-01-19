package generate

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3gen"
)

func newOpenAPISchemaGenerator() *openapi3gen.Generator {
	return openapi3gen.NewGenerator(
		SmartCustomizer(),
		openapi3gen.ThrowErrorOnCycle(),
		openapi3gen.CreateComponentSchemas(openapi3gen.ExportComponentSchemasOptions{
			ExportComponentSchemas: true,
			ExportTopLevelSchema:   true,
		}),
	)
}
func (g *GeneratorImpl) getResolvedRef(schemaRef *openapi3.SchemaRef) (*openapi3.SchemaRef, error) {
	if schemaRef.Value == nil && schemaRef.Ref != "" {
		// Extract the schema name from the reference (e.g., "#/components/schemas/TypeName" -> "TypeName")
		schemaName := extractTypeNameFromRef(schemaRef.Ref)

		componentRef, exists := g.componentSchemas[schemaName]
		if !exists {
			return nil, fmt.Errorf("failed to resolve schema reference %s", schemaRef.Ref)
		}
		return componentRef, nil
	}
	return schemaRef, nil
}

func (g *GeneratorImpl) getJsonSchema(name string, v any) (string, *openapi3.SchemaRef, error) {
	// Pass componentSchemas map to collect all schemas
	schemaRef, err := g.schemaGen.NewSchemaRefForValue(v, g.componentSchemas)
	if err != nil {
		return "", nil, fmt.Errorf("failed to generate JSON schema ref for type %s: %w", name, err)
	}

	if schemaRef == nil {
		return "", nil, fmt.Errorf("schemaRef is nil for type %s", name)
	}

	// When ExportComponentSchemas is enabled, schemaRef.Value may be nil (it's a reference)
	// We need to resolve the reference from componentSchemas
	resolvedRef, err := g.getResolvedRef(schemaRef)
	if err != nil {
		return "", nil, fmt.Errorf("failed to resolve schema reference for type %s: %w", name, err)
	}

	if resolvedRef.Value == nil {
		return "", nil, fmt.Errorf("resolved schema value is nil for type %s", name)
	}

	schemaBytes, err := resolvedRef.Value.MarshalJSON()
	if err != nil {
		return "", nil, err
	}
	var buf bytes.Buffer
	if err := json.Indent(&buf, schemaBytes, "", "  "); err != nil {
		return "", nil, err
	}
	return buf.String(), resolvedRef, err
}

// extractTypeNameFromRef extracts the type name from a $ref like "#/components/schemas/UserType".
func extractTypeNameFromRef(ref string) string {
	parts := strings.Split(ref, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return ref
}
