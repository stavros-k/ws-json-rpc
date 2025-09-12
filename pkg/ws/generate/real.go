package generate

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"strings"
)

type enumValue struct {
	name    string
	val     string
	comment string
}

type enumType struct {
	name    string
	comment string
	values  []enumValue
}

type realGenerator struct {
	typeCache    map[reflect.Type]string
	eventTypes   map[string]eventType
	handlerTypes map[string]handlerInfo

	enums map[string]enumType
}

func (g *realGenerator) AddEventType(name string, resp any, docs EventDocs) {
	g.eventTypes[name] = eventType{respType: resp, docs: docs}
}

func (g *realGenerator) AddHandlerType(name string, req any, resp any, docs HandlerDocs) {
	g.handlerTypes[name] = handlerInfo{reqType: req, respType: resp, docs: docs}
}

func (g *realGenerator) Run() {
	if g.enums == nil {
		g.enums = make(map[string]enumType)
	}
	err := g.scanEnums(".")
	if err != nil {
		panic(err)
	}

	file, err := os.Create("frontend/websocket/generated.ts")
	if err != nil {
		panic(err)
	}
	defer file.Close()

	file.WriteString("// This file is generated. Do not edit manually.\n\n")

	for _, enum := range g.enums {
		_, err := file.WriteString(enumToTypescript(enum))
		if err != nil {
			panic(err)
		}
	}
}

func (g *realGenerator) scanEnums(rootPath string) error {
	tempEnums := make(map[string]enumType)
	typeComments := make(map[string]string)

	err := filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !strings.HasSuffix(path, ".go") || strings.Contains(path, "_test.go") {
			return nil
		}

		fset := token.NewFileSet()
		node, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
		if err != nil {
			return err
		}

		ast.Inspect(node, func(n ast.Node) bool {
			x, ok := n.(*ast.GenDecl)
			if !ok {
				return true
			}

			switch x.Tok {
			case token.TYPE:
				for _, spec := range x.Specs {
					// If its not a type spec (`type X ...`), skip it
					typeSpec, ok := spec.(*ast.TypeSpec)
					if !ok {
						continue
					}

					// Capture its comment
					if x.Doc != nil {
						typeComments[typeSpec.Name.Name] = strings.TrimSpace(x.Doc.Text())
					}
				}

			case token.CONST:
				for _, spec := range x.Specs {
					// If its not a value spec (`const X ...`), skip it
					valueSpec, ok := spec.(*ast.ValueSpec)
					if !ok {
						continue
					}

					// Check type once for this valueSpec
					if valueSpec.Type == nil {
						continue
					}

					// Ident is a simple identifier like "Status" in const A Status = "value"
					ident, ok := valueSpec.Type.(*ast.Ident)
					if !ok {
						continue
					}

					// Only handle simple single-name consts: const StatusOK Status = "OK"
					if len(valueSpec.Names) != 1 || len(valueSpec.Values) != 1 {
						log.Fatalf(
							"Only enums with single declarations per line are supported. Found %d names and %d values for type %q in file %q at line %d",
							len(valueSpec.Names),
							len(valueSpec.Values),
							ident.Name,
							path,
							fset.Position(valueSpec.Pos()).Line,
						)
						continue

					}

					name := valueSpec.Names[0]

					// BasicLit is a literal value like "OK", 42, true - we want string literals
					basicLit, ok := valueSpec.Values[0].(*ast.BasicLit)
					if !ok {
						continue
					}

					value := strings.Trim(basicLit.Value, `"`)
					comment := ""
					if valueSpec.Doc != nil {
						comment = strings.TrimSpace(valueSpec.Doc.Text())
					} else if valueSpec.Comment != nil {
						comment = strings.TrimSpace(valueSpec.Comment.Text())
					}

					newEnumValue := enumValue{
						name:    name.Name,
						val:     value,
						comment: comment,
					}

					if existingEnum, exists := tempEnums[ident.Name]; exists {
						existingEnum.values = append(existingEnum.values, newEnumValue)
						tempEnums[ident.Name] = existingEnum
					} else {
						tempEnums[ident.Name] = enumType{
							name:   ident.Name,
							values: []enumValue{newEnumValue},
						}
					}
				}
			}
			return true
		})

		return nil
	})

	if err != nil {
		return err
	}

	// Merge type comments and only keep enums with values
	for name, enum := range tempEnums {
		if len(enum.values) == 0 {
			continue
		}

		if comment, exists := typeComments[name]; exists {
			enum.comment = comment
		}

		g.enums[name] = enum
	}

	return nil
}

func enumToTypescript(enum enumType) string {
	// 1. Generate TypeScript enum type
	// 2. Generate a constant object for the enum values

	valuesConst := fmt.Sprintf("%sValues", enum.name)
	sb := strings.Builder{}
	commentSb := strings.Builder{}
	for _, value := range enum.values {
		if value.comment == "" {
			continue
		}
		commentSb.WriteString(fmt.Sprintf("\n * `%s`", value.name))
		commentSb.WriteString(fmt.Sprintf("\n * %s\n", value.comment))
	}

	if enum.comment != "" || commentSb.Len() > 0 {
		sb.WriteString("/**\n")
		if enum.comment != "" {
			sb.WriteString(fmt.Sprintf("* %s\n", enum.comment))
		}
		sb.WriteString(commentSb.String())

		sb.WriteString(" */\n")
	}

	// Create the type union
	sb.WriteString(fmt.Sprintf(
		"export type %s = typeof %sValues[keyof typeof %sValues];\n\n",
		enum.name, valuesConst, valuesConst,
	))

	// Create the values object
	sb.WriteString(fmt.Sprintf("export const %sValues = {\n", valuesConst))

	for i, value := range enum.values {
		if value.comment != "" {
			sb.WriteString(fmt.Sprintf("  /** %s */\n", value.comment))
		}
		sb.WriteString(fmt.Sprintf("  %s: \"%s\"", value.name, value.val))
		if i != len(enum.values)-1 {
			sb.WriteString(",\n")
		}
	}
	sb.WriteString("\n} as const;\n")

	return sb.String()
}
