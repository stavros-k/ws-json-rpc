package router

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"ws-json-rpc/backend/pkg/router/generate"
)

// extractParamName extracts the parameter name from a path.
// Currently it does not handle unclosed '{' braces.
func extractParamName(path string) ([]string, error) {
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

	if spec.Group == "" {
		return errors.New("field Group required")
	}

	if spec.Handler == nil {
		return errors.New("field Handler required")
	}

	return nil
}

func generateParameters(spec RouteSpec) ([]generate.ParameterInfo, error) {
	var parameters []generate.ParameterInfo

	// Validate path parameters and collect metadata
	paramsInPath := map[string]struct{}{}
	documentedPathParams := map[string]struct{}{}

	// Extract param names from path
	for section := range strings.SplitSeq(spec.fullPath, "/") {
		paramsName, err := extractParamName(section)
		if err != nil {
			return nil, fmt.Errorf("invalid path %s: %w", spec.fullPath, err)
		}

		for _, paramName := range paramsName {
			paramsInPath[paramName] = struct{}{}
		}
	}

	for name, paramSpec := range spec.Parameters {
		if name == "" {
			return nil, fmt.Errorf("parameter name required for %s %s", spec.method, spec.fullPath)
		}

		if paramSpec.Description == "" {
			return nil, fmt.Errorf("parameter Description required for %s %s", spec.method, spec.fullPath)
		}

		if paramSpec.Type == nil {
			return nil, fmt.Errorf("parameter Type required for %s %s", spec.method, spec.fullPath)
		}

		validInValues := []ParameterIn{ParameterInPath, ParameterInQuery, ParameterInHeader}
		if !slices.Contains(validInValues, paramSpec.In) {
			return nil, fmt.Errorf("parameter In must be one of %v for %s %s", validInValues, spec.method, spec.fullPath)
		}

		parameters = append(parameters, generate.ParameterInfo{
			Name:        name,
			In:          string(paramSpec.In),
			TypeValue:   paramSpec.Type,
			Description: paramSpec.Description,
			Required:    paramSpec.Required,
		})

		if paramSpec.In == ParameterInPath {
			if _, exists := paramsInPath[name]; !exists {
				return nil, fmt.Errorf("documented path parameter %s not found in path", name)
			}

			if !paramSpec.Required {
				return nil, fmt.Errorf("path parameter %s must be required", name)
			}

			documentedPathParams[name] = struct{}{}
		}

		// Validate that all path params are documented
		for name := range paramsInPath {
			if _, exists := documentedPathParams[name]; !exists {
				return nil, fmt.Errorf("path parameter %s not documented", name)
			}
		}
	}

	return parameters, nil
}
