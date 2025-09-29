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

type GoParserOptions struct {
	PrintParsedTypes bool
}

// GoParser analyzes Go source code to extract type information for code generation
type GoParser struct {
	// types stores all discovered exported types, keyed by type name
	// This becomes the main output of the parser containing structs, enums, etc.
	types map[string]*TypeInfo

	// config defines how the Go packages should be loaded and analyzed
	// It specifies what information we need from the Go compiler
	config *packages.Config

	// fset is the file set that tracks position information across all parsed files
	// It maps positions in the AST to actual file locations (filename:line:column)
	fset *token.FileSet

	// pkgs stores all loaded packages, keyed by import path
	// Each package contains the parsed AST, type information, etc.
	pkgs map[string]*packages.Package

	// options holds any configuration options for the parser
	options *GoParserOptions
}

// NewGoParser creates a new parser instance configured to extract type information
func NewGoParser(options *GoParserOptions) *GoParser {
	fset := token.NewFileSet()
	if options == nil {
		options = &GoParserOptions{}
	}
	return &GoParser{
		types:   make(map[string]*TypeInfo),
		pkgs:    make(map[string]*packages.Package),
		fset:    fset,
		options: options,
		config: &packages.Config{
			// Fset must be shared so position information is consistent
			Fset: fset,

			// Tests: false - don't load test files (_test.go files)
			Tests: false,

			// Mode specifies what information to load from packages:
			Mode: packages.NeedTypes | // Type information for type checking
				packages.NeedSyntax | // Parsed AST trees
				packages.NeedTypesInfo | // Maps AST nodes to type information
				packages.NeedName | // Package names
				packages.NeedDeps | // Package dependencies
				packages.NeedCompiledGoFiles, // List of compiled Go files
		},
	}
}

// AddDir loads all Go packages from the specified directory and adds them to the parser.
// It will load the package in that directory and all its file dependencies.
// Returns an error if the directory doesn't exist, loading fails, or a package is already loaded.
func (g *GoParser) AddDir(dir string) error {
	// Verify the directory exists before attempting to load packages
	// os.Stat returns an error if the path doesn't exist or isn't accessible
	if _, err := os.Stat(dir); err != nil {
		return err
	}

	// Load all packages from the directory using the parser's configuration
	// packages.Load performs full Go build analysis including:
	// - Parsing all .go files in the directory
	// - Type checking the code
	// - Resolving imports
	// - Building the AST
	// The 'dir' argument can be:
	// - "./relative/path" - relative directory path
	// - "/absolute/path" - absolute directory path
	// - "import/path" - Go import path
	pkgs, err := packages.Load(g.config, dir)
	if err != nil {
		return err
	}

	// Store each loaded package in our map
	// packages.Load may return multiple packages if there are multiple
	// packages in subdirectories or if dependencies were loaded
	for _, p := range pkgs {
		// Prevent duplicate package loading
		// This could happen if AddDir is called multiple times with
		// overlapping directories or if packages import each other
		if _, exists := g.pkgs[p.PkgPath]; exists {
			return fmt.Errorf("package already exists: %s", p.PkgPath)
		}

		// Store package keyed by its import path
		// PkgPath is the unique import path like "github.com/user/project/pkg"
		g.pkgs[p.PkgPath] = p
	}

	return nil
}

// Run is the main entry point that orchestrates the entire parsing and generation process.
// This appears to be a test/example method since it hardcodes "./test_data" directory.
// In production, this would likely accept parameters or be configured differently.
func (g *GoParser) Run() (map[string]*TypeInfo, error) {
	// Ensure at least one package has been loaded
	if len(g.pkgs) == 0 {
		return nil, fmt.Errorf("no packages loaded, call AddDir first")
	}

	// Execute the two-pass parsing process to extract all type information
	if err := g.parse(); err != nil {
		return nil, err
	}

	return g.types, nil
}

// parse performs a two-pass analysis of all loaded packages to extract type information.
// First pass: Identifies all type declarations and their basic structure
// Second pass: Populates detailed information (struct fields, enum values, etc.)
// This two-pass approach handles forward references and circular dependencies.
func (g *GoParser) parse() error {
	// FIRST PASS: Extract basic type metadata
	// This identifies all exported types (structs, enums, aliases, etc.)
	// and creates TypeInfo entries with basic information.
	// Struct fields are NOT populated yet to avoid forward reference issues.
	if err := g.forEachDecl(g.extractTypeMetadata); err != nil {
		return err
	}

	// SECOND PASS: Populate detailed type information
	// Now that all types are known, we can safely:
	// - Fill in struct fields (which may reference other types)
	// - Match enum constants to their types
	// - Resolve all type references
	if err := g.forEachDecl(g.processDeclaration); err != nil {
		return err
	}

	if g.options.PrintParsedTypes {
		// Debug output: Print all discovered types to console
		// This helps verify the parser found everything correctly
		g.printTypes()
	}

	return nil
}

// forEachDecl is a visitor pattern implementation that iterates over all declarations
// in all loaded packages and applies the provided function to each relevant declaration.
// This is the core iteration mechanism used by both parsing passes.
// The function 'f' is called for each exported TYPE or CONST declaration.
func (g *GoParser) forEachDecl(f func(pkg *packages.Package, file *ast.File, decl *ast.GenDecl) error) error {
	// Iterate through all loaded packages
	// g.pkgs is populated by AddDir() calls
	for _, pkg := range g.pkgs {
		// Each package contains multiple parsed Go files
		// pkg.Syntax is a slice of *ast.File, one for each .go file in the package
		for _, file := range pkg.Syntax {
			// Each file contains multiple top-level declarations
			// file.Decls includes imports, types, consts, vars, and functions
			for _, decl := range file.Decls {
				// Filter for general declarations only
				// ast.GenDecl represents type, const, var, and import declarations
				// ie. ast.FuncDecl (function declarations) are skipped here
				genDecl, ok := decl.(*ast.GenDecl)
				if !ok {
					continue
				}

				// Filter for TYPE and CONST tokens only
				// TYPE: type MyStruct struct{...}, type MyInt int, etc.
				// CONST: const MyEnum1 MyEnum = "value1"
				// Skips VAR (variables) and IMPORT (import statements)
				if genDecl.Tok != token.TYPE && genDecl.Tok != token.CONST {
					continue
				}

				// Sanity check: every declaration should have at least one spec
				// Specs are the actual specifications within the declaration
				// e.g., in "type (A int; B string)", there are 2 specs
				if len(genDecl.Specs) == 0 {
					return g.fmtError(pkg, genDecl, fmt.Errorf("no specifications found"))
				}

				// Performance optimization: skip unexported declarations early
				// This avoids calling processing functions for private types
				if !g.isExportedDecl(genDecl) {
					continue
				}

				// Apply the visitor function to this declaration
				// Wrap any errors with context about where they occurred
				if err := f(pkg, file, genDecl); err != nil {
					return g.fmtError(pkg, genDecl, err)
				}
			}
		}
	}

	return nil
}

// isExportedDecl determines if a declaration contains ANY exported (public) types.
// For Go, exported names start with an uppercase letter.
// This pre-filters declarations to avoid processing private types.
func (g *GoParser) isExportedDecl(decl *ast.GenDecl) bool {
	switch decl.Tok {
	case token.TYPE:
		// For type declarations, check if ANY type in the block is exported
		// This handles grouped declarations like:
		// type (
		//     unexported int
		//     Exported string  <- we want to process this
		//     alsoUnexported float
		// )
		for _, spec := range decl.Specs {
			if typeSpec, ok := spec.(*ast.TypeSpec); ok {
				if ast.IsExported(typeSpec.Name.Name) {
					// Found at least one exported type in this declaration block
					return true
				}
			}
		}
	case token.CONST:
		// For const declarations: const MyEnum1 MyEnum = "value"
		// We check if any constant references an exported type
		// This handles enum values which reference their enum type
		for _, spec := range decl.Specs {
			if valueSpec, ok := spec.(*ast.ValueSpec); ok && valueSpec.Type != nil {
				// valueSpec.Type is the type annotation (e.g., "MyEnum" in the example)
				if ident, ok := valueSpec.Type.(*ast.Ident); ok {
					// Found at least one const referencing an exported type
					// This means it's an enum value we care about
					if ast.IsExported(ident.Name) {
						return true
					}
				}
			}
		}
	}
	return false
}

// analyzeType recursively analyzes Go type expressions from the AST and converts them
// to our TypeExpression interface. This is used for both top-level type declarations
// and field types within structs. It handles all supported Go type constructs.
// Called during both first pass (for type declarations) and second pass (for struct fields).
func (g *GoParser) analyzeType(expr ast.Expr) (TypeExpression, error) {
	switch t := expr.(type) {
	case *ast.Ident:
		// Simple identifier type: string, int, MyType, etc.
		// We treat all identifiers as BasicType - could be:
		// - Built-in types: string, int, bool, any
		// - User-defined types: MyEnum, UUID
		// - Type aliases: type UUID string
		return BasicType{Name: t.Name}, nil

	case *ast.StructType:
		// Struct type definition: struct { ... }
		// Return empty StructType - fields will be populated in second pass
		// to avoid forward reference issues with field types
		return StructType{}, nil

	case *ast.InterfaceType:
		// Interface type definition: interface { ... }
		// Only support empty interface{} which becomes "any"
		if len(t.Methods.List) == 0 {
			return BasicType{Name: "any"}, nil
		}
		// Non-empty interfaces can't be serialized to JSON meaningfully
		return nil, fmt.Errorf("non-empty interfaces not supported")

	case *ast.StarExpr:
		// Pointer type: *T
		// Recursively analyze the pointed-to type
		element, err := g.analyzeType(t.X)
		if err != nil {
			return nil, err
		}
		return PointerType{Element: element}, nil

	case *ast.ArrayType:
		// Array or slice type: []T or [N]T
		// First analyze the element type
		element, err := g.analyzeType(t.Elt)
		if err != nil {
			return nil, err
		}

		// Check if it's a slice (no length) or array (has length)
		if t.Len == nil {
			// Slice type: []T
			return SliceType{Element: element}, nil
		}

		// Array type: [N]T - extract the length
		length := 0
		if lit, ok := t.Len.(*ast.BasicLit); ok {
			// Parse the array length literal
			length, err = strconv.Atoi(lit.Value)
			if err != nil {
				return nil, fmt.Errorf("invalid array length: %s", lit.Value)
			}
		}
		return ArrayType{Element: element, Length: length}, nil

	case *ast.MapType:
		// Map type: map[K]V
		// Recursively analyze both key and value types
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
		// Qualified type from another package: pkg.Type
		// Example: time.Time, uuid.UUID
		// We concatenate as "pkg.Type" string for now
		// TODO: Could handle standard library types specially
		if ident, ok := t.X.(*ast.Ident); ok {
			return BasicType{Name: ident.Name + "." + t.Sel.Name}, nil
		}
		// Complex selectors like pkg1.pkg2.Type not supported
		return nil, fmt.Errorf("complex selector not supported")

	default:
		// Catch-all for unsupported type expressions
		// Examples: chan T, func types, etc.
		return nil, fmt.Errorf("unsupported type: %T", t)
	}
}

// extractTypeMetadata is the FIRST PASS function that identifies and catalogs all type declarations.
// It creates a TypeInfo entry for each exported type with basic metadata, but doesn't populate
// complex details like struct fields (to avoid forward reference issues).
// Called via forEachDecl for every TYPE declaration in the codebase.
func (g *GoParser) extractTypeMetadata(pkg *packages.Package, file *ast.File, decl *ast.GenDecl) error {
	// Only process TYPE declarations (skip CONST, VAR, IMPORT)
	if decl.Tok != token.TYPE {
		return nil
	}

	// Handle each type spec in the declaration
	// This supports both single: type Foo int
	// And grouped: type (Foo int; Bar string; Baz struct{})
	for _, spec := range decl.Specs {
		// Ensure we have a type specification
		typeSpec, ok := spec.(*ast.TypeSpec)
		if !ok {
			continue
		}

		// Validate type has a name (shouldn't happen in valid Go code)
		if typeSpec.Name.Name == "" {
			return g.fmtError(pkg, decl, fmt.Errorf("type name is empty"))
		}

		// Skip unexported types (start with lowercase)
		// We only generate code for public API
		if !ast.IsExported(typeSpec.Name.Name) {
			continue
		}

		// Analyze the type expression to determine what kind of type this is
		// Returns BasicType, StructType (empty), SliceType, MapType, etc.
		typeExpr, err := g.analyzeType(typeSpec.Type)
		if err != nil {
			return err
		}

		var comment Comment
		if len(decl.Specs) == 1 && decl.Doc != nil {
			// Single type declaration - comment is at declaration level
			comment = g.extractComment(decl.Doc, typeSpec.Comment)
		} else {
			// Grouped declaration or no decl.Doc - use spec level
			comment = g.extractComment(typeSpec.Doc, typeSpec.Comment)
		}

		// Create and store the type metadata
		typeInfo := &TypeInfo{
			Name:    typeSpec.Name.Name,
			Type:    typeExpr,
			Comment: comment,
			Position: Position{
				Package: pkg.PkgPath,
				// Use typeSpec position for accurate line numbers in grouped declarations
				Filename: path.Base(g.fset.File(typeSpec.Pos()).Name()),
				Line:     pkg.Fset.Position(typeSpec.Pos()).Line,
			},
		}

		// Store in our type map for second pass and generation
		g.types[typeInfo.Name] = typeInfo
	}

	return nil
}

// processDeclaration is the SECOND PASS function that populates detailed information
// for types identified in the first pass. This includes struct fields and enum values.
// Called via forEachDecl for both TYPE and CONST declarations.
func (g *GoParser) processDeclaration(pkg *packages.Package, file *ast.File, decl *ast.GenDecl) error {
	switch decl.Tok {
	case token.CONST:
		// Process constants that might be enum values
		// Matches constants to their enum types and extracts values
		return g.populateTypeWithEnumInfo(pkg, decl)

	case token.TYPE:
		// Process struct types to extract their fields
		// Now safe to reference other types since first pass completed
		return g.populateTypeWithStructInfo(pkg, decl)

	default:
		// Shouldn't reach here due to forEachDecl filtering, but log if it happens
		// This is a non-fatal error - just logs and continues
		fmt.Println(g.fmtError(pkg, decl, fmt.Errorf("decl is of unknown type: %s", decl.Tok.String())))
	}

	return nil
}

// getEmbeddedName extracts the type name from an embedded field type expression.
// For embedded fields in structs, we need to determine what type is being embedded
// to properly generate the field name.
// Examples:
//   - BasicType{Name: "User"} -> "User"
//   - PointerType{Element: BasicType{Name: "User"}} -> "User" (unwraps pointer)
//   - SliceType{...} -> "" (can't embed slices)
func getEmbeddedName(t TypeExpression) string {
	switch typ := t.(type) {
	case BasicType:
		// Simple embedded type: just return its name
		// e.g., embedded field of type "User"
		return typ.Name

	case PointerType:
		// Embedded pointer: recursively get the pointed-to type's name
		// e.g., embedded field of type "*User" -> extract "User"
		return getEmbeddedName(typ.Element)

	default:
		// Other types (slices, maps, arrays) can't be embedded
		// Return empty string to indicate invalid embedded type
		return ""
	}
}

// populateTypeWithStructInfo is the SECOND PASS function for processing struct types.
// It fills in the field details for struct types that were identified in the first pass.
// This can safely reference other types since all types are now known.
// Called via processDeclaration for TYPE declarations.
func (g *GoParser) populateTypeWithStructInfo(pkg *packages.Package, genDecl *ast.GenDecl) error {
	// Handle each type spec in the declaration
	// Supports both single and grouped type declarations
	for _, spec := range genDecl.Specs {
		// Ensure we have a type specification
		typeSpec, ok := spec.(*ast.TypeSpec)
		if !ok {
			return g.fmtError(pkg, genDecl, fmt.Errorf("expected TypeSpec, got %T", spec))
		}

		// Skip unexported types (lowercase names)
		if !ast.IsExported(typeSpec.Name.Name) {
			continue
		}

		// Only process struct types - skip aliases, interfaces, etc.
		structType, ok := typeSpec.Type.(*ast.StructType)
		if !ok {
			continue // Not a struct, nothing to populate
		}

		// Find the TypeInfo created in the first pass (extractTypeMetadata)
		// This should always exist for exported types
		typeInfo, exists := g.types[typeSpec.Name.Name]
		if !exists {
			return g.fmtError(pkg, genDecl, fmt.Errorf("type info not found for struct: %s", typeSpec.Name.Name))
		}

		// Build the field list for this struct
		var fields []FieldInfo

		hasEmbeddedField := false
		// Iterate through all fields in the struct
		for _, field := range structType.Fields.List {
			// Analyze the field's type (might be basic, slice, map, pointer, etc.)
			fieldType, err := g.analyzeType(field.Type)
			if err != nil {
				return err
			}

			// Handle embedded fields (anonymous fields)
			// Example: type JSONTime struct { time.Time }
			if len(field.Names) == 0 {
				if hasEmbeddedField {
					return g.fmtError(pkg, genDecl, fmt.Errorf("structs cannot have multiple embedded fields"))
				}
				hasEmbeddedField = true

				// Extract the name from the embedded type for identification
				embeddedName := getEmbeddedName(fieldType)
				fields = append(fields, FieldInfo{
					Name:    embeddedName,
					Type:    EmbeddedType{Type: fieldType}, // Wrap in EmbeddedType marker
					Comment: g.extractComment(field.Doc, field.Comment),
				})
				continue
			}

			// Skip unexported named fields (lowercase names)
			// Check first name - if unexported, skip all names in this field
			if !ast.IsExported(field.Names[0].Name) {
				continue
			}

			// Process named fields
			// Note: Go allows multiple names per field: X, Y int
			for _, name := range field.Names {
				fieldInfo := FieldInfo{
					Name:    name.Name,
					Type:    fieldType,
					Comment: g.extractComment(field.Doc, field.Comment),
				}

				// Enforce that all exported fields must have struct tags
				if field.Tag == nil {
					return g.fmtError(pkg, genDecl, fmt.Errorf("no tag found for field %s", name.Name))
				}

				// Process struct tags
				// Tags control JSON serialization behavior
				jsonName, jsonOptions, err := g.parseStructTag("json", field.Tag.Value)
				if err != nil {
					return g.fmtError(pkg, genDecl, err)
				}

				// Enforce that all exported fields must have explicit json tags
				// This is a design choice for explicit API contracts
				if jsonName == "" {
					return g.fmtError(pkg, genDecl, fmt.Errorf("no json name found for field %s", name.Name))
				}

				// Handle json:"-" which explicitly excludes field from JSON
				if jsonName == "-" {
					continue // Skip this field entirely
				}

				// Store JSON metadata for code generation
				fieldInfo.JSONName = jsonName
				fieldInfo.JSONOptions = jsonOptions // e.g., ["omitempty", "string"]

				fields = append(fields, fieldInfo)
			}
		}
		if hasEmbeddedField && len(fields) > 1 {
			return g.fmtError(pkg, genDecl, fmt.Errorf("structs can only have embedded fields if they are the only field"))
		}

		// Replace the empty StructType from first pass with populated version
		typeInfo.Type = StructType{Fields: fields}
	}

	return nil
}

// extractComment combines documentation and inline comments from AST nodes into a Comment struct.
// Go AST separates comments into two types:
// - Doc: comment block appearing above the declaration
// - Comment: inline comment appearing on the same line after the declaration
// Example:
//
//	// This is doc comment
//	type MyType int // This is inline comment
func (g *GoParser) extractComment(doc, comment *ast.CommentGroup) Comment {
	c := Comment{}

	// Doc comments are typically multi-line blocks above declarations
	// These become the main documentation for the type/field
	if doc != nil {
		c.Above = strings.TrimSpace(doc.Text())
	}

	// Inline comments appear after the declaration on the same line
	// These are often used for brief clarifications
	if comment != nil {
		c.Inline = strings.TrimSpace(comment.Text())
	}

	return c
}

// parseStructTag extracts and parses struct field tags for a specific key (e.g., "json").
// Go struct tags are backtick-delimited strings containing key:"value" pairs.
// For JSON tags, the first value is the field name, followed by comma-separated options.
// Example input: `json:"field_name,omitempty,string"`
// Returns: ("field_name", ["omitempty", "string"], nil)
func (g *GoParser) parseStructTag(key string, tagValue string) (string, []string, error) {
	// Validate we have a tag to parse
	if tagValue == "" {
		return "", nil, fmt.Errorf("empty struct tag")
	}

	// Remove surrounding backticks from the raw tag literal
	// AST provides tags with backticks: `json:"name"`
	tagValue = strings.Trim(tagValue, "`")

	// Use Go's built-in struct tag parser for proper parsing
	// This handles edge cases like escaped quotes and spaces
	reflectTag := reflect.StructTag(tagValue)

	// Get the value for the requested key (e.g., "json")
	// Returns empty string if key doesn't exist
	keyValue := reflectTag.Get(key)
	if keyValue == "" {
		return "", nil, fmt.Errorf("key %s not found in struct tag: %s", key, tagValue)
	}

	// Parse comma-separated options within the tag value
	// First element is the name, rest are options
	var options []string
	for option := range strings.SplitSeq(keyValue, ",") {
		options = append(options, strings.TrimSpace(option))
	}

	// Shouldn't happen with valid tags, but check anyway
	if len(options) == 0 {
		return "", nil, fmt.Errorf("no options found in struct tag: %s", tagValue)
	}

	// Return: name, options array, error
	// Example: "field_name", ["omitempty", "string"], nil
	return options[0], options[1:], nil
}

// populateTypeWithEnumInfo is the SECOND PASS function for processing enum types.
// It identifies const declarations that are enum values and associates them with their enum type.
// This parser enforces strict enum conventions for reliable code generation.
// Called via processDeclaration for CONST declarations.
func (g *GoParser) populateTypeWithEnumInfo(pkg *packages.Package, genDecl *ast.GenDecl) error {
	// Track the enum type name across all constants in this block
	// All constants in a block must be of the same enum type
	var enumTypeName string
	var values []EnumValue

	// Process each constant in the const block
	for _, spec := range genDecl.Specs {
		// Ensure we have a value specification (const declaration)
		valueSpec, ok := spec.(*ast.ValueSpec)
		if !ok {
			return g.fmtError(pkg, genDecl, fmt.Errorf("expected ValueSpec, got %T", spec))
		}

		// ENFORCE: All enum members must have explicit type
		// Rejects: const MyEnum1 = "value" (missing type)
		// Requires: const MyEnum1 MyEnum = "value"
		if valueSpec.Type == nil {
			return g.fmtError(pkg, genDecl, fmt.Errorf("enum member %s missing explicit type declaration", valueSpec.Names[0].Name))
		}

		// Ensure the type is a simple identifier (not a complex type expression)
		ident, ok := valueSpec.Type.(*ast.Ident)
		if !ok {
			return g.fmtError(pkg, genDecl, fmt.Errorf("enum member type is not an identifier"))
		}

		// ENFORCE: iota is not supported
		// This parser requires explicit values for predictable code generation
		if ident.Name == "iota" {
			return g.fmtError(pkg, genDecl, fmt.Errorf("iota not supported"))
		}

		// ENFORCE: Each ValueSpec must have exactly one name and one value
		// Rejects: const A, B MyEnum = "a", "b" (multiple names)
		// Rejects: const A MyEnum (no value)
		// Requires: const A MyEnum = "a"
		if len(valueSpec.Names) != 1 {
			return g.fmtError(pkg, genDecl, fmt.Errorf("enum member %s must have exactly one name, found %d",
				valueSpec.Names[0].Name, len(valueSpec.Names)))
		}

		// Ensure exactly one value expression
		if len(valueSpec.Values) != 1 {
			return g.fmtError(pkg, genDecl, fmt.Errorf("enum member %s must have exactly one value, found %d",
				valueSpec.Names[0].Name, len(valueSpec.Values)))
		}

		name := valueSpec.Names[0]
		value := valueSpec.Values[0]

		// First constant in the block establishes the enum type
		// All subsequent constants must use the same type
		if enumTypeName == "" {
			enumTypeName = ident.Name

			// Verify this enum type was discovered in the first pass
			// If not, this const block doesn't represent a valid enum
			if _, exists := g.types[enumTypeName]; !exists {
				return g.fmtError(pkg, genDecl, fmt.Errorf("enum type not found: %s", enumTypeName))
			}
		} else if ident.Name != enumTypeName {
			// Enforce all constants in block have same type
			return g.fmtError(pkg, genDecl, fmt.Errorf("inconsistent enum type: expected %s, got %s",
				enumTypeName, ident.Name))
		}

		// Skip unexported enum members (lowercase names)
		// Example: const unexportedValue MyEnum = "internal"
		if !ast.IsExported(name.Name) {
			continue
		}

		// Create enum value entry
		ev := EnumValue{
			Name:    name.Name,
			Comment: g.extractComment(valueSpec.Doc, valueSpec.Comment),
		}

		// Extract the actual constant value using Go's type checker
		// pkg.TypesInfo.Types maps AST nodes to their computed type/value
		// This gives us the evaluated constant value (handles expressions)
		if t, exists := pkg.TypesInfo.Types[value]; exists && t.IsValue() {
			// t.Value is a go/constant.Value that can be stringified
			ev.Value = t.Value.String()
		} else {
			return g.fmtError(pkg, genDecl, fmt.Errorf("cannot determine value for enum member %s", name.Name))
		}

		values = append(values, ev)
	}

	// If no enum type was found, this const block doesn't define enum values
	if enumTypeName == "" {
		return g.fmtError(pkg, genDecl, fmt.Errorf("enum type not found"))
	}

	// Update the TypeInfo from BasicType to EnumType with the discovered values
	typeInfo := g.types[enumTypeName]
	enumType := EnumType{EnumValues: values}

	// Extract the base type (string, int, etc.) from the original BasicType
	if basic, ok := typeInfo.Type.(BasicType); ok {
		enumType.BaseType = basic.Name
	} else {
		return fmt.Errorf("enum type %s is not a basic type", enumTypeName)
	}

	// Replace the BasicType with the fully populated EnumType
	typeInfo.Type = enumType

	return nil
}
