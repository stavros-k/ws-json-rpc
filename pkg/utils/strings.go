package utils

import "strings"

// Common initialisms that should be all-caps in PascalCase
var initialisms = map[string]struct{}{
	"id":   {},
	"url":  {},
	"uri":  {},
	"api":  {},
	"uuid": {},
	"ip":   {},
}

// ToPascalCase converts a camelCase string to PascalCase with proper initialism handling.
// Common initialisms like "id", "url", "api", "uuid" are all-caps: ID, URL, API, UUID.
// Examples: "id" -> "ID", "userId" -> "UserID", "apiKey" -> "APIKey", "roomDifficulty" -> "RoomDifficulty"
func ToPascalCase(s string) string {
	if s == "" {
		return s
	}

	lower := strings.ToLower(s)

	// Check if the entire string is an initialism
	if _, ok := initialisms[lower]; ok {
		return strings.ToUpper(lower)
	}

	// Split camelCase into words
	words := SplitCamelCase(s)

	// Process each word: if it's an initialism, make it all caps
	// Otherwise, capitalize first letter
	var result strings.Builder
	for _, word := range words {
		if len(word) == 0 {
			continue
		}
		wordLower := strings.ToLower(word)
		if _, ok := initialisms[wordLower]; ok {
			result.WriteString(strings.ToUpper(wordLower))
		} else {
			result.WriteString(strings.ToUpper(word[:1]) + word[1:])
		}
	}

	return result.String()
}

// SplitCamelCase splits a camelCase string into words.
// Examples: "userId" -> ["user", "id"], "callbackUrl" -> ["callback", "url"]
func SplitCamelCase(s string) []string {
	if s == "" {
		return nil
	}

	var words []string
	var currentWord strings.Builder

	for i, r := range s {
		if i == 0 {
			currentWord.WriteRune(r)
			continue
		}

		// Check if this is the start of a new word (uppercase letter)
		if r >= 'A' && r <= 'Z' {
			// Save the current word if it has content
			if currentWord.Len() > 0 {
				words = append(words, currentWord.String())
				currentWord.Reset()
			}
		}

		currentWord.WriteRune(r)
	}

	// Add the last word
	if currentWord.Len() > 0 {
		words = append(words, currentWord.String())
	}

	return words
}
