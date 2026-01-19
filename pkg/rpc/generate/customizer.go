package generate

import (
	"reflect"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3gen"
)

type SchemaDescriber interface {
	// SchemaDescription returns a description for the schema.
	SchemaDescription() string
}

func SmartCustomizer() openapi3gen.Option {
	return openapi3gen.SchemaCustomizer(func(name string, ft reflect.Type, tag reflect.StructTag, schema *openapi3.Schema) error {
		// Create zero value to check interfaces
		var v any
		if ft.Kind() == reflect.Ptr {
			v = reflect.New(ft.Elem()).Interface()
		} else {
			v = reflect.New(ft).Elem().Interface()
		}

		if describer, ok := v.(SchemaDescriber); ok {
			schema.Description = describer.SchemaDescription()
		}

		return nil
	})
}
