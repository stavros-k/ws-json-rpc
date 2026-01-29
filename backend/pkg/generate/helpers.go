package generate

import (
	"errors"
	"strings"
)

// ExtractParamName extracts the parameter name from a path.
// Currently it does not handle unclosed '{' braces.
func ExtractParamName(path string) ([]string, error) {
	dirtyParams := []string{}
	cleanParams := []string{}

	openBracket := strings.Count(path, "{")

	closeBracket := strings.Count(path, "}")
	if openBracket != closeBracket {
		return nil, errors.New("mismatched number of '{' and '}' in path")
	}
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

	return cleanParams, nil
}
