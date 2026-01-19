package typesystem

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
