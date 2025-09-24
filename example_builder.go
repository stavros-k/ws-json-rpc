package main

// import (
// 	"fmt"
// 	"ws-json-rpc/pkg/ws/builder"
// )

// func main2() {
// 	methods := builder.NewEnumType("MethodKind", "subscribe", "unsubscribe", "ping", "echo", "add", "double")

// 	// Create an Address type for nested objects
// 	addressType := builder.NewType("Address").
// 		AddField("Street", builder.StringField()).
// 		AddField("City", builder.StringField()).
// 		AddField("ZipCode", builder.StringField().JSONName("zip_code"))

// 	// Create a User type with various field types
// 	userType := builder.NewType("User").
// 		Description("User represents a system user").
// 		AddField("ID", builder.StringField().Description("Unique identifier").JSONName("id")).
// 		AddField("Name", builder.StringField().Optional().JSONOmitEmpty().Description("User's full name")).
// 		AddField("Email", builder.StringField().JSONName("email_address")).
// 		AddField("Age", builder.IntegerField().Optional()).
// 		AddField("IsActive", builder.BooleanField().JSONName("is_active")).
// 		AddField("CreatedAt", builder.JSONDateField().Description("Account creation date")).
// 		AddField("Role", builder.EnumField("admin", "user", "guest")).
// 		AddField("Address", builder.ObjectField(addressType).Optional()).
// 		AddField("HomeAddress", builder.ObjectField(addressType).Optional())

// 	// Create a schema
// 	schema, err := builder.NewSchema().
// 		AddType(methods).
// 		AddType(userType).
// 		AddType(addressType).
// 		Build()

// 	if err != nil {
// 		fmt.Printf("Schema validation error: %v\n", err)
// 		return
// 	}

// 	// Generate Go code
// 	goGen := builder.NewGoGenerator("models")
// 	goCode, err := goGen.GeneratePackage(schema)
// 	if err != nil {
// 		fmt.Printf("Go code generation error: %v\n", err)
// 		return
// 	}
// 	fmt.Println("Generated Go code:")
// 	fmt.Println(goCode)

// 	// Generate TypeScript code
// 	tsGen := builder.NewTypeScriptGenerator()
// 	tsCode := tsGen.GenerateModule(schema)
// 	fmt.Println("\nGenerated TypeScript code with enhanced utilities:")
// 	fmt.Println(tsCode)
// }
