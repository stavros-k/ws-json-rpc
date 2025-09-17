package builder

import (
	"fmt"
	"testing"
)

func ExampleBuilder() {
	// Create a User type with various field types
	userType := NewType("User").
		Description("User represents a system user").
		AddField("id", StringField().Description("Unique identifier")).
		AddField("name", StringField().Optional().JSONOmitEmpty().Description("User's full name")).
		AddField("email", StringField().JSONName("email_address")).
		AddField("age", IntegerField().Optional()).
		AddField("isActive", BooleanField().JSONName("is_active")).
		AddField("createdAt", JSONDateField().Description("Account creation date")).
		AddField("role", EnumField("admin", "user", "guest"))

	// Create an Address type for nested objects
	addressType := NewType("Address").
		AddField("street", StringField()).
		AddField("city", StringField()).
		AddField("zipCode", StringField().JSONName("zip_code"))

	// Add address to user
	userWithAddress := userType.AddField("address", ObjectField(addressType).Optional())

	// Create a schema
	schema, err := NewSchema().
		AddType(userWithAddress).
		AddType(addressType).
		Build()

	if err != nil {
		fmt.Printf("Failed to build schema: %v\n", err)
		return
	}

	// Generate Go code
	goGen := NewGoGenerator("models")
	goCode, err := goGen.GeneratePackage(schema)
	if err != nil {
		fmt.Printf("Failed to generate Go code: %v\n", err)
		return
	}
	fmt.Println("Generated Go code:")
	fmt.Println(goCode)

	// Generate TypeScript code
	tsGen := NewTypeScriptGenerator()
	tsCode := tsGen.GenerateModule(schema)
	fmt.Println("Generated TypeScript code:")
	fmt.Println(tsCode)
}

func TestBuilderDSL(t *testing.T) {
	// Test basic field creation - we need to use AddField to set the name
	userType := NewType("TestUser").
		AddField("name", StringField().Optional().JSONOmitEmpty())

	typeDef, err := userType.Build()
	if err != nil {
		t.Fatalf("Failed to build type: %v", err)
	}
	nameField := typeDef.Fields["name"]

	if nameField.Name != "name" {
		t.Errorf("Expected field name 'name', got %s", nameField.Name)
	}
	if !nameField.Optional {
		t.Error("Expected field to be optional")
	}
	if !nameField.JSONOmitEmpty {
		t.Error("Expected field to have JSONOmitEmpty")
	}

	// Test enum field
	enumType := NewType("TestEnum").
		AddField("status", EnumField("active", "inactive", "pending"))
	enumTypeDef, err := enumType.Build()
	if err != nil {
		t.Fatalf("Failed to build enum type: %v", err)
	}
	enumField := enumTypeDef.Fields["status"]

	if len(enumField.EnumValues) != 3 {
		t.Errorf("Expected 3 enum values, got %d", len(enumField.EnumValues))
	}

	// Test custom field
	customType := NewType("TestCustom").
		AddField("customField", CustomField("MyCustomType"))
	customTypeDef, err := customType.Build()
	if err != nil {
		t.Fatalf("Failed to build custom type: %v", err)
	}
	customField := customTypeDef.Fields["customField"]

	if customField.CustomType != "MyCustomType" {
		t.Errorf("Expected custom type 'MyCustomType', got %s", customField.CustomType)
	}

	// Test JSONDate field
	dateType := NewType("TestDate").
		AddField("timestamp", JSONDateField())
	dateTypeDef, err := dateType.Build()
	if err != nil {
		t.Fatalf("Failed to build date type: %v", err)
	}
	dateField := dateTypeDef.Fields["timestamp"]

	if dateField.Type != JSONDateType {
		t.Errorf("Expected JSONDateType, got %s", dateField.Type)
	}
}

func TestTypeBuilder(t *testing.T) {
	// Create a simple type
	userType := NewType("User").
		Description("A user in the system").
		AddField("id", StringField()).
		AddField("name", StringField().Optional())

	typeDef, err := userType.Build()
	if err != nil {
		t.Fatalf("Failed to build type: %v", err)
	}

	if typeDef.Name != "User" {
		t.Errorf("Expected type name 'User', got %s", typeDef.Name)
	}
	if typeDef.Description != "A user in the system" {
		t.Errorf("Expected description, got %s", typeDef.Description)
	}
	if len(typeDef.Fields) != 2 {
		t.Errorf("Expected 2 fields, got %d", len(typeDef.Fields))
	}

	// Test field retrieval
	nameField, exists := typeDef.Fields["name"]
	if !exists {
		t.Error("Expected 'name' field to exist")
	}
	if !nameField.Optional {
		t.Error("Expected 'name' field to be optional")
	}
}

func TestSchemaBuilder(t *testing.T) {
	userType := NewType("User").
		AddField("id", StringField()).
		AddField("name", StringField())

	addressType := NewType("Address").
		AddField("street", StringField()).
		AddField("city", StringField())

	schema, err := NewSchema().
		AddType(userType).
		AddType(addressType).
		Build()

	if err != nil {
		t.Fatalf("Failed to build schema: %v", err)
	}

	if len(schema.Types) != 2 {
		t.Errorf("Expected 2 types in schema, got %d", len(schema.Types))
	}

	// Test type retrieval
	retrievedUser, exists := schema.Types["User"]
	if !exists {
		t.Error("Expected 'User' type to exist in schema")
	}
	if retrievedUser.Name != "User" {
		t.Errorf("Expected type name 'User', got %s", retrievedUser.Name)
	}
}
