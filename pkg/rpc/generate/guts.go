package generate

import (
	"fmt"
	"os"
	"sort"

	"github.com/coder/guts"
	"github.com/coder/guts/bindings"
	"github.com/coder/guts/config"
)

type GutsGenerator struct {
	tsParser *guts.Typescript
	vm       *bindings.Bindings
}

func NewGutsGenerator(goTypesDirPath string) (*GutsGenerator, error) {
	var err error

	gutsGenerator := &GutsGenerator{}
	gutsGenerator.vm, err = bindings.New()
	if err != nil {
		return nil, fmt.Errorf("failed to create bindings VM: %w", err)
	}
	gutsGenerator.tsParser, err = newTypescriptASTFromGoTypesDir(goTypesDirPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create TypeScript AST from go types dir: %w", err)
	}
	return gutsGenerator, nil
}

func newTypescriptASTFromGoTypesDir(goTypesDirPath string) (*guts.Typescript, error) {
	// Initialize guts Go parser
	goParser, err := guts.NewGolangParser()
	if err != nil {
		return nil, fmt.Errorf("failed to create guts parser: %w", err)
	}
	goParser.PreserveComments()

	if _, err := os.Stat(goTypesDirPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("go types dir path %s does not exist", goTypesDirPath)
	}

	// Include the package where RPC types are defined
	if err := goParser.IncludeGenerate(goTypesDirPath); err != nil {
		return nil, fmt.Errorf("failed to include go types dir for parsing: %w", err)
	}

	// Try to get TypeScript AST from guts
	ts, err := goParser.ToTypescript()
	if err != nil {
		return nil, fmt.Errorf("failed to generate TypeScript AST: %w", err)
	}
	ts.ApplyMutations(
		config.EnumAsTypes,
		config.ExportTypes,
		config.InterfaceToType,
	)
	return ts, nil
}

func (g *GutsGenerator) WriteTypescriptASTToFile(ts *guts.Typescript, filePath string) error {
	str, err := ts.Serialize()
	if err != nil {
		return fmt.Errorf("failed to serialize TypeScript AST: %w", err)
	}
	err = os.WriteFile(filePath, []byte(str), 0644)
	if err != nil {
		return fmt.Errorf("failed to write TypeScript AST to file: %w", err)
	}
	return nil
}

func (g *GutsGenerator) SerializeNode(name string) (string, error) {
	node, exists := g.tsParser.Node(name)
	if !exists {
		return "", fmt.Errorf("node %s not found in TypeScript AST", name)
	}

	typescriptNode, err := g.vm.ToTypescriptNode(node)
	if err != nil {
		return "", fmt.Errorf("failed to convert node to TypeScript: %w", err)
	}

	serializedNode, err := g.vm.SerializeToTypescript(typescriptNode)
	if err != nil {
		return "", fmt.Errorf("failed to serialize node to TypeScript: %w", err)
	}

	return serializedNode, nil
}

// ExtractReferences extracts all type references from a TypeScript node.
// Returns a deduplicated sorted list of referenced type names.
func (g *GutsGenerator) ExtractReferences(name string) ([]string, error) {
	node, exists := g.tsParser.Node(name)
	if !exists {
		return nil, fmt.Errorf("node %s not found in TypeScript AST", name)
	}

	refs := make(map[string]struct{})
	g.collectTypeReferences(node, refs)

	// Convert to sorted slice
	refList := make([]string, 0, len(refs))
	for ref := range refs {
		refList = append(refList, ref)
	}

	// Sort for deterministic output
	sort.Strings(refList)

	return refList, nil
}

// collectTypeReferences recursively collects all type references from a node
func (g *GutsGenerator) collectTypeReferences(node bindings.Node, refs map[string]struct{}) {
	switch n := node.(type) {
	case *bindings.Alias:
		// Type alias: type Foo = Bar
		g.collectExpressionTypeReferences(n.Type, refs)

	case *bindings.Interface:
		// Interface: interface Foo { bar: Bar }
		for _, field := range n.Fields {
			g.collectExpressionTypeReferences(field.Type, refs)
		}
	}
}

// collectExpressionTypeReferences recursively collects references from an expression type
func (g *GutsGenerator) collectExpressionTypeReferences(expr bindings.ExpressionType, refs map[string]struct{}) {
	if expr == nil {
		return
	}

	switch e := expr.(type) {
	case *bindings.ReferenceType:
		// Direct reference to another type
		refs[e.Name.String()] = struct{}{}

		// Check generic arguments
		for _, arg := range e.Arguments {
			g.collectExpressionTypeReferences(arg, refs)
		}

	case *bindings.UnionType:
		// Union: A | B
		for _, member := range e.Types {
			g.collectExpressionTypeReferences(member, refs)
		}

	case *bindings.TypeIntersection:
		// Intersection: A & B
		for _, member := range e.Types {
			g.collectExpressionTypeReferences(member, refs)
		}

	case *bindings.ArrayLiteralType:
		// Array: T[]
		for _, elem := range e.Elements {
			g.collectExpressionTypeReferences(elem, refs)
		}

	case *bindings.TypeLiteralNode:
		// Inline object: { foo: Bar }
		for _, member := range e.Members {
			g.collectExpressionTypeReferences(member.Type, refs)
		}

	// Primitive types - no references to collect
	case *bindings.LiteralKeyword:
	case *bindings.LiteralType:
	}
}
