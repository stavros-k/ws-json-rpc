package generate

import (
	"fmt"
	"os"

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
