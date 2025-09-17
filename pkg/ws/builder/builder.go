package builder

import (
	"errors"
	"fmt"
	"strings"
)

type FieldType string

const (
	StringType   FieldType = "string"
	NumberType   FieldType = "number"
	IntegerType  FieldType = "integer"
	BooleanType  FieldType = "boolean"
	ArrayType    FieldType = "array"
	ObjectType   FieldType = "object"
	EnumType     FieldType = "enum"
	CustomType   FieldType = "custom"
	JSONDateType FieldType = "jsondate"
)

type TypeBuilder interface {
	Build() (TypeDefinition, error)
	GetName() string
	AddField(name string, field FieldBuilder) TypeBuilder
	Description(desc string) TypeBuilder
}

type FieldBuilder interface {
	Optional() FieldBuilder
	JSONOmitEmpty() FieldBuilder
	JSONName(tag string) FieldBuilder
	Description(desc string) FieldBuilder
	Build() FieldDefinition
}


type TypeDefinition struct {
	Name        string                     `json:"name"`
	Type        FieldType                  `json:"type"`
	Fields      map[string]FieldDefinition `json:"fields,omitempty"`
	EnumValues  []string                   `json:"enumValues,omitempty"`
	CustomType  string                     `json:"customType,omitempty"`
	ArrayOf     *TypeDefinition            `json:"arrayOf,omitempty"`
	Description string                     `json:"description,omitempty"`
}

type FieldDefinition struct {
	Name             string            `json:"name"`
	Type             FieldType         `json:"type"`
	Optional         bool              `json:"optional"`
	JSONOmitEmpty    bool              `json:"jsonOmitEmpty"`
	JSONTag          string            `json:"jsonTag,omitempty"`
	Description      string            `json:"description,omitempty"`
	NestedType       *TypeDefinition   `json:"nestedType,omitempty"`
	EnumValues       []string          `json:"enumValues,omitempty"`
	CustomType       string            `json:"customType,omitempty"`
	ArrayOf          *TypeDefinition   `json:"arrayOf,omitempty"`
}

type typeBuilder struct {
	name         string
	fields       map[string]FieldDefinition
	fieldBuilders map[string]FieldBuilder  // Store field builders for deferred building
	description  string
	typeOverride *FieldType  // Override the default ObjectType
	enumValues   []string    // For enum types
	errors       []error
}

type fieldBuilder struct {
	name              string
	fieldType         FieldType
	optional          bool
	jsonOmitEmpty     bool
	jsonTag           string
	description       string
	nestedType        *TypeDefinition
	nestedTypeBuilder TypeBuilder
	enumValues        []string
	customType        string
	arrayOf           *TypeDefinition
}

func NewType(name string) TypeBuilder {
	return &typeBuilder{
		name:   name,
		fields: make(map[string]FieldDefinition),
	}
}

func NewEnumType(name string, values ...string) TypeBuilder {
	enumType := EnumType
	return &typeBuilder{
		name:         name,
		typeOverride: &enumType,
		enumValues:   values,
		// Don't initialize fields for enum types
	}
}

func (t *typeBuilder) GetName() string {
	return t.name
}

func (t *typeBuilder) Description(desc string) TypeBuilder {
	t.description = desc
	return t
}

func (t *typeBuilder) AddField(name string, field FieldBuilder) TypeBuilder {
	// Initialize fieldBuilders if needed
	if t.fieldBuilders == nil {
		t.fieldBuilders = make(map[string]FieldBuilder)
	}

	// Check for duplicate field names
	if _, exists := t.fieldBuilders[name]; exists {
		// Store the error for later retrieval during Build()
		if t.errors == nil {
			t.errors = make([]error, 0)
		}
		t.errors = append(t.errors, fmt.Errorf("duplicate field name: %s", name))
		return t
	}

	// Validate field name
	if name == "" {
		if t.errors == nil {
			t.errors = make([]error, 0)
		}
		t.errors = append(t.errors, errors.New("field name cannot be empty"))
		return t
	}

	// Set the field name in the builder
	if fb, ok := field.(*fieldBuilder); ok {
		fb.name = name
	}

	// Store the field builder for deferred building
	t.fieldBuilders[name] = field
	return t
}

func (t *typeBuilder) Build() (TypeDefinition, error) {
	return t.buildWithContext(nil)
}

func (t *typeBuilder) buildWithContext(schemaBuilder *SchemaBuilder) (TypeDefinition, error) {
	// Check for validation errors
	if len(t.errors) > 0 {
		return TypeDefinition{}, fmt.Errorf("validation errors: %v", t.errors)
	}

	// Validate type name
	if t.name == "" {
		return TypeDefinition{}, errors.New("type name cannot be empty")
	}

	// Build fields if we have field builders (skip for enum types)
	var fields map[string]FieldDefinition
	if t.typeOverride == nil || *t.typeOverride != EnumType {
		fields = make(map[string]FieldDefinition)
		if t.fieldBuilders != nil {
			for name, fb := range t.fieldBuilders {
				if fieldBuilderInstance, ok := fb.(*fieldBuilder); ok {
					fields[name] = fieldBuilderInstance.buildWithContext(schemaBuilder)
				} else {
					fields[name] = fb.Build()
				}
			}
		} else if t.fields != nil {
			fields = t.fields
		}
	}

	// Determine the type - use override if set, otherwise default to ObjectType
	typeDef := TypeDefinition{
		Name:        t.name,
		Type:        ObjectType,
		Fields:      fields,
		Description: t.description,
	}

	// Apply type override (e.g., for enum types)
	if t.typeOverride != nil {
		typeDef.Type = *t.typeOverride
		typeDef.EnumValues = t.enumValues
		// Enum types don't have fields
		if *t.typeOverride == EnumType {
			typeDef.Fields = nil
		}
	}

	return typeDef, nil
}

func StringField() FieldBuilder {
	return &fieldBuilder{
		fieldType: StringType,
	}
}

func NumberField() FieldBuilder {
	return &fieldBuilder{
		fieldType: NumberType,
	}
}

func IntegerField() FieldBuilder {
	return &fieldBuilder{
		fieldType: IntegerType,
	}
}

func BooleanField() FieldBuilder {
	return &fieldBuilder{
		fieldType: BooleanType,
	}
}

func ArrayField(elementType TypeDefinition) FieldBuilder {
	return &fieldBuilder{
		fieldType: ArrayType,
		arrayOf:   &elementType,
	}
}

func ObjectField(nestedTypeBuilder TypeBuilder) FieldBuilder {
	return &fieldBuilder{
		fieldType:           ObjectType,
		nestedTypeBuilder:   nestedTypeBuilder,
	}
}

func EnumField(values ...string) FieldBuilder {
	return &fieldBuilder{
		fieldType:  EnumType,
		enumValues: values,
	}
}

func JSONDateField() FieldBuilder {
	return &fieldBuilder{
		fieldType:  JSONDateType,
		customType: "JSONDate",
	}
}

func CustomField(customType string) FieldBuilder {
	return &fieldBuilder{
		fieldType:  CustomType,
		customType: customType,
	}
}


func (f *fieldBuilder) Optional() FieldBuilder {
	f.optional = true
	return f
}

func (f *fieldBuilder) JSONOmitEmpty() FieldBuilder {
	f.jsonOmitEmpty = true
	return f
}

func (f *fieldBuilder) JSONName(tag string) FieldBuilder {
	f.jsonTag = tag
	return f
}

func (f *fieldBuilder) Description(desc string) FieldBuilder {
	f.description = desc
	return f
}

func (f *fieldBuilder) Build() FieldDefinition {
	return f.buildWithContext(nil)
}

func (f *fieldBuilder) buildWithContext(schemaBuilder *SchemaBuilder) FieldDefinition {
	jsonTag := f.jsonTag
	if jsonTag == "" {
		jsonTag = f.name
	}

	tags := []string{fmt.Sprintf(`json:"%s`, jsonTag)}
	if f.jsonOmitEmpty {
		tags[0] += ",omitempty"
	}
	tags[0] += `"`

	// Resolve nested type if we have a builder
	var nestedType *TypeDefinition
	if f.nestedTypeBuilder != nil && schemaBuilder != nil {
		if cached, exists := schemaBuilder.typeCache[f.nestedTypeBuilder]; exists {
			nestedType = &cached
		} else {
			// Build the nested type
			if builtType, err := f.nestedTypeBuilder.Build(); err == nil {
				schemaBuilder.typeCache[f.nestedTypeBuilder] = builtType
				nestedType = &builtType
			}
		}
	} else if f.nestedType != nil {
		nestedType = f.nestedType
	}

	return FieldDefinition{
		Name:             f.name,
		Type:             f.fieldType,
		Optional:         f.optional,
		JSONOmitEmpty:    f.jsonOmitEmpty,
		JSONTag:          strings.Join(tags, " "),
		Description:      f.description,
		NestedType:       nestedType,
		EnumValues:       f.enumValues,
		CustomType:       f.customType,
		ArrayOf:          f.arrayOf,
	}
}

type Schema struct {
	Types map[string]TypeDefinition `json:"types"`
}

type SchemaBuilder struct {
	types       map[string]TypeDefinition
	typeCache   map[TypeBuilder]TypeDefinition  // Cache for built types
	errors      []error
}

func NewSchema() *SchemaBuilder {
	return &SchemaBuilder{
		types:     make(map[string]TypeDefinition),
		typeCache: make(map[TypeBuilder]TypeDefinition),
	}
}

func (s *SchemaBuilder) AddType(builder TypeBuilder) *SchemaBuilder {
	// Use buildWithContext if available, otherwise fallback to Build
	var typeDef TypeDefinition
	var err error

	if tb, ok := builder.(*typeBuilder); ok {
		typeDef, err = tb.buildWithContext(s)
	} else {
		typeDef, err = builder.Build()
	}

	if err != nil {
		if s.errors == nil {
			s.errors = make([]error, 0)
		}
		s.errors = append(s.errors, fmt.Errorf("failed to build type %s: %w", builder.GetName(), err))
		return s
	}

	// Check for duplicate type names
	if _, exists := s.types[typeDef.Name]; exists {
		if s.errors == nil {
			s.errors = make([]error, 0)
		}
		s.errors = append(s.errors, fmt.Errorf("duplicate type name: %s", typeDef.Name))
		return s
	}

	// Cache the built type
	s.typeCache[builder] = typeDef
	s.types[typeDef.Name] = typeDef
	return s
}

func (s *SchemaBuilder) Build() (Schema, error) {
	if len(s.errors) > 0 {
		return Schema{}, fmt.Errorf("schema validation errors: %v", s.errors)
	}

	return Schema{
		Types: s.types,
	}, nil
}

func (s *SchemaBuilder) GetType(name string) (TypeDefinition, bool) {
	typeDef, exists := s.types[name]
	return typeDef, exists
}
