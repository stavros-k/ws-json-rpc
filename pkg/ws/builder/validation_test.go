package builder

import (
	"strings"
	"testing"
)

func TestDuplicateFieldValidation(t *testing.T) {
	// Test duplicate field names
	userType := NewType("User").
		AddField("id", StringField()).
		AddField("name", StringField()).
		AddField("id", IntegerField()) // Duplicate field name

	_, err := userType.Build()
	if err == nil {
		t.Error("Expected error for duplicate field name")
	}

	if !strings.Contains(err.Error(), "duplicate field name: id") {
		t.Errorf("Expected duplicate field error, got: %v", err)
	}
}

func TestEmptyFieldNameValidation(t *testing.T) {
	// Test empty field name
	userType := NewType("User").
		AddField("", StringField())

	_, err := userType.Build()
	if err == nil {
		t.Error("Expected error for empty field name")
	}

	if !strings.Contains(err.Error(), "field name cannot be empty") {
		t.Errorf("Expected empty field name error, got: %v", err)
	}
}

func TestEmptyTypeNameValidation(t *testing.T) {
	// Test empty type name
	userType := NewType("").
		AddField("id", StringField())

	_, err := userType.Build()
	if err == nil {
		t.Error("Expected error for empty type name")
	}

	if !strings.Contains(err.Error(), "type name cannot be empty") {
		t.Errorf("Expected empty type name error, got: %v", err)
	}
}

func TestDuplicateTypeValidation(t *testing.T) {
	userType1 := NewType("User").AddField("id", StringField())
	userType2 := NewType("User").AddField("name", StringField())

	schema := NewSchema().
		AddType(userType1).
		AddType(userType2) // Duplicate type name

	_, err := schema.Build()
	if err == nil {
		t.Error("Expected error for duplicate type name")
	}

	if !strings.Contains(err.Error(), "duplicate type name: User") {
		t.Errorf("Expected duplicate type name error, got: %v", err)
	}
}

func TestValidSchema(t *testing.T) {
	// Test valid schema with no errors
	userType := NewType("User").
		AddField("id", StringField()).
		AddField("name", StringField())

	addressType := NewType("Address").
		AddField("street", StringField())

	schema := NewSchema().
		AddType(userType).
		AddType(addressType)

	result, err := schema.Build()
	if err != nil {
		t.Errorf("Expected no error for valid schema, got: %v", err)
	}

	if len(result.Types) != 2 {
		t.Errorf("Expected 2 types in schema, got %d", len(result.Types))
	}
}

func TestMultipleErrors(t *testing.T) {
	// Test type with multiple errors
	userType := NewType("User").
		AddField("id", StringField()).
		AddField("", StringField()). // Empty field name
		AddField("id", IntegerField()) // Duplicate field name

	_, err := userType.Build()
	if err == nil {
		t.Error("Expected error for multiple validation issues")
	}

	errStr := err.Error()
	if !strings.Contains(errStr, "field name cannot be empty") {
		t.Error("Expected empty field name error")
	}
	if !strings.Contains(errStr, "duplicate field name: id") {
		t.Error("Expected duplicate field name error")
	}
}