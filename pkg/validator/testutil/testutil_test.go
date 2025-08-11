package testutil

import (
	"path/filepath"
	"testing"
)

func TestGenerateDescription(t *testing.T) {
	tests := []struct {
		scenarioName string
		expected     string
	}{
		{"scenario-1", "Duplicate script blocks consolidation"},
		{"scenario-2", "Complex include consolidation"},
		{"unknown-scenario", "Refactoring scenario unknown-scenario"},
	}

	for _, test := range tests {
		t.Run(test.scenarioName, func(t *testing.T) {
			result := GenerateDescription(test.scenarioName)
			if result != test.expected {
				t.Errorf("Expected '%s', got '%s'", test.expected, result)
			}
		})
	}
}

func TestGenerateRealisticDescription(t *testing.T) {
	tests := []struct {
		scenarioName string
		expected     string
	}{
		{"flask-microservice", "Realistic Flask microservice CI/CD pipeline optimization"},
		{"unknown-app", "Realistic application scenario: unknown-app"},
	}

	for _, test := range tests {
		t.Run(test.scenarioName, func(t *testing.T) {
			result := GenerateRealisticDescription(test.scenarioName)
			if result != test.expected {
				t.Errorf("Expected '%s', got '%s'", test.expected, result)
			}
		})
	}
}

func TestGetDefaultExpectations(t *testing.T) {
	expectations := GetDefaultExpectations("test-scenario")

	if !expectations.ShouldSucceed {
		t.Error("Expected ShouldSucceed to be true")
	}

	if expectations.ExpectedIssueReduction != 1 {
		t.Errorf("Expected ExpectedIssueReduction to be 1, got %d", expectations.ExpectedIssueReduction)
	}

	if expectations.MaxAllowedNewIssues != 0 {
		t.Errorf("Expected MaxAllowedNewIssues to be 0, got %d", expectations.MaxAllowedNewIssues)
	}

	if !expectations.SemanticEquivalence {
		t.Error("Expected SemanticEquivalence to be true")
	}

	if expectations.PerformanceImprovement {
		t.Error("Expected PerformanceImprovement to be false")
	}
}

func TestGetRealisticExpectations(t *testing.T) {
	expectations := GetRealisticExpectations("test-app")

	if !expectations.ShouldSucceed {
		t.Error("Expected ShouldSucceed to be true")
	}

	if expectations.ExpectedIssueReduction != 3 {
		t.Errorf("Expected ExpectedIssueReduction to be 3, got %d", expectations.ExpectedIssueReduction)
	}

	if expectations.MaxAllowedNewIssues != 2 {
		t.Errorf("Expected MaxAllowedNewIssues to be 2, got %d", expectations.MaxAllowedNewIssues)
	}

	if expectations.SemanticEquivalence {
		t.Error("Expected SemanticEquivalence to be false for realistic scenarios")
	}

	if !expectations.PerformanceImprovement {
		t.Error("Expected PerformanceImprovement to be true for realistic scenarios")
	}

	if expectations.MinimumJobsAnalyzed != 5 {
		t.Errorf("Expected MinimumJobsAnalyzed to be 5, got %d", expectations.MinimumJobsAnalyzed)
	}
}

func TestFileExists(t *testing.T) {
	// Test with this file itself
	currentFile := filepath.Join(".", "testutil_test.go")
	if !FileExists(currentFile) {
		t.Error("Expected current test file to exist")
	}

	// Test with non-existent file
	if FileExists("non-existent-file.txt") {
		t.Error("Expected non-existent file to not exist")
	}
}

func TestJobChangeTypeConstants(t *testing.T) {
	if JobAdded != "added" {
		t.Errorf("Expected JobAdded to be 'added', got '%s'", JobAdded)
	}

	if JobRemoved != "removed" {
		t.Errorf("Expected JobRemoved to be 'removed', got '%s'", JobRemoved)
	}

	if JobUnchanged != "unchanged" {
		t.Errorf("Expected JobUnchanged to be 'unchanged', got '%s'", JobUnchanged)
	}

	if JobImproved != "improved" {
		t.Errorf("Expected JobImproved to be 'improved', got '%s'", JobImproved)
	}

	if JobRenamed != "renamed" {
		t.Errorf("Expected JobRenamed to be 'renamed', got '%s'", JobRenamed)
	}
}

func TestContainsChangePatterns(t *testing.T) {
	// This is a simplified test - in practice ContainsChange needs a proper RefactoringResult
	// We're just testing that it handles the different pattern types without panicking
	testPatterns := []string{
		"duplication",
		"consolidation",
		"template",
		"extends",
		"cache",
		"variables",
		"dependencies",
		"needs",
		"matrix",
		"include",
		"unknown-pattern",
	}

	for _, pattern := range testPatterns {
		t.Run(pattern, func(t *testing.T) {
			// Create a result with nil ActualChanges to test the nil safety
			result := &RefactoringResult{ActualChanges: nil}
			// This should handle nil gracefully and not panic
			contains := ContainsChange(result, pattern)
			if contains {
				t.Errorf("Expected no matches for pattern '%s' with nil changes", pattern)
			}
		})
	}
}
