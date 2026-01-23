package router

import (
	"reflect"
	"testing"
)

func TestExtractParamName(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected []string
	}{
		{
			name:     "no params",
			path:     "/users",
			expected: []string{},
		},
		{
			name:     "simple one param",
			path:     "/users/{userID}",
			expected: []string{"userID"},
		},
		{
			name:     "simple one param with regex",
			path:     "/users/{userID:[0-9]+}",
			expected: []string{"userID"},
		},
		{
			name:     "multiple params",
			path:     "/users/{userID}/{userName}",
			expected: []string{"userID", "userName"},
		},
		{
			name:     "multiple params with regex",
			path:     "/users/{userID:[0-9]+}/{userName:[a-z]+}",
			expected: []string{"userID", "userName"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractParamName(tt.path)
			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("extractParamName(%q) = %v, want %v", tt.path, got, tt.expected)
			}
		})
	}
}
