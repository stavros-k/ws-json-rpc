package router

import (
	"errors"
	"strings"
)

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

// sanitizePath removes double slashes and trailing slashes from a path.
func sanitizePath(path string) string {
	cleanPath := path
	for strings.Contains(cleanPath, "//") {
		cleanPath = strings.ReplaceAll(cleanPath, "//", "/")
	}

	cleanPath = strings.TrimSuffix(cleanPath, "/")
	if cleanPath == "" {
		cleanPath = "/"
	}

	return cleanPath
}

// validateRouteSpec validates a RouteSpec.
func validateRouteSpec(spec RouteSpec) error {
	if spec.OperationID == "" {
		return errors.New("field OperationID required")
	}

	if spec.Summary == "" {
		return errors.New("field Summary required")
	}

	if spec.Description == "" {
		return errors.New("field Description required")
	}

	if len(spec.Tags) == 0 {
		return errors.New("field Tags requires at least one tag")
	}

	return nil
}
