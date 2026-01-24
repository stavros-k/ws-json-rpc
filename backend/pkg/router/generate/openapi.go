package generate

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
)

// toOpenAPISchema converts extracted type metadata to an OpenAPI schema.
func toOpenAPISchema(typeInfo *TypeInfo) (*openapi3.Schema, error) {
	switch typeInfo.Kind {
	case TypeKindObject:
		return buildObjectSchema(typeInfo)
	case TypeKindStringEnum:
		return buildEnumSchema(typeInfo)
	default:
		return nil, fmt.Errorf("unsupported type kind: %s", typeInfo.Kind)
	}
}

// buildObjectSchema creates an OpenAPI object schema.
func buildObjectSchema(typeInfo *TypeInfo) (*openapi3.Schema, error) {
	schema := &openapi3.Schema{
		Type:        &openapi3.Types{"object"},
		Description: typeInfo.Description,
		Deprecated:  typeInfo.Deprecated != nil,
		Properties:  make(openapi3.Schemas),
		Required:    []string{},
	}

	for _, field := range typeInfo.Fields {
		fieldSchema, err := buildFieldSchema(field)
		if err != nil {
			return nil, fmt.Errorf("failed to build schema for field %s: %w", field.Name, err)
		}

		// Mark field as deprecated
		if field.Deprecated != nil {
			fieldSchema.Value.Deprecated = true
		}

		schema.Properties[field.Name] = fieldSchema

		if field.TypeInfo.Required {
			schema.Required = append(schema.Required, field.Name)
		}
	}

	return schema, nil
}

// buildFieldSchema creates an OpenAPI schema for a field.
func buildFieldSchema(field FieldInfo) (*openapi3.SchemaRef, error) {
	// Use structured type information
	return buildSchemaFromFieldType(field.TypeInfo, field.Description)
}

// buildSchemaFromFieldType converts a FieldType to an OpenAPI schema.
func buildSchemaFromFieldType(ft FieldType, description string) (*openapi3.SchemaRef, error) {
	switch ft.Kind {
	case FieldKindPrimitive:
		schema := &openapi3.Schema{
			Type:        &openapi3.Types{ft.Type},
			Description: description,
		}
		if ft.Format != "" {
			schema.Format = ft.Format
		}

		if ft.Nullable {
			schema.Nullable = true
		}

		return &openapi3.SchemaRef{Value: schema}, nil

	case FieldKindArray:
		var itemSchema *openapi3.SchemaRef

		if ft.ItemsType != nil {
			var err error

			itemSchema, err = buildSchemaFromFieldType(*ft.ItemsType, "")
			if err != nil {
				return nil, err
			}
		}

		schema := &openapi3.Schema{
			Type:        &openapi3.Types{"array"},
			Items:       itemSchema,
			Description: description,
		}
		if ft.Nullable {
			schema.Nullable = true
		}

		return &openapi3.SchemaRef{Value: schema}, nil

	case FieldKindReference, FieldKindEnum:
		// Return a $ref to the type in components/schemas
		ref := createSchemaRef(ft.Type)
		// Note: nullable on $ref requires wrapping in allOf or oneOf in OpenAPI 3.0
		// For now, we'll handle this at a higher level if needed
		return ref, nil

	case FieldKindObject:
		schema := &openapi3.Schema{
			Type:        &openapi3.Types{"object"},
			Description: description,
		}
		if ft.Nullable {
			schema.Nullable = true
		}

		return &openapi3.SchemaRef{Value: schema}, nil

	case FieldKindUnknown:
		// Unknown type - fail with error
		return nil, errors.New("cannot generate OpenAPI schema for unknown field type")

	default:
		// Unhandled type kind - fail with error
		return nil, fmt.Errorf("unhandled field kind: %s", ft.Kind)
	}
}

// buildEnumSchema creates an OpenAPI enum schema.
// FIXME: Once upstream supports OpenAPI 3.1, switch to using oneOf with const.
func buildEnumSchema(typeInfo *TypeInfo) (*openapi3.Schema, error) {
	values := make([]any, len(typeInfo.EnumValues))

	var enumDesc strings.Builder

	if typeInfo.Description != "" {
		enumDesc.WriteString(typeInfo.Description)
		enumDesc.WriteString("\n\n")
	}

	enumDesc.WriteString("Possible values:\n")

	for i, ev := range typeInfo.EnumValues {
		values[i] = ev.Value

		// Build enum value description with deprecation info
		if ev.Deprecated != nil {
			enumDesc.WriteString(fmt.Sprintf("- `%s`: **[DEPRECATED]** ", ev.Value))
			if ev.Deprecated.Message != "" {
				enumDesc.WriteString(ev.Deprecated.Message)
			}
			if ev.Description != "" {
				enumDesc.WriteString(" - " + ev.Description)
			}
			enumDesc.WriteString("\n")
		} else if ev.Description != "" {
			enumDesc.WriteString(fmt.Sprintf("- `%s`: %s\n", ev.Value, ev.Description))
		} else {
			enumDesc.WriteString(fmt.Sprintf("- `%s`\n", ev.Value))
		}
	}

	return &openapi3.Schema{
		Type:        &openapi3.Types{"string"},
		Enum:        values,
		Description: enumDesc.String(),
		Deprecated:  typeInfo.Deprecated != nil,
	}, nil
}

// buildComponentSchemas builds OpenAPI component schemas from all extracted types.
func buildComponentSchemas(doc *APIDocumentation) (openapi3.Schemas, error) {
	schemas := make(openapi3.Schemas)

	for name, typeInfo := range doc.Types {
		schema, err := toOpenAPISchema(typeInfo)
		if err != nil {
			return nil, fmt.Errorf("failed to build schema for %s: %w", name, err)
		}

		schemas[name] = &openapi3.SchemaRef{Value: schema}
	}

	return schemas, nil
}

// generateOpenAPISpec generates a complete OpenAPI specification from documentation.
func generateOpenAPISpec(doc *APIDocumentation) (*openapi3.T, error) {
	spec := &openapi3.T{
		OpenAPI:    "3.0.3",
		Info:       &openapi3.Info{},
		Paths:      openapi3.NewPaths(),
		Components: &openapi3.Components{Schemas: make(openapi3.Schemas)},
	}

	// Build all component schemas
	schemas, err := buildComponentSchemas(doc)
	if err != nil {
		return nil, fmt.Errorf("failed to build component schemas: %w", err)
	}

	spec.Components.Schemas = schemas

	// Build paths from routes
	for path, pathRoutes := range doc.Routes {
		pathItem := &openapi3.PathItem{}

		for method, route := range pathRoutes.Verbs {
			op, err := buildOperation(route, doc.Types)
			if err != nil {
				return nil, fmt.Errorf("failed to build operation %s %s: %w", method, route.Path, err)
			}

			// Add operation to path item
			switch method {
			case "GET":
				pathItem.Get = op
			case "POST":
				pathItem.Post = op
			case "PUT":
				pathItem.Put = op
			case "PATCH":
				pathItem.Patch = op
			case "DELETE":
				pathItem.Delete = op
			}
		}

		spec.Paths.Set(path, pathItem)
	}

	return spec, nil
}

// buildOperation builds an OpenAPI operation from RouteInfo.
func buildOperation(route *RouteInfo, types map[string]*TypeInfo) (*openapi3.Operation, error) {
	op := &openapi3.Operation{
		OperationID: route.OperationID,
		Summary:     route.Summary,
		Description: route.Description,
		Tags:        route.Tags,
		Deprecated:  route.Deprecated,
		Responses:   &openapi3.Responses{},
	}

	// Add parameters
	for _, param := range route.Parameters {
		p := &openapi3.Parameter{
			Name:        param.Name,
			In:          param.In,
			Required:    param.Required,
			Description: param.Description,
		}

		// Build schema for parameter type
		if typeInfo, ok := types[param.TypeName]; ok {
			schema, err := toOpenAPISchema(typeInfo)
			if err != nil {
				return nil, fmt.Errorf("failed to build schema for parameter %s: %w", param.Name, err)
			}

			p.Schema = &openapi3.SchemaRef{Value: schema}
		} else {
			// Primitive type - create inline schema
			p.Schema = &openapi3.SchemaRef{
				Value: &openapi3.Schema{Type: &openapi3.Types{param.TypeName}},
			}
		}

		op.Parameters = append(op.Parameters, &openapi3.ParameterRef{Value: p})
	}

	// Add request body
	if route.Request != nil {
		op.RequestBody = &openapi3.RequestBodyRef{
			Value: &openapi3.RequestBody{
				Required:    true,
				Description: route.Request.Description,
				Content:     createJSONContent(route.Request.TypeName, route.Request.Examples),
			},
		}
	}

	// Add responses
	for statusCode, resp := range route.Responses {
		statusStr := strconv.Itoa(statusCode)
		response := &openapi3.Response{Description: &resp.Description}

		if resp.TypeName != "" {
			response.Content = createJSONContent(resp.TypeName, resp.Examples)
		}

		op.Responses.Set(statusStr, &openapi3.ResponseRef{Value: response})
	}

	return op, nil
}

// createJSONContent creates OpenAPI content for application/json with given type and examples.
func createJSONContent(typeName string, examples map[string]any) openapi3.Content {
	return openapi3.Content{
		"application/json": &openapi3.MediaType{
			Schema:   createSchemaRef(typeName),
			Examples: convertExamplesToOpenAPI(examples),
		},
	}
}

// createSchemaRef creates a schema reference for the given type name.
func createSchemaRef(typeName string) *openapi3.SchemaRef {
	return &openapi3.SchemaRef{
		Ref: "#/components/schemas/" + typeName,
	}
}

// convertExamplesToOpenAPI converts examples map to OpenAPI format.
func convertExamplesToOpenAPI(examples map[string]any) openapi3.Examples {
	if len(examples) == 0 {
		return nil
	}

	result := make(openapi3.Examples)
	for name, value := range examples {
		result[name] = &openapi3.ExampleRef{Value: &openapi3.Example{Value: value}}
	}

	return result
}
