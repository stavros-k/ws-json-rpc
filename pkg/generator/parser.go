package generator

import (
	"fmt"
	"go/ast"
	"go/token"
	"os"
	"path"
	"reflect"
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
func (g *GoParser) Parse() error {
	// Parse all types with their names and comments and positions
	if err := g.forEachDecl(g.extractTypeMetadata); err != nil {
		return err
	}

	if err := g.forEachDecl(g.processDeclaration); err != nil {
		return err
	}

	g.printTypes()

	return nil
}

func (g *GoParser) forEachDecl(f func(pkg *packages.Package, file *ast.File, decl *ast.GenDecl) error) error {
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

// Single method to analyze any type expression
func (g *GoParser) analyzeType(expr ast.Expr) (TypeExpression, error) {
	switch t := expr.(type) {
	case *ast.Ident:
		return BasicType{Name: t.Name}, nil

	case *ast.StructType:
		// Return empty struct, fields populated in second pass
		return StructType{}, nil

	case *ast.InterfaceType:
		if len(t.Methods.List) == 0 {
			return BasicType{Name: "any"}, nil
		}
		return nil, fmt.Errorf("non-empty interfaces not supported")

	case *ast.StarExpr:
		element, err := g.analyzeType(t.X)
		if err != nil {
			return nil, err
		}
		return PointerType{Element: element}, nil

	case *ast.ArrayType:
		element, err := g.analyzeType(t.Elt)
		if err != nil {
			return nil, err
		}

		if t.Len == nil {
			return SliceType{Element: element}, nil
		}

		length := 0
		if lit, ok := t.Len.(*ast.BasicLit); ok {
			length, err = strconv.Atoi(lit.Value)
			if err != nil {
				return nil, fmt.Errorf("invalid array length: %s", lit.Value)
			}
		}
		return ArrayType{Element: element, Length: length}, nil

	case *ast.MapType:
		key, err := g.analyzeType(t.Key)
		if err != nil {
			return nil, err
		}
		value, err := g.analyzeType(t.Value)
		if err != nil {
			return nil, err
		}
		return MapType{Key: key, Value: value}, nil

	case *ast.SelectorExpr:
		if ident, ok := t.X.(*ast.Ident); ok {
			return BasicType{Name: ident.Name + "." + t.Sel.Name}, nil
		}
		return nil, fmt.Errorf("complex selector not supported")

	default:
		return nil, fmt.Errorf("unsupported type: %T", t)
	}
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

	typeExpr, err := g.analyzeType(typeSpec.Type)
	if err != nil {
		return err
	}

	typeInfo := &TypeInfo{
		Name: typeSpec.Name.Name,
		Type: typeExpr,
		Position: Position{
			Package:  pkg.PkgPath,
			Filename: path.Base(g.fset.File(decl.Pos()).Name()),
			Line:     pkg.Fset.Position(decl.Pos()).Line,
		},
	}

	if decl.Doc != nil {
		typeInfo.Comment.Above = strings.TrimSpace(decl.Doc.Text())
	}

	g.types[typeInfo.Name] = typeInfo

	return nil
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

func getEmbeddedName(t TypeExpression) string {
	switch typ := t.(type) {
	case BasicType:
		return typ.Name
	case PointerType:
		return getEmbeddedName(typ.Element)
	default:
		return ""
	}
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

	if !ast.IsExported(typeSpec.Name.Name) {
		return nil // Skip unexported types
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
		fieldType, err := g.analyzeType(field.Type)
		if err != nil {
			return err
		}

		if len(field.Names) == 0 {
			// Embedded field
			embeddedName := getEmbeddedName(fieldType)
			fields = append(fields, FieldInfo{
				Name:       embeddedName,
				Type:       fieldType,
				IsEmbedded: true,
				// ... comments ...
			})
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

			typeInfo, err := g.analyzeType(field.Type)
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

	typeInfo.Type = StructType{Fields: fields}

	return nil
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
	enumType := EnumType{EnumValues: values}
	if basic, ok := typeInfo.Type.(BasicType); ok {
		enumType.BaseType = basic.Name
	} else {
		return fmt.Errorf("enum type %s is not a basic type", enumTypeName)
	}
	typeInfo.Type = enumType

	return nil
}
