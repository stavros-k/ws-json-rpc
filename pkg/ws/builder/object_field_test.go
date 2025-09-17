package builder

import (
	"strings"
	"testing"
)

func TestObjectFieldWithTypeReuse(t *testing.T) {
	// Create an Address type
	addressType := NewType("Address").
		AddField("street", StringField()).
		AddField("city", StringField()).
		AddField("zipCode", StringField())

	// Create a User type that reuses the Address type multiple times
	userType := NewType("User").
		AddField("id", StringField()).
		AddField("name", StringField()).
		AddField("homeAddress", ObjectField(addressType).Optional()).
		AddField("workAddress", ObjectField(addressType).Optional()).
		AddField("billingAddress", ObjectField(addressType))

	// Build the schema - should handle type reuse correctly
	schema, err := NewSchema().
		AddType(userType).
		AddType(addressType).
		Build()

	if err != nil {
		t.Fatalf("Failed to build schema with reused types: %v", err)
	}

	// Verify the schema has both types
	if len(schema.Types) != 2 {
		t.Errorf("Expected 2 types in schema, got %d", len(schema.Types))
	}

	// Verify User type has the correct fields
	userTypeDef, exists := schema.Types["User"]
	if !exists {
		t.Fatal("User type not found in schema")
	}

	expectedFields := []string{"id", "name", "homeAddress", "workAddress", "billingAddress"}
	for _, fieldName := range expectedFields {
		if _, exists := userTypeDef.Fields[fieldName]; !exists {
			t.Errorf("Expected field '%s' not found in User type", fieldName)
		}
	}

	// Verify that address fields reference the Address type
	homeAddressField := userTypeDef.Fields["homeAddress"]
	if homeAddressField.Type != ObjectType {
		t.Errorf("Expected homeAddress to be ObjectType, got %s", homeAddressField.Type)
	}
	if homeAddressField.NestedType == nil || homeAddressField.NestedType.Name != "Address" {
		t.Error("Expected homeAddress to reference Address type")
	}

	// Verify Address type exists and has correct fields
	addressTypeDef, exists := schema.Types["Address"]
	if !exists {
		t.Fatal("Address type not found in schema")
	}

	expectedAddressFields := []string{"street", "city", "zipCode"}
	for _, fieldName := range expectedAddressFields {
		if _, exists := addressTypeDef.Fields[fieldName]; !exists {
			t.Errorf("Expected field '%s' not found in Address type", fieldName)
		}
	}
}

func TestObjectFieldGeneratesCorrectCode(t *testing.T) {
	// Create types with nested objects
	addressType := NewType("Address").
		AddField("street", StringField()).
		AddField("city", StringField())

	userType := NewType("User").
		AddField("id", StringField()).
		AddField("address", ObjectField(addressType))

	schema, err := NewSchema().
		AddType(userType).
		AddType(addressType).
		Build()

	if err != nil {
		t.Fatalf("Failed to build schema: %v", err)
	}

	// Test Go code generation
	goGen := NewGoGenerator("models")
	goCode, err := goGen.GeneratePackage(schema)
	if err != nil {
		t.Fatalf("Failed to generate Go code: %v", err)
	}

	// Should contain both struct definitions
	if !strings.Contains(goCode, "type User struct") {
		t.Error("Generated Go code should contain User struct")
	}
	if !strings.Contains(goCode, "type Address struct") {
		t.Error("Generated Go code should contain Address struct")
	}
	if !strings.Contains(goCode, "Address Address") {
		t.Error("Generated Go code should have Address field of type Address")
	}

	// Test TypeScript code generation
	tsGen := NewTypeScriptGenerator()
	tsCode := tsGen.GenerateModule(schema)

	// Should contain both interface definitions
	if !strings.Contains(tsCode, "export interface User") {
		t.Error("Generated TypeScript code should contain User interface")
	}
	if !strings.Contains(tsCode, "export interface Address") {
		t.Error("Generated TypeScript code should contain Address interface")
	}
	if !strings.Contains(tsCode, "address: Address") {
		t.Error("Generated TypeScript code should have address field of type Address")
	}

	t.Log("Generated Go code:")
	t.Log(goCode)
	t.Log("Generated TypeScript code:")
	t.Log(tsCode)
}