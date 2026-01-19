package typesystem

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"ws-json-rpc/pkg/utils"
)

// TypeParser handles parsing of .type.json files.
type TypeParser struct {
	l        *slog.Logger
	registry *TypeRegistry
}

// NewTypeParser creates a new parser.
func NewTypeParser(l *slog.Logger) *TypeParser {
	return &TypeParser{
		l:        l.With(slog.String("component", "typeparser")),
		registry: NewTypeRegistry(l.With(slog.String("component", "typeregistry"))),
	}
}

// TypeFile represents the root structure of a .type.json file.
// Keys are type names (except $schema which is ignored).
type TypeFile map[string]TypeDefinition

// TypeDefinition represents a single type definition.
type TypeDefinition struct {
	Kind        TypeKind        `json:"kind"`
	Description string          `json:"description,omitempty"`
	Values      []EnumValue     `json:"values,omitempty"`    // For enum
	Fields      []FieldDef      `json:"fields,omitempty"`    // For object
	Target      json.RawMessage `json:"target,omitempty"`    // For alias
	Format      StringFormat    `json:"format,omitempty"`    // For alias with string format
	KeyType     json.RawMessage `json:"keyType,omitempty"`   // For map
	ValueType   *FieldTypeDef   `json:"valueType,omitempty"` // For map
	ItemType    *FieldTypeDef   `json:"itemType,omitempty"`  // For array
	Raw         string          `json:"-"`                   // Formatted JSON (not serialized)
}

// FieldDef represents a field in an object.
type FieldDef struct {
	Name        string        `json:"name"`
	Description string        `json:"description,omitempty"`
	Optional    bool          `json:"optional,omitempty"`
	Nullable    bool          `json:"nullable,omitempty"`
	Type        PrimitiveType `json:"type,omitempty"`   // For primitives
	Ref         string        `json:"$ref,omitempty"`   // For type references
	Format      StringFormat  `json:"format,omitempty"` // For string formats
	Items       *FieldTypeDef `json:"items,omitempty"`  // For array fields
	Map         *MapFieldDef  `json:"map,omitempty"`    // For map fields
}

// MapFieldDef represents a map field.
type MapFieldDef struct {
	KeyType   json.RawMessage `json:"keyType"`
	ValueType *FieldTypeDef   `json:"valueType"`
}

// FieldTypeDef represents a type in a field definition.
type FieldTypeDef struct {
	Type     PrimitiveType `json:"type,omitempty"`
	Ref      string        `json:"$ref,omitempty"`
	Format   StringFormat  `json:"format,omitempty"`
	Nullable bool          `json:"nullable,omitempty"`
}

// TypeRef represents a $ref to another type.
type TypeRef struct {
	Ref string `json:"$ref"`
}

// ParseFile parses and registers types from a single .type.json file.
func (p *TypeParser) ParseFile(path string) error {
	typeFile, err := p.parseFile(path)
	if err != nil {
		return err
	}

	// Register all types from the file
	for name, def := range typeFile {
		if err := p.registerType(name, def); err != nil {
			return fmt.Errorf("failed to register type %s: %w", name, err)
		}
	}

	return nil
}

// ParseDirectory reads all .type.json files from a directory.
func (p *TypeParser) ParseDirectory(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("failed to read directory: %w", err)
	}

	// First pass: parse all files
	allTypes := make(map[string]TypeDefinition)
	for _, entry := range entries {
		if entry.IsDir() {
			p.l.Debug("skipping directory", slog.String("path", entry.Name()))
			continue
		}
		if !strings.HasSuffix(entry.Name(), ".type.json") {
			p.l.Debug("skipping non-type file", slog.String("path", entry.Name()))
			continue
		}

		path := filepath.Join(dir, entry.Name())
		typeFile, err := p.parseFile(path)
		if err != nil {
			return fmt.Errorf("failed to parse %s: %w", entry.Name(), err)
		}

		// Check for duplicates and merge types
		for name, def := range typeFile {
			if _, exists := allTypes[name]; exists {
				return fmt.Errorf("duplicate type %s in %s", name, entry.Name())
			}
			allTypes[name] = def
		}
	}

	// Second pass: register all types with their raw JSON
	for name, def := range allTypes {
		if err := p.registerTypeWithRaw(name, def, def.Raw); err != nil {
			return fmt.Errorf("failed to register type %s: %w", name, err)
		}
	}

	p.l.Info("parsed types from directory", slog.Int("count", len(allTypes)), slog.String("directory", dir))

	return nil
}

// parseFile parses a single .type.json file.
func (p *TypeParser) parseFile(path string) (TypeFile, error) {
	p.l.Info("parsing type file", slog.String("path", path))

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// First unmarshal to a raw map to handle $schema
	raw, err := utils.FromJSON[map[string]json.RawMessage](data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	// Convert to TypeFile, skipping $schema
	typeFile := make(TypeFile)
	for key, value := range raw {
		if key == "$schema" {
			continue
		}

		typeDef, err := utils.FromJSON[TypeDefinition](value)
		if err != nil {
			return nil, fmt.Errorf("failed to parse type %s: %w", key, err)
		}

		// Marshal definition with indentation for readable documentation
		typeDef.Raw = string(utils.MustToJSONIndent(typeDef))

		typeFile[key] = typeDef
	}

	return typeFile, nil
}

// registerType registers a type with the registry.
func (p *TypeParser) registerType(name string, def TypeDefinition) error {
	return p.registerTypeWithRaw(name, def, "")
}

// registerTypeWithRaw registers a type with its raw JSON definition.
func (p *TypeParser) registerTypeWithRaw(name string, def TypeDefinition, rawJSON string) error {
	switch def.Kind {
	case TypeKindEnum:
		return p.registerEnum(name, def, rawJSON)
	case TypeKindObject:
		return p.registerObject(name, def, rawJSON)
	case TypeKindAlias:
		return p.registerAlias(name, def, rawJSON)
	case TypeKindMap:
		return p.registerMap(name, def, rawJSON)
	case TypeKindArray:
		return p.registerArray(name, def, rawJSON)
	default:
		return fmt.Errorf("unknown type kind: %s", def.Kind)
	}
}

// registerEnum registers an enum type.
func (p *TypeParser) registerEnum(name string, def TypeDefinition, rawJSON string) error {
	if len(def.Values) == 0 {
		return fmt.Errorf("enum %s has no values", name)
	}

	node := NewEnumNode(name, def.Description, def.Values, rawJSON)
	return p.registry.Register(name, node)
}

// registerObject registers an object type.
func (p *TypeParser) registerObject(name string, def TypeDefinition, rawJSON string) error {
	fields := make([]FieldMetadata, 0, len(def.Fields))

	for _, fieldDef := range def.Fields {
		propType, err := p.parseFieldType(fieldDef)
		if err != nil {
			return fmt.Errorf("invalid field %s: %w", fieldDef.Name, err)
		}

		fields = append(fields, FieldMetadata{
			Name:        fieldDef.Name,
			Description: fieldDef.Description,
			Type:        propType,
			Optional:    fieldDef.Optional,
			Nullable:    fieldDef.Nullable,
		})
	}

	node := NewObjectNode(name, def.Description, fields, rawJSON)
	return p.registry.Register(name, node)
}

// parseFieldType parses a field definition into a PropertyType.
func (p *TypeParser) parseFieldType(field FieldDef) (*PropertyType, error) {
	// Type reference
	if field.Ref != "" {
		return &PropertyType{
			Ref:      field.Ref,
			Nullable: field.Nullable,
		}, nil
	}

	// Primitive type
	if field.Type != "" {
		return &PropertyType{
			Primitive: field.Type,
			Format:    field.Format,
			Nullable:  field.Nullable,
		}, nil
	}

	// Array type
	if field.Items != nil {
		itemType, err := p.parseFieldTypeDef(field.Items)
		if err != nil {
			return nil, fmt.Errorf("invalid array item type: %w", err)
		}
		return &PropertyType{
			Items:    itemType,
			Nullable: field.Nullable,
		}, nil
	}

	// Map type
	if field.Map != nil {
		keyType, err := p.parseMapKeyType(field.Map.KeyType)
		if err != nil {
			return nil, fmt.Errorf("invalid map key type: %w", err)
		}
		valueType, err := p.parseFieldTypeDef(field.Map.ValueType)
		if err != nil {
			return nil, fmt.Errorf("invalid map value type: %w", err)
		}
		return &PropertyType{
			MapKey:   keyType,
			MapValue: valueType,
			Nullable: field.Nullable,
		}, nil
	}

	return nil, fmt.Errorf("field has no type, $ref, items, or map")
}

// parseFieldTypeDef parses a FieldTypeDef into a PropertyType.
func (p *TypeParser) parseFieldTypeDef(def *FieldTypeDef) (*PropertyType, error) {
	if def.Ref != "" {
		return &PropertyType{
			Ref:      def.Ref,
			Nullable: def.Nullable,
		}, nil
	}

	if def.Type != "" {
		return &PropertyType{
			Primitive: def.Type,
			Format:    def.Format,
			Nullable:  def.Nullable,
		}, nil
	}

	return nil, fmt.Errorf("field type has no type or $ref")
}

// parseMapKeyType parses a map key type.
func (p *TypeParser) parseMapKeyType(raw json.RawMessage) (*PropertyType, error) {
	// Try primitive type
	primType, err := utils.FromJSON[PrimitiveType](raw)
	if err == nil {
		if primType == PrimitiveTypeString {
			return &PropertyType{
				Primitive: PrimitiveTypeString,
			}, nil
		}
		return nil, fmt.Errorf("invalid key type: %s (must be 'string' or enum ref)", primType)
	}

	// Try TypeRef
	typeRef, err := utils.FromJSON[TypeRef](raw)
	if err == nil && typeRef.Ref != "" {
		return &PropertyType{
			Ref: typeRef.Ref,
		}, nil
	}

	return nil, fmt.Errorf("invalid map key type")
}

// registerAlias registers an alias type.
func (p *TypeParser) registerAlias(name string, def TypeDefinition, rawJSON string) error {
	targetType, err := p.parseAliasTarget(def.Target, def.Format)
	if err != nil {
		return fmt.Errorf("invalid alias target: %w", err)
	}

	node := NewAliasNode(name, def.Description, targetType, rawJSON)
	return p.registry.Register(name, node)
}

// parseAliasTarget parses an alias target.
func (p *TypeParser) parseAliasTarget(raw json.RawMessage, format StringFormat) (*PropertyType, error) {
	// Try primitive type
	primType, err := utils.FromJSON[PrimitiveType](raw)
	if err == nil {
		return &PropertyType{
			Primitive: primType,
			Format:    format,
		}, nil
	}

	// Try TypeRef
	typeRef, err := utils.FromJSON[TypeRef](raw)
	if err == nil && typeRef.Ref != "" {
		return &PropertyType{
			Ref: typeRef.Ref,
		}, nil
	}

	return nil, fmt.Errorf("invalid target type")
}

// registerMap registers a map type.
func (p *TypeParser) registerMap(name string, def TypeDefinition, rawJSON string) error {
	keyType, err := p.parseMapKeyType(def.KeyType)
	if err != nil {
		return fmt.Errorf("invalid key type: %w", err)
	}

	valueType, err := p.parseFieldTypeDef(def.ValueType)
	if err != nil {
		return fmt.Errorf("invalid value type: %w", err)
	}

	node := NewMapNode(name, def.Description, keyType, valueType, rawJSON)
	return p.registry.Register(name, node)
}

// registerArray registers an array type.
func (p *TypeParser) registerArray(name string, def TypeDefinition, rawJSON string) error {
	itemType, err := p.parseFieldTypeDef(def.ItemType)
	if err != nil {
		return fmt.Errorf("invalid item type: %w", err)
	}

	node := NewArrayNode(name, def.Description, itemType, rawJSON)
	return p.registry.Register(name, node)
}

// GetRegistry returns the type registry.
func (p *TypeParser) GetRegistry() *TypeRegistry {
	return p.registry
}
