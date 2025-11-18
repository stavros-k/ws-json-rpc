package generate

import (
	"bytes"
	"fmt"
	"sort"
)

// GoOptions contains options for Go code generation.
type GoOptions struct {
	PackageName string   // Go package name
	OutputFile  string   // Output file path
	AddImports  []string // Additional imports to include
}

// TSOptions contains options for TypeScript code generation.
type TSOptions struct {
	OutputFile string // Output file path
}

// CSharpOptions contains options for C# code generation.
type CSharpOptions struct {
	Namespace  string // C# namespace
	OutputFile string // Output file path
}

// MethodMapping represents a method's request/response types for API client type generation.
type MethodMapping struct {
	Name       string // Method name (e.g., "user.create")
	ParamType  string // Request type name or "null"
	ResultType string // Response type name or "null"
}

// EventMapping represents an event's data type for API client type generation.
type EventMapping struct {
	Name       string // Event name (e.g., "data.created")
	ResultType string // Event data type name or "null"
}

// GenerateTypeScriptAPIMappings generates TypeScript type mappings for methods and events.
func GenerateTypeScriptAPIMappings(methods []MethodMapping, events []EventMapping) string {
	var buf bytes.Buffer

	// Generate ApiMethods mapping
	buf.WriteString("/**\n")
	buf.WriteString(" * Type mapping for RPC methods.\n")
	buf.WriteString(" * Maps method names to their request and response types.\n")
	buf.WriteString(" */\n")
	buf.WriteString("export type ApiMethods = {\n")

	// Sort method names for deterministic output
	sortedMethods := make([]MethodMapping, len(methods))
	copy(sortedMethods, methods)
	sort.Slice(sortedMethods, func(i, j int) bool {
		return sortedMethods[i].Name < sortedMethods[j].Name
	})

	for _, method := range sortedMethods {
		reqType := method.ParamType
		if reqType == "null" {
			reqType = "never"
		}
		resType := method.ResultType
		if resType == "null" {
			resType = "never"
		}
		buf.WriteString(fmt.Sprintf("  \"%s\": { req: %s; res: %s };\n", method.Name, reqType, resType))
	}
	buf.WriteString("};\n\n")

	// Generate ApiEvents mapping
	buf.WriteString("/**\n")
	buf.WriteString(" * Type mapping for WebSocket events.\n")
	buf.WriteString(" * Maps event names to their data types.\n")
	buf.WriteString(" */\n")
	buf.WriteString("export type ApiEvents = {\n")

	// Sort event names for deterministic output
	sortedEvents := make([]EventMapping, len(events))
	copy(sortedEvents, events)
	sort.Slice(sortedEvents, func(i, j int) bool {
		return sortedEvents[i].Name < sortedEvents[j].Name
	})

	for _, event := range sortedEvents {
		dataType := event.ResultType
		if dataType == "null" {
			dataType = "never"
		}
		buf.WriteString(fmt.Sprintf("  \"%s\": { data: %s };\n", event.Name, dataType))
	}
	buf.WriteString("};\n")

	return buf.String()
}
