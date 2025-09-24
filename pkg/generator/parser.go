package generator

import (
	"errors"
	"fmt"
	"go/ast"
	"go/token"
	"os"
	"path"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"golang.org/x/tools/go/packages"
)

type GoParser struct {
	types  map[string]*TypeInfo
	config *packages.Config
	fset   *token.FileSet
	pkgs   map[string]*packages.Package
}

func NewGoParser() *GoParser {
	fset := token.NewFileSet()
	return &GoParser{
		types: make(map[string]*TypeInfo),
		pkgs:  make(map[string]*packages.Package),
		fset:  fset,
		config: &packages.Config{
			Fset:  fset,
			Tests: false,
			Mode: packages.NeedTypes |
				packages.NeedSyntax |
				packages.NeedTypesInfo |
				packages.NeedName |
				packages.NeedDeps |
				packages.NeedTypesSizes |
				packages.NeedCompiledGoFiles,
		},
	}
}

type TypeKind string

func (t TypeKind) String() string {
	return string(t)
}

const (
	StructType TypeKind = "struct"
	EnumType   TypeKind = "enum"
	BasicType  TypeKind = "basic"
	SliceType  TypeKind = "slice"
	MapType    TypeKind = "map"
)

type Comment struct {
	// Comment above the declaration
	Above  string
	Inline string // Comment on the same line
}

func (c Comment) IsEmpty() bool {
	return c.Above == "" && c.Inline == ""
}

func (c Comment) String() string {
	var sb strings.Builder
	sb.WriteString(c.Above)
	if c.Inline != "" {
		sb.WriteString(" | ")
		sb.WriteString(c.Inline)
	}
	return sb.String()
}

type EnumValue struct {
	Name    string
	Value   string // The actual value, (ie "1", "foo", etc)
	Comment Comment
}

func (ev EnumValue) String() string {
	var sb strings.Builder
	sb.WriteString(ev.Name)
	sb.WriteString(" = ")
	sb.WriteString(ev.Value)
	if ev.Comment.IsEmpty() {
		return sb.String()
	}

	if !ev.Comment.IsEmpty() {
		sb.WriteString(" // ")
		sb.WriteString(ev.Comment.String())
		return sb.String()
	}

	return sb.String()
}

type Position struct {
	Package  string
	Filename string
	Line     int
}

func (p Position) String() string {
	return fmt.Sprintf("%s - %s:%d", p.Package, p.Filename, p.Line)
}

type TypeInfo struct {
	Name       string // Name of the type (ie TypeInfo)
	Kind       TypeKind
	Underlying string // ie for enums Kind is EnumType, Underlying is int, string, etc
	Comment    Comment
	Position   Position
	Fields     []FieldInfo // For struct types
	EnumValues []EnumValue // For enum types
}

func (ti TypeInfo) String() string {
	var sb strings.Builder

	sb.WriteString("Name: ")
	sb.WriteString(ti.Name)
	sb.WriteString("\n")
	sb.WriteString("  Kind: ")
	sb.WriteString(ti.Kind.String())
	sb.WriteString("\n")
	sb.WriteString("  Underlying: ")
	sb.WriteString(ti.Underlying)
	sb.WriteString("\n")

	if !ti.Comment.IsEmpty() {
		sb.WriteString("  Comment: ")
		sb.WriteString(ti.Comment.String())
		sb.WriteString("\n")
	}
	sb.WriteString("  Position: ")
	sb.WriteString(ti.Position.String())
	sb.WriteString("\n")

	if len(ti.Fields) > 0 {
		sb.WriteString("  Fields:")
		for _, field := range ti.Fields {
			sb.WriteString("\n    - ")
			sb.WriteString(field.String())
		}
	}

	if len(ti.EnumValues) > 0 {
		sb.WriteString("  Values:")
		for _, ev := range ti.EnumValues {
			sb.WriteString("\n    - ")
			sb.WriteString(ev.String())
		}
	}

	return sb.String()
}

type FieldTypeInfo struct {
	IsPointer   bool
	IsSlice     bool
	IsArray     bool
	IsMap       bool
	IsEmbedded  bool           // For embedded fields
	BaseType    string         // For simple types: "User", "string", etc.
	KeyType     *FieldTypeInfo // For maps: recursive type info
	ValueType   *FieldTypeInfo // For maps, slices, arrays: recursive type info
	ArrayLength string         // For fixed arrays: [5]int
}

func (fti FieldTypeInfo) String() string {
	var sb strings.Builder

	if fti.IsPointer {
		sb.WriteString("*")
	}

	if fti.IsSlice {
		sb.WriteString("[]")
		sb.WriteString(fti.ValueType.String())
	} else if fti.IsArray {
		sb.WriteString("[")
		sb.WriteString(fti.ArrayLength)
		sb.WriteString("]")
		sb.WriteString(fti.ValueType.String())
	} else if fti.IsMap {
		sb.WriteString("map[")
		sb.WriteString(fti.KeyType.String())
		sb.WriteString("]")
		sb.WriteString(fti.ValueType.String())
	} else {
		sb.WriteString(fti.BaseType)
	}

	return sb.String()
}

type FieldInfo struct {
	Name        string
	Type        *FieldTypeInfo
	JSONName    string
	JSONOptions []string
	Comment     Comment
}

func (fi FieldInfo) String() string {
	var sb strings.Builder
	if fi.Name != "" {
		sb.WriteString(fi.Name)
		sb.WriteString(" ")
	} else {
		sb.WriteString("'embedded' ")
	}
	sb.WriteString("(")
	sb.WriteString(fi.Type.String())
	sb.WriteString(")")

	if fi.JSONName != "" {
		sb.WriteString(" `")
		sb.WriteString(fi.JSONName)
		sb.WriteString("`")
		for _, opt := range fi.JSONOptions {
			sb.WriteString(", ")
			sb.WriteString(opt)
		}
	}

	if !fi.Comment.IsEmpty() {
		sb.WriteString(" // ")
		sb.WriteString(fi.Comment.String())
	}

	return sb.String()
}

func (g *GoParser) AddDir(dir string) error {
	if _, err := os.Stat(dir); err != nil {
		return err
	}
	pkgs, err := packages.Load(g.config, dir)
	if err != nil {
		return err
	}
	for _, p := range pkgs {
		if _, exists := g.pkgs[p.PkgPath]; exists {
			return fmt.Errorf("package already exists: %s", p.PkgPath)
		}
		g.pkgs[p.PkgPath] = p
	}

	return nil
}

func (g *GoParser) Run() error {
	if err := g.AddDir("./test_data"); err != nil {
		return err
	}

	if err := g.Parse(); err != nil {
		return err
	}

	tsGenerator := NewTSGenerator(&TSGeneratorOptions{
		GenerateEnumValues: true,
	})
	tsGenerator.Generate(g.types)

	return nil
}

func (g *GoParser) ForEachDecl(f func(pkg *packages.Package, file *ast.File, decl *ast.GenDecl) error) error {
	// Loop over all packages
	for _, pkg := range g.pkgs {
		// Loop over all files in the package
		for _, file := range pkg.Syntax {
			// Loop over all declarations in the file
			for _, decl := range file.Decls {
				// We only care for general declarations (types, consts, vars)
				genDecl, ok := decl.(*ast.GenDecl)
				if !ok {
					continue
				}
				// Specifically, We only care about type and const declarations
				if genDecl.Tok != token.TYPE && genDecl.Tok != token.CONST {
					continue
				}

				if len(genDecl.Specs) == 0 {
					return g.fmtError(pkg, genDecl, fmt.Errorf("no specifications found"))
				}

				if err := f(pkg, file, genDecl); err != nil {
					return g.fmtError(pkg, genDecl, err)
				}
			}
		}
	}

	return nil
}

func (g *GoParser) extractTypeMetadata(pkg *packages.Package, file *ast.File, decl *ast.GenDecl) error {
	if decl.Tok != token.TYPE {
		return nil
	}

	if len(decl.Specs) > 1 {
		return g.fmtError(pkg, decl, fmt.Errorf("multiple type specifications found, %+v", decl))
	}

	spec := decl.Specs[0]
	typeSpec, ok := spec.(*ast.TypeSpec)
	if !ok {
		return nil
	}

	if typeSpec.Name.Name == "" {
		return g.fmtError(pkg, decl, fmt.Errorf("type name is empty"))
	}

	if !ast.IsExported(typeSpec.Name.Name) {
		return nil
	}

	typeInfo := &TypeInfo{
		Name: typeSpec.Name.Name,
		Position: Position{
			Package:  pkg.PkgPath,
			Filename: path.Base(g.fset.File(decl.Pos()).Name()),
			Line:     pkg.Fset.Position(decl.Pos()).Line},
	}

	if decl.Doc != nil {
		typeInfo.Comment.Above = strings.TrimSpace(decl.Doc.Text())
	}

	// get the type
	switch t := typeSpec.Type.(type) {
	case *ast.StructType:
		typeInfo.Underlying = "struct"
		typeInfo.Kind = StructType
	case *ast.Ident:
		typeInfo.Underlying = t.Name
		typeInfo.Kind = BasicType
	case *ast.ArrayType:
		typeInfo.Underlying = "slice"
		typeInfo.Kind = SliceType
	case *ast.MapType:
		typeInfo.Underlying = "map"
		typeInfo.Kind = MapType
	case *ast.StarExpr:
		typeInfo.Underlying = "pointer"
		typeInfo.Kind = BasicType

	default:
		return g.fmtError(pkg, decl, fmt.Errorf("unsupported type: %T", t))
	}

	g.types[typeInfo.Name] = typeInfo

	return nil
}
func (g *GoParser) Parse() error {
	// Parse all types with their names and comments and positions
	if err := g.ForEachDecl(g.extractTypeMetadata); err != nil {
		return err
	}

	if err := g.ForEachDecl(g.processDeclaration); err != nil {
		return err
	}

	g.printTypes()

	return nil
}

func (g *GoParser) printTypes() {
	types := make([]*TypeInfo, 0, len(g.types))
	for _, t := range g.types {
		types = append(types, t)
	}
	sort.Slice(types, func(i, j int) bool {
		return types[i].Name < types[j].Name
	})

	for _, t := range types {
		fmt.Printf("\n- %v\n", t)
	}
}

func (g *GoParser) processDeclaration(pkg *packages.Package, file *ast.File, decl *ast.GenDecl) error {
	switch decl.Tok {
	case token.CONST:
		return g.populateTypeWithEnumInfo(pkg, decl)
	case token.TYPE:
		return g.populateTypeWithStructInfo(pkg, decl)
	default:
		fmt.Println(g.fmtError(pkg, decl, fmt.Errorf("decl is of unknown type: %s", decl.Tok.String())))
	}

	return nil
}

func (g *GoParser) populateTypeWithStructInfo(pkg *packages.Package, genDecl *ast.GenDecl) error {
	// Should have exactly one spec for a type declaration
	if len(genDecl.Specs) != 1 {
		return g.fmtError(pkg, genDecl, fmt.Errorf("expected exactly one type specification, found %d", len(genDecl.Specs)))
	}

	spec := genDecl.Specs[0]
	typeSpec, ok := spec.(*ast.TypeSpec)
	if !ok {
		return g.fmtError(pkg, genDecl, fmt.Errorf("expected TypeSpec, got %T", spec))
	}

	// Only process struct types
	structType, ok := typeSpec.Type.(*ast.StructType)
	if !ok {
		return nil // Not a struct, skip
	}

	// Find the type info we created in extractTypeMetadata
	typeInfo, exists := g.types[typeSpec.Name.Name]
	if !exists {
		return g.fmtError(pkg, genDecl, fmt.Errorf("type info not found for struct: %s", typeSpec.Name.Name))
	}

	// Extract struct fields
	var fields []FieldInfo
	for _, field := range structType.Fields.List {
		if len(field.Names) == 0 {
			fieldInfo := FieldInfo{
				Name: "",
			}

			typeInfo, err := g.analyzeFieldType(field.Type)
			if err != nil {
				return g.fmtError(pkg, genDecl, err)
			}

			typeInfo.IsEmbedded = true
			fieldInfo.Type = typeInfo

			if field.Doc != nil {
				fieldInfo.Comment.Above = strings.TrimSpace(field.Doc.Text())
			}

			if field.Comment != nil {
				fieldInfo.Comment.Inline = strings.TrimSpace(field.Comment.Text())
			}

			fields = append(fields, fieldInfo)
			continue
		}

		// Ignore unexported fields
		if !ast.IsExported(field.Names[0].Name) {
			continue // Ignore unexported fields
		}

		// Named fields
		for _, name := range field.Names {
			fieldInfo := FieldInfo{
				Name: name.Name,
			}

			typeInfo, err := g.analyzeFieldType(field.Type)
			if err != nil {
				return g.fmtError(pkg, genDecl, err)
			}
			fieldInfo.Type = typeInfo

			if field.Doc != nil {
				fieldInfo.Comment.Above = strings.TrimSpace(field.Doc.Text())
			}
			if field.Comment != nil {
				fieldInfo.Comment.Inline = strings.TrimSpace(field.Comment.Text())
			}

			if field.Tag != nil {
				jsonName, jsonOptions, err := g.parseStructTag("json", field.Tag.Value)
				if err != nil {
					return g.fmtError(pkg, genDecl, err)
				}
				if jsonName == "-" {
					continue // Ignore fields with json:"-"
				}
				if jsonName == "" {
					return g.fmtError(pkg, genDecl, fmt.Errorf("no json name found for field %s", name.Name))
				}
				fieldInfo.JSONName = jsonName
				fieldInfo.JSONOptions = jsonOptions
			}

			fields = append(fields, fieldInfo)
		}
	}

	typeInfo.Fields = fields
	return nil
}

func (g *GoParser) analyzeFieldType(expr ast.Expr) (*FieldTypeInfo, error) {
	typeInfo := &FieldTypeInfo{}
	current := expr

	// Check for pointer
	if star, ok := current.(*ast.StarExpr); ok {
		typeInfo.IsPointer = true
		current = star.X
	}

	// Check for slice/array
	if array, ok := current.(*ast.ArrayType); ok {
		if array.Len == nil {
			typeInfo.IsSlice = true
		} else {
			typeInfo.IsArray = true
			if lit, ok := array.Len.(*ast.BasicLit); ok {
				typeInfo.ArrayLength = lit.Value
			} else {
				return nil, fmt.Errorf("unsupported array length expression: %T", array.Len)
			}
		}

		// Recursively analyze element type
		elemType, err := g.analyzeFieldType(array.Elt)
		if err != nil {
			return nil, err
		}
		typeInfo.ValueType = elemType
		return typeInfo, nil
	}

	// Check for map
	if mapType, ok := current.(*ast.MapType); ok {
		typeInfo.IsMap = true

		// Recursively analyze key type
		keyType, err := g.analyzeFieldType(mapType.Key)
		if err != nil {
			return nil, err
		}
		typeInfo.KeyType = keyType

		// Recursively analyze value type
		valueType, err := g.analyzeFieldType(mapType.Value)
		if err != nil {
			return nil, err
		}
		typeInfo.ValueType = valueType
		return typeInfo, nil
	}

	// Simple base type
	switch t := current.(type) {
	case *ast.Ident:
		typeInfo.BaseType = t.Name
	case *ast.SelectorExpr:
		// Handle pkg.Type
		if ident, ok := t.X.(*ast.Ident); ok {
			typeInfo.BaseType = ident.Name + "." + t.Sel.Name
		} else {
			return nil, fmt.Errorf("complex selector expression not supported: %T", t.X)
		}
	default:
		return nil, fmt.Errorf("unsupported base type expression: %T", t)
	}

	return typeInfo, nil
}

func (g *GoParser) parseStructTag(key string, tagValue string) (string, []string, error) {
	if tagValue == "" {
		return "", nil, fmt.Errorf("empty struct tag")
	}

	// Remove surrounding backticks
	tagValue = strings.Trim(tagValue, "`")

	reflectTag := reflect.StructTag(tagValue)
	keyValue := reflectTag.Get(key)
	if keyValue == "" {
		return "", nil, fmt.Errorf("key %s not found in struct tag: %s", key, tagValue)
	}

	var options []string

	// Split options by comma
	for _, option := range strings.Split(keyValue, ",") {
		options = append(options, strings.TrimSpace(option))
	}

	if len(options) == 0 {
		return "", nil, fmt.Errorf("no options found in struct tag: %s", tagValue)
	}

	return options[0], options[1:], nil
}
func (g *GoParser) populateTypeWithEnumInfo(pkg *packages.Package, genDecl *ast.GenDecl) error {
	var enumTypeName string
	var values []EnumValue

	for _, spec := range genDecl.Specs {
		valueSpec, ok := spec.(*ast.ValueSpec)
		if !ok {
			return g.fmtError(pkg, genDecl, fmt.Errorf("expected ValueSpec, got %T", spec))
		}

		// ENFORCE: All enum members must have explicit type
		if valueSpec.Type == nil {
			return g.fmtError(pkg, genDecl, fmt.Errorf("enum member %s missing explicit type declaration", valueSpec.Names[0].Name))
		}

		ident, ok := valueSpec.Type.(*ast.Ident)
		if !ok {
			return g.fmtError(pkg, genDecl, fmt.Errorf("enum member type is not an identifier"))
		}

		// ENFORCE: iota is not supported
		if ident.Name == "iota" {
			return g.fmtError(pkg, genDecl, fmt.Errorf("iota not supported"))
		}

		if !ast.IsExported(ident.Name) {
			return nil
		}

		// ENFORCE: Each ValueSpec must have exactly one name and one value
		// (ie MyEnum1 MyEnum = "MyEnum1")
		// This means we do not support grouped declarations like:
		// const (
		//     MyEnum1 MyEnum = "MyEnum1"
		//     MyEnum2          = "MyEnum2"
		// )
		if len(valueSpec.Names) != 1 {
			if len(valueSpec.Names) == 0 {
				return g.fmtError(pkg, genDecl, fmt.Errorf("enum member declaration has no names"))
			}
			return g.fmtError(pkg, genDecl, fmt.Errorf("enum member declaration has multiple names"))
		}

		if len(valueSpec.Values) != 1 {
			return g.fmtError(pkg, genDecl, fmt.Errorf("enum member %s must have exactly one value, found %d", valueSpec.Names[0].Name, len(valueSpec.Values)))
		}

		name := valueSpec.Names[0]
		value := valueSpec.Values[0]

		// First enum member sets the type, all others must match
		if enumTypeName == "" {
			enumTypeName = ident.Name
			// Verify this type exists in our parsed types
			if _, exists := g.types[enumTypeName]; !exists {
				return g.fmtError(pkg, genDecl, fmt.Errorf("enum type not found: %s", enumTypeName))
			}
		} else if ident.Name != enumTypeName {
			return g.fmtError(pkg, genDecl, fmt.Errorf("inconsistent enum type: expected %s, got %s", enumTypeName, ident.Name))
		}

		// Skip unexported enum members
		if !ast.IsExported(name.Name) {
			continue
		}

		ev := EnumValue{
			Name: name.Name,
		}

		if t, exists := pkg.TypesInfo.Types[value]; exists && t.IsValue() {
			ev.Value = t.Value.String()
		} else {
			return g.fmtError(pkg, genDecl, fmt.Errorf("cannot determine value for enum member %s", name.Name))
		}

		if valueSpec.Doc != nil {
			ev.Comment.Above = strings.TrimSpace(valueSpec.Doc.Text())
		}

		if valueSpec.Comment != nil {
			ev.Comment.Inline = strings.TrimSpace(valueSpec.Comment.Text())
		}

		values = append(values, ev)
	}

	if enumTypeName == "" {
		return g.fmtError(pkg, genDecl, fmt.Errorf("enum type not found"))
	}

	typeInfo := g.types[enumTypeName]
	typeInfo.Kind = EnumType
	typeInfo.EnumValues = values
	return nil
}

func (g *GoParser) fmtError(pkg *packages.Package, decl *ast.GenDecl, err error) error {
	var sb strings.Builder

	// Package
	sb.WriteString("Error: \n")
	sb.WriteString("  Package: ")
	sb.WriteString(pkg.PkgPath)
	sb.WriteString("\n")

	sb.WriteString("  Position: ")
	sb.WriteString(path.Base(g.fset.File(decl.Pos()).Name()))
	sb.WriteString(":")
	sb.WriteString(strconv.Itoa(pkg.Fset.Position(decl.Pos()).Line))
	sb.WriteString("\n")

	// Declaration
	sb.WriteString("  Declaration: ")
	sb.WriteString(decl.Tok.String())
	if len(decl.Specs) > 0 {
		switch s := decl.Specs[0].(type) {
		case *ast.TypeSpec:
			if s.Name.Name != "" {
				sb.WriteString(" (type: ")
				sb.WriteString(s.Name.Name)
				sb.WriteString(")")
			}
		case *ast.ValueSpec:
			if len(s.Names) > 0 && s.Names[0].Name != "" {
				sb.WriteString(" (const: ")
				sb.WriteString(s.Names[0].Name)
				sb.WriteString(")")
			}
		}
	}
	sb.WriteString("\n")

	// Error
	sb.WriteString("  Message: ")
	sb.WriteString(err.Error())
	sb.WriteString("\n")

	return errors.New(sb.String())
}
