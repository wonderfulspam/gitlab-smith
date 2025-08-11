package differ

import (
	"testing"

	"github.com/wonderfulspam/gitlab-smith/pkg/parser"
)

func TestImprovementPatternDetection(t *testing.T) {
	// Test default consolidation pattern
	oldConfig := &parser.GitLabConfig{
		Jobs: map[string]*parser.JobConfig{
			"build": {
				BeforeScript: []string{"npm ci"},
				Image:        "node:16",
			},
			"test": {
				BeforeScript: []string{"npm ci"},
				Image:        "node:16",
			},
		},
	}

	newConfig := &parser.GitLabConfig{
		Default: &parser.JobConfig{
			BeforeScript: []string{"npm ci"},
			Image:        "node:16",
		},
		Jobs: map[string]*parser.JobConfig{
			"build": {},
			"test":  {},
		},
	}

	result := Compare(oldConfig, newConfig)

	if len(result.ImprovementTags) == 0 {
		t.Error("Expected improvement tags to be detected")
	}

	if len(result.Improvements) == 0 {
		t.Error("Expected improvements to be detected")
	}

	// Check for consolidation tag
	hasConsolidation := false
	for _, tag := range result.ImprovementTags {
		if tag == "consolidation" {
			hasConsolidation = true
			break
		}
	}

	if !hasConsolidation {
		t.Errorf("Expected 'consolidation' tag, got: %v", result.ImprovementTags)
	}
}

func TestTemplateExtractionDetection(t *testing.T) {
	// Test template extraction pattern
	oldConfig := &parser.GitLabConfig{
		Jobs: map[string]*parser.JobConfig{
			"build": {
				Image: "docker:20.10.16",
			},
		},
	}

	newConfig := &parser.GitLabConfig{
		Jobs: map[string]*parser.JobConfig{
			".docker_base": {
				Image: "docker:20.10.16",
			},
			"build": {
				Extends: []interface{}{"docker_base"},
			},
		},
	}

	result := Compare(oldConfig, newConfig)

	// Check for template tags
	hasTemplates := false
	for _, tag := range result.ImprovementTags {
		if tag == "templates" {
			hasTemplates = true
			break
		}
	}

	if !hasTemplates {
		t.Errorf("Expected 'templates' tag, got: %v", result.ImprovementTags)
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsSubstring(s, substr)))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
