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
