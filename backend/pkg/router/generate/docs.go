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
	sortDocumentation(doc)

	// Create output file
	f, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create documentation file: %w", err)
	}
	defer f.Close()

	// Write JSON using utils
	if err := utils.ToJSONStreamIndent(f, doc); err != nil {
		return fmt.Errorf("failed to write documentation: %w", err)
	}

	return nil
}

// sortDocumentation creates a copy of the documentation with sorted fields
func sortDocumentation(doc *APIDocumentation) {
	for name, typeInfo := range doc.Types {
		// Sort references
		sort.Strings(typeInfo.References)

		// Sort referenced by
		sort.Strings(typeInfo.ReferencedBy)

		// Sort used by
		sortUsageInfo(typeInfo.UsedBy)

		doc.Types[name] = typeInfo
	}
}

// sortUsageInfo sorts usage information for deterministic output
func sortUsageInfo(usages []UsageInfo) {
	sort.Slice(usages, func(i, j int) bool {
		if usages[i].OperationID != usages[j].OperationID {
			return usages[i].OperationID < usages[j].OperationID
		}
		return usages[i].Role < usages[j].Role
	})
}
