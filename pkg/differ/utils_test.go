package differ

import (
	"testing"
)

func TestEqualStringSlices(t *testing.T) {
	tests := []struct {
		name     string
		a        []string
		b        []string
		expected bool
	}{
		{"Empty slices", []string{}, []string{}, true},
		{"Same order", []string{"a", "b", "c"}, []string{"a", "b", "c"}, true},
		{"Different order", []string{"a", "b", "c"}, []string{"c", "a", "b"}, true},
		{"Different length", []string{"a", "b"}, []string{"a", "b", "c"}, false},
		{"Different content", []string{"a", "b", "c"}, []string{"a", "b", "d"}, false},
		{"One nil", nil, []string{"a"}, false},
		{"Both nil", nil, nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := equalStringSlices(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("equalStringSlices(%v, %v) = %v, want %v", tt.a, tt.b, result, tt.expected)
			}
		})
	}
}

func TestGenerateSummary(t *testing.T) {
	tests := []struct {
		name     string
		result   *DiffResult
		expected string
	}{
		{
			name: "No changes",
			result: &DiffResult{
				Semantic:     []ConfigDiff{},
				Dependencies: []ConfigDiff{},
				Performance:  []ConfigDiff{},
				HasChanges:   false,
			},
			expected: "No semantic differences found",
		},
		{
			name: "Only semantic changes",
			result: &DiffResult{
				Semantic:     []ConfigDiff{{}},
				Dependencies: []ConfigDiff{},
				Performance:  []ConfigDiff{},
				HasChanges:   true,
			},
			expected: "semantic changes (1 total changes)",
		},
		{
			name: "Multiple change types",
			result: &DiffResult{
				Semantic:     []ConfigDiff{{}, {}},
				Dependencies: []ConfigDiff{{}},
				Performance:  []ConfigDiff{{}, {}, {}},
				HasChanges:   true,
			},
			expected: "semantic changes, dependency changes, performance-related changes (6 total changes)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateSummary(tt.result)
			if result != tt.expected {
				t.Errorf("generateSummary() = '%s', want '%s'", result, tt.expected)
			}
		})
	}
}
