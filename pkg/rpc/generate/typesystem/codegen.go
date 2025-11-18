package typesystem

import (
	"fmt"
	"ws-json-rpc/pkg/utils"
)

// CollectGoImports collects all Go imports needed for this PropertyType.
func (pt *PropertyType) CollectGoImports(imports map[string]struct{}) {
	if pt.Items != nil {
		pt.Items.CollectGoImports(imports)
		return
	}

	if pt.MapKey != nil && pt.MapValue != nil {
		pt.MapKey.CollectGoImports(imports)
		pt.MapValue.CollectGoImports(imports)
		return
	}

	// Check for special format imports
	if pt.Primitive == PrimitiveTypeString {
		switch pt.Format {
		case StringFormatDateTime:
			imports["time"] = struct{}{}
		case StringFormatUUID:
			imports["github.com/google/uuid"] = struct{}{}
		}
	}
}

// ToGoType converts a PropertyType to a Go type string.
func (pt *PropertyType) ToGoType() string {
	if pt.Ref != "" {
		// Convert reference name to proper Go naming conventions
		goName := utils.ToPascalCase(pt.Ref)
		if pt.Nullable {
			return fmt.Sprintf("*%s", goName)
		}
		return goName
	}

	if pt.Items != nil {
		itemType := pt.Items.ToGoType()
		return fmt.Sprintf("[]%s", itemType)
	}

	if pt.MapKey != nil && pt.MapValue != nil {
		keyType := pt.MapKey.ToGoType()
		valueType := pt.MapValue.ToGoType()
		return fmt.Sprintf("map[%s]%s", keyType, valueType)
	}

	// Primitive types
	var baseType string
	switch pt.Primitive {
	case PrimitiveTypeString:
		switch pt.Format {
		case StringFormatDateTime:
			baseType = "time.Time"
		case StringFormatUUID:
			baseType = "uuid.UUID"
		default:
			baseType = "string"
		}
	case PrimitiveTypeNumber:
		baseType = "float64"
	case PrimitiveTypeInteger:
		baseType = "int64"
	case PrimitiveTypeBoolean:
		baseType = "bool"
	default:
		baseType = "interface{}"
	}

	if pt.Nullable {
		return fmt.Sprintf("*%s", baseType)
	}
	return baseType
}

// ToTypeScriptType converts a PropertyType to a TypeScript type string.
func (pt *PropertyType) ToTypeScriptType() string {
	if pt.Ref != "" {
		// Convert reference name to proper TypeScript naming conventions
		tsName := utils.ToPascalCase(pt.Ref)
		if pt.Nullable {
			return fmt.Sprintf("%s | null", tsName)
		}
		return tsName
	}

	if pt.Items != nil {
		itemType := pt.Items.ToTypeScriptType()
		if pt.Nullable {
			return fmt.Sprintf("Array<%s> | null", itemType)
		}
		return fmt.Sprintf("Array<%s>", itemType)
	}

	if pt.MapKey != nil && pt.MapValue != nil {
		valueType := pt.MapValue.ToTypeScriptType()
		if pt.Nullable {
			return fmt.Sprintf("Record<string, %s> | null", valueType)
		}
		return fmt.Sprintf("Record<string, %s>", valueType)
	}

	// Primitive types
	var baseType string
	switch pt.Primitive {
	case PrimitiveTypeString:
		baseType = "string"
	case PrimitiveTypeNumber, PrimitiveTypeInteger:
		baseType = "number"
	case PrimitiveTypeBoolean:
		baseType = "boolean"
	default:
		baseType = "unknown"
	}

	if pt.Nullable {
		return fmt.Sprintf("%s | null", baseType)
	}
	return baseType
}

// ToCSharpType converts a PropertyType to a C# type string.
func (pt *PropertyType) ToCSharpType() string {
	if pt.Ref != "" {
		// Convert reference name to proper C# naming conventions
		csName := utils.ToPascalCase(pt.Ref)
		if pt.Nullable {
			return fmt.Sprintf("%s?", csName)
		}
		return csName
	}

	if pt.Items != nil {
		itemType := pt.Items.ToCSharpType()
		return fmt.Sprintf("List<%s>", itemType)
	}

	if pt.MapKey != nil && pt.MapValue != nil {
		keyType := pt.MapKey.ToCSharpType()
		valueType := pt.MapValue.ToCSharpType()
		return fmt.Sprintf("Dictionary<%s, %s>", keyType, valueType)
	}

	// Primitive types
	var baseType string
	switch pt.Primitive {
	case PrimitiveTypeString:
		switch pt.Format {
		case StringFormatDateTime:
			baseType = "DateTime"
		case StringFormatUUID:
			baseType = "Guid"
		default:
			baseType = "string"
		}
	case PrimitiveTypeNumber:
		baseType = "double"
	case PrimitiveTypeInteger:
		baseType = "long"
	case PrimitiveTypeBoolean:
		baseType = "bool"
	default:
		baseType = "object"
	}

	if pt.Nullable && baseType != "string" {
		return fmt.Sprintf("%s?", baseType)
	}
	return baseType
}
