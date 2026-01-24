package generate

import (
	"fmt"
	"os"
	"sort"
	"ws-json-rpc/backend/pkg/utils"
)

// GenerateAPIDocs generates a custom API documentation JSON file
// Similar to api_docs.json for RPC, but for REST APIs
func GenerateAPIDocs(doc *APIDocumentation, outputPath string) error {
	// Sort types and routes for deterministic output
	sortedDoc := sortDocumentation(doc)

	// Create output file
	f, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create documentation file: %w", err)
	}
	defer f.Close()

	// Write JSON using utils
	if err := utils.ToJSONStreamIndent(f, sortedDoc); err != nil {
		return fmt.Errorf("failed to write documentation: %w", err)
	}

	return nil
}

// sortDocumentation creates a copy of the documentation with sorted fields
func sortDocumentation(doc *APIDocumentation) *APIDocumentation {
	sorted := &APIDocumentation{
		Types:  make(map[string]*TypeInfo),
		Routes: make(map[string]*PathRoutes),
	}

	// Copy and sort types
	for name, typeInfo := range doc.Types {
		sortedType := *typeInfo

		// Sort references
		sort.Strings(sortedType.References)

		// Sort used by
		sortUsageInfo(sortedType.UsedBy)

		sorted.Types[name] = &sortedType
	}

	// Copy routes (keyed by path)
	for path, pathRoutes := range doc.Routes {
		sorted.Routes[path] = pathRoutes
	}

	return sorted
}

// sortUsageInfo sorts usage information for deterministic output
func sortUsageInfo(usages []UsageInfo) {
	sort.Slice(usages, func(i, j int) bool {
		if usages[i].Location != usages[j].Location {
			return usages[i].Location < usages[j].Location
		}
		if usages[i].Target != usages[j].Target {
			return usages[i].Target < usages[j].Target
		}
		return usages[i].Role < usages[j].Role
	})
}

// GetTypesSortedByName returns type names sorted alphabetically
func GetTypesSortedByName(doc *APIDocumentation) []string {
	names := make([]string, 0, len(doc.Types))
	for name := range doc.Types {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// GetRoutesSortedByPath returns routes sorted by path then method
func GetRoutesSortedByPath(doc *APIDocumentation) []*RouteInfo {
	routes := make([]*RouteInfo, 0)

	// Flatten PathRoutes into a list of RouteInfo
	for _, pathRoutes := range doc.Routes {
		for _, route := range pathRoutes.Routes {
			routes = append(routes, route)
		}
	}

	// Sort by path then method
	sort.Slice(routes, func(i, j int) bool {
		if routes[i].Path != routes[j].Path {
			return routes[i].Path < routes[j].Path
		}
		return routes[i].Method < routes[j].Method
	})

	return routes
}
