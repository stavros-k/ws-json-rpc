package builder

import (
	"fmt"
	"strings"
	"testing"
)

func TestEnhancedTypeScriptGeneration(t *testing.T) {
	// Test schema with JSONDate fields
	userType := NewType("User").
		AddField("id", StringField()).
		AddField("name", StringField()).
		AddField("createdAt", JSONDateField()).
		AddField("updatedAt", JSONDateField())

	// Test schema without JSONDate fields
	simpleType := NewType("Simple").
		AddField("id", StringField()).
		AddField("count", IntegerField())

	// Schema with JSONDate - should generate utilities
	schemaWithDates := NewSchema().AddType(userType)
	schema1, err := schemaWithDates.Build()
	if err != nil {
		t.Fatalf("Failed to build schema with dates: %v", err)
	}

	tsGen := NewTypeScriptGenerator()
	tsCodeWithDates := tsGen.GenerateModule(schema1)

	// Should contain combined utilities
	if !strings.Contains(tsCodeWithDates, "export function jsonReplacer") {
		t.Error("Expected combined jsonReplacer function")
	}
	if !strings.Contains(tsCodeWithDates, "export function jsonReviver") {
		t.Error("Expected combined jsonReviver function")
	}
	if !strings.Contains(tsCodeWithDates, "export function parseJSON") {
		t.Error("Expected convenience parseJSON function")
	}
	if !strings.Contains(tsCodeWithDates, "export function stringifyJSON") {
		t.Error("Expected convenience stringifyJSON function")
	}

	// Schema without JSONDate - should NOT generate utilities
	schemaWithoutDates := NewSchema().AddType(simpleType)
	schema2, err := schemaWithoutDates.Build()
	if err != nil {
		t.Fatalf("Failed to build schema without dates: %v", err)
	}

	tsCodeWithoutDates := tsGen.GenerateModule(schema2)

	// Should NOT contain utilities
	if strings.Contains(tsCodeWithoutDates, "export function jsonReplacer") {
		t.Error("Should not contain jsonReplacer when no custom types are used")
	}
	if strings.Contains(tsCodeWithoutDates, "JSON utilities") {
		t.Error("Should not contain utility functions when no custom types are used")
	}

	fmt.Println("=== TypeScript with JSONDate utilities ===")
	fmt.Println(tsCodeWithDates)

	fmt.Println("\n=== TypeScript without utilities ===")
	fmt.Println(tsCodeWithoutDates)
}

func TestMultipleCustomTypes(t *testing.T) {
	// This is a placeholder for when we add more custom types
	// Currently we only have JSONDate, but the architecture supports extensibility

	userType := NewType("User").
		AddField("id", StringField()).
		AddField("createdAt", JSONDateField()).
		AddField("metadata", CustomField("CustomMetadata"))

	schema := NewSchema().AddType(userType)
	result, err := schema.Build()
	if err != nil {
		t.Fatalf("Failed to build schema: %v", err)
	}

	tsGen := NewTypeScriptGenerator()
	tsCode := tsGen.GenerateModule(result)

	// Should still generate utilities for JSONDate
	if !strings.Contains(tsCode, "export function jsonReplacer") {
		t.Error("Expected jsonReplacer function")
	}

	fmt.Println("=== TypeScript with mixed custom types ===")
	fmt.Println(tsCode)
}

func ExampleTypeScriptGenerator_enhanced() {
	// Create a schema with JSONDate fields
	userType := NewType("User").
		Description("A user with timestamps").
		AddField("id", StringField().Description("User ID")).
		AddField("name", StringField().Optional()).
		AddField("email", StringField().JSONName("email_address")).
		AddField("createdAt", JSONDateField().Description("When the user was created")).
		AddField("lastLoginAt", JSONDateField().Optional().Description("Last login time"))

	schema, _ := NewSchema().AddType(userType).Build()

	// Generate TypeScript with enhanced utilities
	tsGen := NewTypeScriptGenerator()
	tsCode := tsGen.GenerateModule(schema)

	fmt.Println("Generated TypeScript with enhanced JSON utilities:")
	fmt.Println(tsCode)
}