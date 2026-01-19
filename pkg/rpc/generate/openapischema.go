package generate

import (
	"fmt"
	"ws-json-rpc/pkg/utils"

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

func (g *GeneratorImpl) getJsonSchema(name string, v any) (string, *openapi3.SchemaRef, error) {
	schemaRef, err := g.schemaGen.NewSchemaRefForValue(v, nil)
	if err != nil {
		return "", nil, fmt.Errorf("failed to generate JSON schema for type %s: %w", name, err)
	}

	if schemaRef == nil || schemaRef.Value == nil {
		return "", nil, fmt.Errorf("failed to generate JSON schema for type %s: %w", name, err)
	}

	schemaBytes, err := schemaRef.Value.MarshalJSON()
	if err != nil {
		return "", nil, err
	}

	schema := utils.MustFromJSON[map[string]any](schemaBytes)
	indented, err := utils.ToJSONIndent(schema)
	return string(indented), schemaRef, err
}
