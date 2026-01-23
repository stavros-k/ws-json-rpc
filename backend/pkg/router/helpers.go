package router

import (
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
)

func ptr[T any](v T) *T {
	return &v
}

func jsonContent(schema *openapi3.SchemaRef, examples map[string]any) openapi3.Content {
	return openapi3.Content{
		"application/json": &openapi3.MediaType{
			Schema:   schema,
			Examples: examplesToOpenAPIExamples(examples),
		},
	}
}

func extractParamName(path string) []string {
	dirtyParams := []string{}
	cleanParams := []string{}

	// Find the content between '{' and '}'
	// Examples:
	// - {userID} -> userID
	// - {userID:[0-9]+} -> userID:[0-9]+
	start := -1
	for i, ch := range path {
		if ch == '{' {
			start = i + 1
		} else if ch == '}' && start >= 0 {
			dirtyParams = append(dirtyParams, path[start:i])
			start = -1
		}
	}

	// Now split on ':' to remove any regex matchers
	// Examples:
	// - userID -> userID
	// - userID:[0-9]+ -> userID
	for _, param := range dirtyParams {
		parts := strings.Split(param, ":")
		param = parts[0]
		if param != "" {
			cleanParams = append(cleanParams, param)
		}
	}

	return cleanParams
}
