package typesystem

import (
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"ws-json-rpc/pkg/utils"
)

const (
	testGoPackage   = "models"
	testCSNamespace = "MyApp.Models"
)

func TestCodeGeneration(t *testing.T) {
	// Discover all test cases from testdata directory
	testCases, err := discoverTestCases("testdata")
	if err != nil {
		t.Fatalf("Failed to discover test cases: %v", err)
	}

	if len(testCases) == 0 {
		t.Fatal("No test cases found in testdata directory")
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Parse the input file
			parser := NewTypeParser(slog.New(slog.NewTextHandler(os.Stdout, nil)))
			inputPath := filepath.Join(tc.dir, "input.type.json")

			typeFile, err := parser.parseFile(inputPath)
			if err != nil {
				t.Fatalf("Failed to parse input file: %v", err)
			}

			// Register all types from the file
			for name, def := range typeFile {
				err = parser.registerType(name, def)
				if err != nil {
					t.Fatalf("Failed to register type %s: %v", name, err)
				}
			}

			// Test Go code generation if expected file exists
			if tc.hasGoExpected {
				t.Run("Go", func(t *testing.T) {
					generated, err := parser.GenerateCompleteGo(testGoPackage)
					if err != nil {
						t.Fatalf("Failed to generate Go code: %v", err)
					}

					expected, err := os.ReadFile(filepath.Join(tc.dir, "expected.go"))
					if err != nil {
						t.Fatalf("Failed to read expected Go file: %v", err)
					}

					if !compareCode(generated, string(expected)) {
						t.Errorf("Go code mismatch\nExpected:\n%s\n\nGot:\n%s", string(expected), generated)
					}
				})
			}

			// Test TypeScript code generation if expected file exists
			if tc.hasTSExpected {
				t.Run("TypeScript", func(t *testing.T) {
					generated, err := parser.GenerateCompleteTypeScript()
					if err != nil {
						t.Fatalf("Failed to generate TypeScript code: %v", err)
					}

					expected, err := os.ReadFile(filepath.Join(tc.dir, "expected.ts"))
					if err != nil {
						t.Fatalf("Failed to read expected TypeScript file: %v", err)
					}

					if !compareCode(generated, string(expected)) {
						t.Errorf("TypeScript code mismatch\nExpected:\n%s\n\nGot:\n%s", string(expected), generated)
					}
				})
			}

			// Test C# code generation if expected file exists
			if tc.hasCSExpected {
				t.Run("C#", func(t *testing.T) {
					generated, err := parser.GenerateCompleteCSharp(testCSNamespace)
					if err != nil {
						t.Fatalf("Failed to generate C# code: %v", err)
					}

					expected, err := os.ReadFile(filepath.Join(tc.dir, "expected.cs"))
					if err != nil {
						t.Fatalf("Failed to read expected C# file: %v", err)
					}

					if !compareCode(generated, string(expected)) {
						t.Errorf("C# code mismatch\nExpected:\n%s\n\nGot:\n%s", string(expected), generated)
					}
				})
			}
		})
	}
}

// TestCase represents a discovered test case
type TestCase struct {
	name          string
	dir           string
	hasGoExpected bool
	hasTSExpected bool
	hasCSExpected bool
}

// discoverTestCases walks the testdata directory and discovers all test cases
func discoverTestCases(testdataDir string) ([]TestCase, error) {
	var testCases []TestCase

	entries, err := os.ReadDir(testdataDir)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		testDir := filepath.Join(testdataDir, entry.Name())
		inputPath := filepath.Join(testDir, "input.type.json")

		// Check if input.type.json exists
		if _, err := os.Stat(inputPath); os.IsNotExist(err) {
			continue
		}

		// Check which expected files exist
		tc := TestCase{
			name:          utils.ToPascalCase(entry.Name()),
			dir:           testDir,
			hasGoExpected: utils.FileExists(filepath.Join(testDir, "expected.go")),
			hasTSExpected: utils.FileExists(filepath.Join(testDir, "expected.ts")),
			hasCSExpected: utils.FileExists(filepath.Join(testDir, "expected.cs")),
		}

		testCases = append(testCases, tc)
	}

	return testCases, nil
}

// compareCode compares two code strings, ignoring trailing whitespace differences
func compareCode(a, b string) bool {
	// Normalize line endings and trim
	a = strings.TrimSpace(strings.ReplaceAll(a, "\r\n", "\n"))
	b = strings.TrimSpace(strings.ReplaceAll(b, "\r\n", "\n"))

	return a == b
}
