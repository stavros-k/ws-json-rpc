package generator

// import (
// 	"fmt"
// 	"reflect"
// 	"sort"
// 	"strings"
// 	"testing"
// )

// func TestGoParser_ParseTypes(t *testing.T) {
// 	parser := NewGoParser()

// 	// Add test data directory
// 	err := parser.AddDir("./test_data")
// 	if err != nil {
// 		t.Fatalf("Failed to add test directory: %v", err)
// 	}

// 	// Parse the types
// 	err = parser.Parse()
// 	if err != nil {
// 		t.Fatalf("Failed to parse: %v", err)
// 	}

// 	// Test cases for different types
// 	t.Run("User struct", func(t *testing.T) {
// 		testUserStruct(t, parser)
// 	})

// 	t.Run("Address struct", func(t *testing.T) {
// 		testAddressStruct(t, parser)
// 	})

// 	t.Run("EmbeddedExample struct", func(t *testing.T) {
// 		testEmbeddedStruct(t, parser)
// 	})

// 	t.Run("ComplexTypes struct", func(t *testing.T) {
// 		testComplexTypesStruct(t, parser)
// 	})

// 	t.Run("Role enum", func(t *testing.T) {
// 		testRoleEnum(t, parser)
// 	})

// 	t.Run("Status enum", func(t *testing.T) {
// 		testStatusEnum(t, parser)
// 	})

// 	t.Run("Priority enum", func(t *testing.T) {
// 		testPriorityEnum(t, parser)
// 	})
// }

// func testUserStruct(t *testing.T, parser *GoParser) {
// 	typeInfo, exists := parser.types["User"]
// 	if !exists {
// 		t.Fatal("User type not found")
// 	}

// 	// Check basic type info
// 	assertEqual(t, "Name", typeInfo.Name, "User")
// 	assertEqual(t, "Kind", string(typeInfo.Kind), "struct")
// 	assertEqual(t, "Underlying", typeInfo.Underlying, "struct")
// 	assertContains(t, "Comment", typeInfo.Comment.Above, "represents a user")

// 	// Check fields count (excluding unexported and json:"-" fields)
// 	expectedFieldCount := 10 // All exported fields except Internal (json:"-")
// 	assertEqual(t, "Field count", len(typeInfo.Fields), expectedFieldCount)

// 	// Create a map for easier field lookup
// 	fields := make(map[string]FieldInfo)
// 	for _, f := range typeInfo.Fields {
// 		fields[f.Name] = f
// 	}

// 	// Test ID field
// 	idField := fields["ID"]
// 	assertEqual(t, "ID.Type", idField.Type.BaseType, "string")
// 	assertEqual(t, "ID.JSONName", idField.JSONName, "id")
// 	assertFalse(t, "ID.IsPointer", idField.Type.IsPointer)
// 	assertContains(t, "ID.Comment", idField.Comment.Above, "unique identifier")

// 	// Test Email field with omitempty
// 	emailField := fields["Email"]
// 	assertEqual(t, "Email.Type", emailField.Type.BaseType, "string")
// 	assertEqual(t, "Email.JSONName", emailField.JSONName, "email")
// 	assertSliceContains(t, "Email.JSONOptions", emailField.JSONOptions, "omitempty")
// 	assertContains(t, "Email.Comment", emailField.Comment.Inline, "email address")

// 	// Test Address nested struct field
// 	addressField := fields["Address"]
// 	assertEqual(t, "Address.Type", addressField.Type.BaseType, "Address")
// 	assertEqual(t, "Address.JSONName", addressField.JSONName, "address")

// 	// Test Roles slice field
// 	rolesField := fields["Roles"]
// 	assertTrue(t, "Roles.IsSlice", rolesField.Type.IsSlice)
// 	assertEqual(t, "Roles.ValueType", rolesField.Type.ValueType.BaseType, "Role")

// 	// Test Metadata map field
// 	metadataField := fields["Metadata"]
// 	assertTrue(t, "Metadata.IsMap", metadataField.Type.IsMap)
// 	assertEqual(t, "Metadata.KeyType", metadataField.Type.KeyType.BaseType, "string")
// 	assertEqual(t, "Metadata.ValueType", metadataField.Type.ValueType.BaseType, "any")

// 	// Test Manager pointer field
// 	managerField := fields["Manager"]
// 	assertTrue(t, "Manager.IsPointer", managerField.Type.IsPointer)
// 	assertEqual(t, "Manager.BaseType", managerField.Type.BaseType, "User")

// 	// Test Scores array field
// 	scoresField := fields["Scores"]
// 	assertTrue(t, "Scores.IsArray", scoresField.Type.IsArray)
// 	assertEqual(t, "Scores.ArrayLength", scoresField.Type.ArrayLength, "5")
// 	assertEqual(t, "Scores.ValueType", scoresField.Type.ValueType.BaseType, "int")

// 	// Ensure unexported field is not included
// 	if _, found := fields["password"]; found {
// 		t.Error("Unexported field 'password' should not be included")
// 	}

// 	// Ensure json:"-" field is not included
// 	if _, found := fields["Internal"]; found {
// 		t.Error("Field with json:\"-\" tag should not be included")
// 	}
// }

// func testAddressStruct(t *testing.T, parser *GoParser) {
// 	typeInfo, exists := parser.types["Address"]
// 	if !exists {
// 		t.Fatal("Address type not found")
// 	}

// 	assertEqual(t, "Name", typeInfo.Name, "Address")
// 	assertEqual(t, "Kind", string(typeInfo.Kind), "struct")
// 	assertEqual(t, "Field count", len(typeInfo.Fields), 4)

// 	// Check all fields are strings with correct JSON names
// 	expectedFields := map[string]string{
// 		"Street":  "street",
// 		"City":    "city",
// 		"Country": "country",
// 		"ZipCode": "zip_code",
// 	}

// 	for _, field := range typeInfo.Fields {
// 		expectedJSON, ok := expectedFields[field.Name]
// 		if !ok {
// 			t.Errorf("Unexpected field: %s", field.Name)
// 			continue
// 		}
// 		assertEqual(t, fmt.Sprintf("%s.Type", field.Name), field.Type.BaseType, "string")
// 		assertEqual(t, fmt.Sprintf("%s.JSONName", field.Name), field.JSONName, expectedJSON)
// 	}
// }

// func testEmbeddedStruct(t *testing.T, parser *GoParser) {
// 	typeInfo, exists := parser.types["EmbeddedExample"]
// 	if !exists {
// 		t.Fatal("EmbeddedExample type not found")
// 	}

// 	assertEqual(t, "Field count", len(typeInfo.Fields), 3)

// 	// Find embedded fields
// 	var embeddedAddress *FieldInfo
// 	var embeddedUser *FieldInfo
// 	var nameField *FieldInfo

// 	for i := range typeInfo.Fields {
// 		field := &typeInfo.Fields[i]
// 		if field.Name == "" && field.Type.BaseType == "Address" {
// 			embeddedAddress = field
// 		} else if field.Name == "" && field.Type.BaseType == "User" {
// 			embeddedUser = field
// 		} else if field.Name == "Name" {
// 			nameField = field
// 		}
// 	}

// 	// Test embedded Address
// 	if embeddedAddress == nil {
// 		t.Fatal("Embedded Address field not found")
// 	}
// 	assertTrue(t, "Address.IsEmbedded", embeddedAddress.Type.IsEmbedded)
// 	assertEqual(t, "Address.BaseType", embeddedAddress.Type.BaseType, "Address")

// 	// Test embedded *User
// 	if embeddedUser == nil {
// 		t.Fatal("Embedded User field not found")
// 	}
// 	assertTrue(t, "User.IsEmbedded", embeddedUser.Type.IsEmbedded)
// 	assertTrue(t, "User.IsPointer", embeddedUser.Type.IsPointer)
// 	assertEqual(t, "User.BaseType", embeddedUser.Type.BaseType, "User")

// 	// Test regular field
// 	if nameField == nil {
// 		t.Fatal("Name field not found")
// 	}
// 	assertEqual(t, "Name.Type", nameField.Type.BaseType, "string")
// }

// func testComplexTypesStruct(t *testing.T, parser *GoParser) {
// 	typeInfo, exists := parser.types["ComplexTypes"]
// 	if !exists {
// 		t.Fatal("ComplexTypes type not found")
// 	}

// 	fields := make(map[string]FieldInfo)
// 	for _, f := range typeInfo.Fields {
// 		fields[f.Name] = f
// 	}

// 	// Test SliceOfPointers: []*User
// 	sliceOfPointers := fields["SliceOfPointers"]
// 	assertTrue(t, "SliceOfPointers.IsSlice", sliceOfPointers.Type.IsSlice)
// 	assertTrue(t, "SliceOfPointers.ValueType.IsPointer", sliceOfPointers.Type.ValueType.IsPointer)
// 	assertEqual(t, "SliceOfPointers.ValueType.BaseType", sliceOfPointers.Type.ValueType.BaseType, "User")

// 	// Test MapOfSlices: map[string][]string
// 	mapOfSlices := fields["MapOfSlices"]
// 	assertTrue(t, "MapOfSlices.IsMap", mapOfSlices.Type.IsMap)
// 	assertEqual(t, "MapOfSlices.KeyType", mapOfSlices.Type.KeyType.BaseType, "string")
// 	assertTrue(t, "MapOfSlices.ValueType.IsSlice", mapOfSlices.Type.ValueType.IsSlice)
// 	assertEqual(t, "MapOfSlices.ValueType.ValueType", mapOfSlices.Type.ValueType.ValueType.BaseType, "string")

// 	// Test NestedMap: map[string]map[int]bool
// 	nestedMap := fields["NestedMap"]
// 	assertTrue(t, "NestedMap.IsMap", nestedMap.Type.IsMap)
// 	assertEqual(t, "NestedMap.KeyType", nestedMap.Type.KeyType.BaseType, "string")
// 	assertTrue(t, "NestedMap.ValueType.IsMap", nestedMap.Type.ValueType.IsMap)
// 	assertEqual(t, "NestedMap.ValueType.KeyType", nestedMap.Type.ValueType.KeyType.BaseType, "int")
// 	assertEqual(t, "NestedMap.ValueType.ValueType", nestedMap.Type.ValueType.ValueType.BaseType, "bool")

// 	// Test SliceOfMaps: []map[string]int
// 	sliceOfMaps := fields["SliceOfMaps"]
// 	assertTrue(t, "SliceOfMaps.IsSlice", sliceOfMaps.Type.IsSlice)
// 	assertTrue(t, "SliceOfMaps.ValueType.IsMap", sliceOfMaps.Type.ValueType.IsMap)
// 	assertEqual(t, "SliceOfMaps.ValueType.KeyType", sliceOfMaps.Type.ValueType.KeyType.BaseType, "string")
// 	assertEqual(t, "SliceOfMaps.ValueType.ValueType", sliceOfMaps.Type.ValueType.ValueType.BaseType, "int")
// }

// func testRoleEnum(t *testing.T, parser *GoParser) {
// 	typeInfo, exists := parser.types["Role"]
// 	if !exists {
// 		t.Fatal("Role type not found")
// 	}

// 	assertEqual(t, "Name", typeInfo.Name, "Role")
// 	assertEqual(t, "Kind", string(typeInfo.Kind), "enum")
// 	assertEqual(t, "Underlying", typeInfo.Underlying, "string")
// 	assertEqual(t, "EnumValues count", len(typeInfo.EnumValues), 4)

// 	// Check enum values
// 	expectedValues := map[string]string{
// 		"RoleAdmin":     `"admin"`,
// 		"RoleUser":      `"user"`,
// 		"RoleGuest":     `"guest"`,
// 		"RoleModerator": `"moderator"`,
// 	}

// 	for _, ev := range typeInfo.EnumValues {
// 		expected, ok := expectedValues[ev.Name]
// 		if !ok {
// 			t.Errorf("Unexpected enum value: %s", ev.Name)
// 			continue
// 		}
// 		assertEqual(t, fmt.Sprintf("%s.Value", ev.Name), ev.Value, expected)

// 		// Check comments
// 		if ev.Name == "RoleAdmin" {
// 			assertContains(t, "RoleAdmin.Comment", ev.Comment.Above, "Administrator")
// 		}
// 		if ev.Name == "RoleGuest" {
// 			assertContains(t, "RoleGuest.Comment.Inline", ev.Comment.Inline, "guest")
// 		}
// 	}
// }

// func testStatusEnum(t *testing.T, parser *GoParser) {
// 	typeInfo, exists := parser.types["Status"]
// 	if !exists {
// 		t.Fatal("Status type not found")
// 	}

// 	assertEqual(t, "Name", typeInfo.Name, "Status")
// 	assertEqual(t, "Kind", string(typeInfo.Kind), "enum")
// 	assertEqual(t, "Underlying", typeInfo.Underlying, "int")
// 	assertEqual(t, "EnumValues count", len(typeInfo.EnumValues), 3)

// 	// Check enum values
// 	expectedValues := map[string]string{
// 		"StatusActive":   "1",
// 		"StatusInactive": "2",
// 		"StatusPending":  "3",
// 	}

// 	for _, ev := range typeInfo.EnumValues {
// 		expected, ok := expectedValues[ev.Name]
// 		if !ok {
// 			t.Errorf("Unexpected enum value: %s", ev.Name)
// 			continue
// 		}
// 		assertEqual(t, fmt.Sprintf("%s.Value", ev.Name), ev.Value, expected)
// 	}
// }

// func testPriorityEnum(t *testing.T, parser *GoParser) {
// 	typeInfo, exists := parser.types["Priority"]
// 	if !exists {
// 		t.Fatal("Priority type not found")
// 	}

// 	assertEqual(t, "Name", typeInfo.Name, "Priority")
// 	assertEqual(t, "Kind", string(typeInfo.Kind), "enum")
// 	assertEqual(t, "Underlying", typeInfo.Underlying, "string")
// 	assertEqual(t, "EnumValues count", len(typeInfo.EnumValues), 3)
// }

// // Helper functions for assertions

// func assertEqual(t *testing.T, name string, got, want interface{}) {
// 	t.Helper()
// 	if !reflect.DeepEqual(got, want) {
// 		t.Errorf("%s mismatch:\n  got:  %v\n  want: %v", name, got, want)
// 	}
// }

// func assertContains(t *testing.T, name string, haystack, needle string) {
// 	t.Helper()
// 	if !strings.Contains(haystack, needle) {
// 		t.Errorf("%s: expected to contain '%s', got '%s'", name, needle, haystack)
// 	}
// }

// func assertTrue(t *testing.T, name string, value bool) {
// 	t.Helper()
// 	if !value {
// 		t.Errorf("%s: expected true, got false", name)
// 	}
// }

// func assertFalse(t *testing.T, name string, value bool) {
// 	t.Helper()
// 	if value {
// 		t.Errorf("%s: expected false, got true", name)
// 	}
// }

// func assertSliceContains(t *testing.T, name string, slice []string, value string) {
// 	t.Helper()
// 	for _, v := range slice {
// 		if v == value {
// 			return
// 		}
// 	}
// 	t.Errorf("%s: slice %v does not contain '%s'", name, slice, value)
// }

// // Benchmark tests

// func BenchmarkParse(b *testing.B) {
// 	for i := 0; i < b.N; i++ {
// 		parser := NewGoParser()
// 		_ = parser.AddDir("./test_data")
// 		_ = parser.Parse()
// 	}
// }

// func BenchmarkParseStructs(b *testing.B) {
// 	parser := NewGoParser()
// 	_ = parser.AddDir("./test_data")

// 	b.ResetTimer()
// 	for i := 0; i < b.N; i++ {
// 		parser.types = make(map[string]*TypeInfo)
// 		_ = parser.forEachDecl(parser.extractTypeMetadata)
// 		_ = parser.forEachDecl(parser.processDeclaration)
// 	}
// }

// // Table-driven test for field type analysis
// func TestAnalyzeFieldType(t *testing.T) {
// 	// This would require creating AST nodes programmatically
// 	// which is complex, so we'll test it through the integration tests above
// }

// // Test error cases
// func TestParserErrors(t *testing.T) {
// 	t.Run("Invalid directory", func(t *testing.T) {
// 		parser := NewGoParser()
// 		err := parser.AddDir("./non_existent_dir")
// 		if err == nil {
// 			t.Error("Expected error for non-existent directory")
// 		}
// 	})

// 	t.Run("Duplicate package", func(t *testing.T) {
// 		parser := NewGoParser()
// 		err := parser.AddDir("./test_data")
// 		if err != nil {
// 			t.Fatalf("First add failed: %v", err)
// 		}

// 		err = parser.AddDir("./test_data")
// 		if err == nil {
// 			t.Error("Expected error for duplicate package")
// 		}
// 	})

// 	t.Run("Struct with field without json name", func(t *testing.T) {
// 		parser := NewGoParser()
// 		err := parser.AddDir("./test_data")
// 		if err != nil {
// 			t.Fatalf("Failed to add test directory: %v", err)
// 		}

// 		// Parse the types
// 		err = parser.Parse()
// 		if err != nil {
// 			t.Fatalf("Failed to parse: %v", err)
// 		}
// 	})
// }

// // Test helper to verify all types were found
// func TestAllTypesFound(t *testing.T) {
// 	parser := NewGoParser()
// 	_ = parser.AddDir("./test_data")
// 	_ = parser.Parse()

// 	expectedTypes := []string{
// 		"User", "Address", "EmbeddedExample", "ComplexTypes",
// 		"Role", "Status", "Priority",
// 	}

// 	for _, typeName := range expectedTypes {
// 		if _, exists := parser.types[typeName]; !exists {
// 			t.Errorf("Expected type not found: %s", typeName)
// 		}
// 	}

// 	// Verify count
// 	if len(parser.types) != len(expectedTypes) {
// 		// Get actual type names for debugging
// 		var actualTypes []string
// 		for name := range parser.types {
// 			actualTypes = append(actualTypes, name)
// 		}
// 		sort.Strings(actualTypes)
// 		t.Errorf("Type count mismatch. Expected %d, got %d\nActual types: %v",
// 			len(expectedTypes), len(parser.types), actualTypes)
// 	}
// }
