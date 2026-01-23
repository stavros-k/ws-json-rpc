package router

import (
	"fmt"
	"reflect"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3gen"
)

// ========================================
// Core Interfaces - Small & Focused
// ========================================

// SchemaDescriber adds description to the schema
type SchemaDescriber interface {
	SchemaDescription() string
}

// SchemaExampler provides example value(s)
type SchemaExampler interface {
	SchemaExample() any
}

type SchemaRequireder interface {
	SchemaRequired() []string
}

// SchemaFormatter sets the format field (email, uuid, date-time, etc.)
type SchemaFormatter interface {
	SchemaFormat() string
}

// SchemaEnumValue represents a single enum value with metadata
// For now we'll use plain enum, but when OpenAPI 3.1 is supported,
// we can switch to oneOf with const
type SchemaEnumValue struct {
	Value       any
	Title       string // Not used yet, but ready for 3.1
	Description string // Not used yet, but ready for 3.1
}

// SchemaEnumer defines enum values with descriptions
type SchemaEnumer interface {
	SchemaEnumValues() []SchemaEnumValue
}

// SchemaDefaulter sets the default value
type SchemaDefaulter interface {
	SchemaDefault() any
}

// SchemaConstraints adds validation constraints
type SchemaConstraints interface {
	AddSchemaConstraints(*openapi3.Schema)
}

// SchemaCustomizer provides full control over the schema
type SchemaCustomizer interface {
	CustomizeSchema(*openapi3.Schema)
}

// SchemaDeprecator marks the schema as deprecated
type SchemaDeprecator interface {
	SchemaIsDeprecated()
}

// ========================================
// The Smart Customizer
// ========================================

func SmartCustomizer() openapi3gen.Option {
	return openapi3gen.SchemaCustomizer(func(name string, ft reflect.Type, tag reflect.StructTag, schema *openapi3.Schema) error {
		// Create zero value to check interfaces
		var v any
		if ft.Kind() == reflect.Ptr {
			v = reflect.New(ft.Elem()).Interface()
		} else {
			v = reflect.New(ft).Elem().Interface()
		}

		// Priority 1: Full customizer (complete override)
		if customizer, ok := v.(SchemaCustomizer); ok {
			customizer.CustomizeSchema(schema)
			return nil
		}

		// Priority 2: Individual interfaces (composable)
		if describer, ok := v.(SchemaDescriber); ok {
			schema.Description = describer.SchemaDescription()
		}

		if exampler, ok := v.(SchemaExampler); ok {
			schema.Example = exampler.SchemaExample()
		}

		if requireder, ok := v.(SchemaRequireder); ok {
			schema.Required = requireder.SchemaRequired()
		}

		if formatter, ok := v.(SchemaFormatter); ok {
			schema.Format = formatter.SchemaFormat()
		}

		if _, ok := v.(SchemaDeprecator); ok {
			schema.Deprecated = true
		}

		// Handle enum values
		// For OpenAPI 3.0: use plain enum array
		// TODO: When kin-openapi supports 3.1, switch to oneOf with const
		if enumer, ok := v.(SchemaEnumer); ok {
			values := enumer.SchemaEnumValues()

			if len(values) > 0 {
				// OpenAPI 3.0 style - plain enum
				schema.Enum = make([]any, len(values))
				for i, val := range values {
					schema.Enum[i] = val.Value
				}

				// Build description that includes enum value descriptions
				// This gives users some benefit even in 3.0
				if schema.Description != "" {
					schema.Description += "\n\nPossible values:\n"
				} else {
					schema.Description = "Possible values:\n"
				}

				for _, val := range values {
					if val.Description != "" {
						schema.Description += "- `" + toString(val.Value) + "`: " + val.Description + "\n"
					} else {
						schema.Description += "- `" + toString(val.Value) + "`\n"
					}
				}

				// TODO: Uncomment when kin-openapi supports OpenAPI 3.1
				/*
				   schema.OneOf = make([]*openapi3.SchemaRef, len(values))
				   for i, val := range values {
				       enumSchema := &openapi3.Schema{
				           Const: val.Value,
				       }
				       if val.Title != "" {
				           enumSchema.Title = val.Title
				       }
				       if val.Description != "" {
				           enumSchema.Description = val.Description
				       }
				       schema.OneOf[i] = &openapi3.SchemaRef{Value: enumSchema}
				   }
				*/
			}
		}

		if defaulter, ok := v.(SchemaDefaulter); ok {
			schema.Default = defaulter.SchemaDefault()
		}

		if constraints, ok := v.(SchemaConstraints); ok {
			constraints.AddSchemaConstraints(schema)
		}

		return nil
	})
}

// Helper to convert value to string for description
func toString(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", v)
}
