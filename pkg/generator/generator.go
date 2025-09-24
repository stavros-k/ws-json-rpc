package generator

// import (
// 	"fmt"
// 	"go/ast"
// 	"go/token"
// 	"go/types"
// 	"reflect"
// 	"strconv"
// 	"strings"

// 	"golang.org/x/tools/go/packages"
// )

// func (ti TypeInfo) String() string {
// 	sb := strings.Builder{}
// 	sb.WriteString(ti.Name)
// 	sb.WriteString(" (")
// 	sb.WriteString(ti.Kind.String())
// 	sb.WriteString(")")
// 	if !ti.Comment.IsEmpty() {
// 		sb.WriteString(" // ")
// 		sb.WriteString(ti.Comment.String())
// 	}
// 	if ti.Underlying != "" {
// 		sb.WriteString(" (alias of ")
// 		sb.WriteString(ti.Underlying)
// 		sb.WriteString(")")
// 	}
// 	sb.WriteString("IsAlias: ")
// 	sb.WriteString(strconv.FormatBool(ti.IsAlias))
// 	if ti.PackagePath != "" {
// 		sb.WriteString(" PackagePath: ")
// 		sb.WriteString(ti.PackagePath)
// 	}
// 	if len(ti.Fields) > 0 {
// 		sb.WriteString("\n  Fields:")
// 		for _, field := range ti.Fields {
// 			sb.WriteString("\n    - ")
// 			sb.WriteString(field.String())
// 		}
// 	}

// 	if len(ti.EnumValues) > 0 {
// 		sb.WriteString("\n  Enum Values:")
// 		for _, ev := range ti.EnumValues {
// 			sb.WriteString("\n    - ")
// 			sb.WriteString(ev.String())
// 		}
// 	}

// 	return sb.String()
// }

// type FieldInfo struct {
// 	Name       string
// 	Type       string
// 	Tag        string
// 	TagOptions []string
// 	Comment    Comment
// 	IsEmbedded bool
// }

// func (fi FieldInfo) String() string {
// 	sb := strings.Builder{}
// 	sb.WriteString(fi.Name)
// 	sb.WriteString(" (")
// 	sb.WriteString(fi.Type)
// 	sb.WriteString(")")
// 	if fi.Tag != "" {
// 		sb.WriteString(" `")
// 		sb.WriteString(fi.Tag)
// 		sb.WriteString("`")
// 		for _, opt := range fi.TagOptions {
// 			sb.WriteString(", ")
// 			sb.WriteString(opt)
// 		}
// 	}
// 	return sb.String()
// }

// type generator struct {
// 	types map[string]*TypeInfo
// }

// func NewGenerator() *generator {
// 	return &generator{
// 		types: make(map[string]*TypeInfo),
// 	}
// }

// func (g *generator) Run() {
// 	cfg := &packages.Config{
// 		Mode: packages.NeedTypes |
// 			packages.NeedSyntax |
// 			packages.NeedTypesInfo |
// 			packages.NeedName |
// 			packages.NeedDeps |
// 			packages.NeedTypesSizes |
// 			packages.NeedCompiledGoFiles, // Need this for accurate positions
// 	}

// 	pkgs, err := packages.Load(cfg, "./test_data")
// 	if err != nil {
// 		panic(err)
// 	}

// 	for _, pkg := range pkgs {
// 		for _, file := range pkg.Syntax {
// 			for _, decl := range file.Decls {
// 				g.extractTypes(pkg, file, decl)
// 			}
// 		}
// 	}

// 	fmt.Println("Extracted Types:")
// 	for _, info := range g.types {
// 		fmt.Println(info.String())
// 		fmt.Println()
// 	}
// }

// func (g *generator) extractTypes(pkg *packages.Package, file *ast.File, decl ast.Decl) {
// 	genDecl, ok := decl.(*ast.GenDecl)
// 	if !ok {
// 		return
// 	}

// 	// Process type declarations
// 	if genDecl.Tok == token.TYPE {
// 		for _, spec := range genDecl.Specs {
// 			typeSpec := spec.(*ast.TypeSpec)
// 			info := extractTypeInfo(pkg, file, genDecl, typeSpec)
// 			g.types[info.Name] = info
// 		}
// 	}

// 	// Process const declarations (potential enums)
// 	if genDecl.Tok == token.CONST {
// 		enumInfos := extractEnums(pkg)
// 		for name, info := range enumInfos {
// 			g.types[name] = info
// 		}
// 	}
// }

// func extractTypeInfo(pkg *packages.Package, file *ast.File, genDecl *ast.GenDecl, typeSpec *ast.TypeSpec) *TypeInfo {
// 	info := &TypeInfo{
// 		Name:        typeSpec.Name.Name,
// 		PackagePath: pkg.PkgPath,
// 	}

// 	// Check if it's an alias (has '=' in declaration)
// 	// In AST, typeSpec.Assign indicates position of '='
// 	info.IsAlias = typeSpec.Assign != token.NoPos

// 	info.Position = Position{
// 		Filename: file.Name.Name,
// 		Line:     pkg.Fset.Position(typeSpec.Pos()).Line,
// 	}

// 	// Alternatively, use go/types for definitive answer
// 	obj := pkg.TypesInfo.Defs[typeSpec.Name]
// 	if typeName, ok := obj.(*types.TypeName); ok {
// 		info.IsAlias = typeName.IsAlias()

// 		// Get underlying type information
// 		underlying := typeName.Type().Underlying()
// 		info.Underlying = underlying.String()
// 	}

// 	// Extract type-specific information
// 	switch t := typeSpec.Type.(type) {
// 	case *ast.StructType:
// 		info.Kind = "struct"
// 		info.Fields = extractStructFields(pkg, file, t)

// 	case *ast.InterfaceType:
// 		info.Kind = "interface"
// 		// Extract interface methods similarly

// 	case *ast.Ident:
// 		info.Kind = "basic"
// 		info.Underlying = t.Name

// 	case *ast.SelectorExpr:
// 		info.Kind = "external"
// 		// e.g., time.Time
// 		if ident, ok := t.X.(*ast.Ident); ok {
// 			info.Underlying = ident.Name + "." + t.Sel.Name
// 		}
// 	}

// 	// Extract comments (covered in detail below)
// 	info.Comment = extractTypeComment(genDecl)

// 	return info
// }

// func extractStructFields(pkg *packages.Package, file *ast.File, structType *ast.StructType) []FieldInfo {
// 	var fields []FieldInfo

// 	for _, field := range structType.Fields.List {
// 		// Handle embedded fields (no names)
// 		if len(field.Names) == 0 {
// 			fieldInfo := FieldInfo{
// 				IsEmbedded: true,
// 				Type:       exprToString(pkg, field.Type),
// 				Comment:    extractFieldComment(field),
// 			}

// 			// For embedded fields, extract the type name as field name
// 			switch t := field.Type.(type) {
// 			case *ast.Ident:
// 				fieldInfo.Name = t.Name
// 				// case *ast.SelectorExpr:
// 				// 	if sel, ok := t.Sel.(*ast.Ident); ok {
// 				// 		fieldInfo.Name = sel.Name
// 				// 	}
// 			}

// 			fields = append(fields, fieldInfo)
// 			continue
// 		}

// 		// Regular fields (possibly multiple names: X, Y int)
// 		for _, name := range field.Names {
// 			fieldInfo := FieldInfo{
// 				Name:    name.Name,
// 				Type:    exprToString(pkg, field.Type),
// 				Comment: extractFieldComment(field),
// 			}

// 			// Extract struct tags
// 			if field.Tag != nil {
// 				// extract json tag
// 				tag := strings.Trim(field.Tag.Value, "`")
// 				// Parse struct tag using reflect.StructTag
// 				st := reflect.StructTag(tag)
// 				jsonTag := st.Get("json")
// 				if jsonTag != "" {
// 					parts := strings.Split(jsonTag, ",")
// 					fieldInfo.Tag = parts[0]
// 					fieldInfo.TagOptions = parts[1:]
// 				}
// 			}

// 			fields = append(fields, fieldInfo)
// 		}
// 	}

// 	return fields
// }
// func exprToString(pkg *packages.Package, expr ast.Expr) string {
// 	// Use types.TypeString for accurate representation
// 	if t := pkg.TypesInfo.Types[expr]; t.Type != nil {
// 		return t.Type.String()
// 	}

// 	// Fallback to manual conversion
// 	switch t := expr.(type) {
// 	case *ast.Ident:
// 		return t.Name
// 	case *ast.StarExpr:
// 		return "*" + exprToString(pkg, t.X)
// 	case *ast.ArrayType:
// 		if t.Len == nil {
// 			return "[]" + exprToString(pkg, t.Elt)
// 		}
// 		// Handle fixed arrays [N]Type
// 		if lit, ok := t.Len.(*ast.BasicLit); ok {
// 			return "[" + lit.Value + "]" + exprToString(pkg, t.Elt)
// 		}
// 	case *ast.SelectorExpr:
// 		return exprToString(pkg, t.X) + "." + t.Sel.Name
// 	case *ast.MapType:
// 		return "map[" + exprToString(pkg, t.Key) + "]" + exprToString(pkg, t.Value)
// 	}

// 	return "unknown"
// }

// func extractFieldComment(field *ast.Field) Comment {
// 	var comment Comment

// 	// Comment above the field
// 	if field.Doc != nil {
// 		comment.Above = strings.TrimSpace(field.Doc.Text())
// 	}

// 	// Inline comment (same line as field)
// 	if field.Comment != nil {
// 		comment.Inline = strings.TrimSpace(field.Comment.Text())
// 	}

// 	return comment
// }

// func extractEnums(pkg *packages.Package) map[string]*TypeInfo {
// 	enums := make(map[string]*TypeInfo)

// 	for _, file := range pkg.Syntax {
// 		for _, decl := range file.Decls {
// 			genDecl, ok := decl.(*ast.GenDecl)
// 			if !ok || genDecl.Tok != token.CONST {
// 				continue
// 			}

// 			// Check if this might be an enum pattern
// 			enumInfo := tryExtractEnum(pkg, genDecl)
// 			if enumInfo != nil {
// 				enums[enumInfo.Name] = enumInfo
// 			}
// 		}
// 	}

// 	return enums
// }

// func tryExtractEnum(pkg *packages.Package, genDecl *ast.GenDecl) *TypeInfo {
// 	if len(genDecl.Specs) == 0 {
// 		return nil
// 	}

// 	var enumType string
// 	var enumTypeName string
// 	var values []EnumValue
// 	hasIota := false

// 	for _, spec := range genDecl.Specs {
// 		valueSpec := spec.(*ast.ValueSpec)

// 		// Check if this const group has a type
// 		if valueSpec.Type != nil {
// 			if ident, ok := valueSpec.Type.(*ast.Ident); ok {
// 				enumType = ident.Name
// 				enumTypeName = enumType // Might be a custom type representing enum
// 			}
// 		}

// 		// Check for iota
// 		for _, value := range valueSpec.Values {
// 			if ident, ok := value.(*ast.Ident); ok && ident.Name == "iota" {
// 				hasIota = true
// 			}
// 		}

// 		// Extract each constant
// 		for j, name := range valueSpec.Names {
// 			ev := EnumValue{
// 				Name: name.Name,
// 			}

// 			// Get the value if explicitly set
// 			if j < len(valueSpec.Values) {
// 				if t := pkg.TypesInfo.Types[valueSpec.Values[j]]; t.Type != nil {
// 					ev.Value = t.Value.String()
// 				}
// 			} else if hasIota {
// 				// Implicit iota continuation
// 				ev.Value = "iota"
// 			}

// 			var comments []string
// 			// Extract comment
// 			if valueSpec.Doc != nil {
// 				comments = append(comments, valueSpec.Doc.Text())
// 			}
// 			if valueSpec.Comment != nil {
// 				comments = append(comments, valueSpec.Comment.Text())
// 			}
// 			for i, cmt := range comments {
// 				comments[i] = strings.TrimSpace(cmt)
// 			}

// 			// ev.Comment = strings.Join(comments, " - ")
// 			values = append(values, ev)
// 		}
// 	}

// 	// Heuristic: Consider it an enum if:
// 	// 1. Has iota, or
// 	// 2. Has a common type, or
// 	// 3. Has multiple related constants in a group
// 	if hasIota || enumTypeName != "" || len(values) > 2 {
// 		name := enumTypeName
// 		// if name == "" {
// 		// 	// Try to infer name from comment or first constant
// 		// 	name = inferEnumName(genDecl, values)
// 		// }

// 		return &TypeInfo{
// 			Name:       name,
// 			Kind:       "enum",
// 			EnumValues: values,
// 			Comment:    extractTypeComment(genDecl),
// 		}
// 	}

// 	return nil
// }
