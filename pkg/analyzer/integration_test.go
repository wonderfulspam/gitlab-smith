package analyzer

import (
	"testing"

	"github.com/wonderfulspam/gitlab-smith/pkg/parser"
)

// TestIntegrationAdvancedChecks tests the advanced check functionality 
// that was in the deleted analyzer_unit_test.go file
func TestIntegrationAdvancedChecks(t *testing.T) {
	analyzer := New()

	t.Run("Complex GitLab config with advanced patterns", func(t *testing.T) {
		config := &parser.GitLabConfig{
			Stages: []string{"build", "test", "deploy"},
			Variables: map[string]interface{}{
				"NODE_VERSION": "16",
				"API_SECRET":   "secret123", // Should trigger security issue
			},
			Jobs: map[string]*parser.JobConfig{
				// Template job
				".base": {
					Image:        "node:16",
					BeforeScript: []string{"npm install", "npm run setup"},
				},
				// Multiple similar jobs that could use matrix
				"test_node14": {
					Stage: "test",
					Image: "node:14",
					Variables: map[string]interface{}{
						"NODE_VERSION": "14",
					},
					Script: []string{"npm test"},
				},
				"test_node16": {
					Stage: "test", 
					Image: "node:16",
					Variables: map[string]interface{}{
						"NODE_VERSION": "16",
					},
					Script: []string{"npm test"},
				},
				"test_node18": {
					Stage: "test",
					Image: "node:18", 
					Variables: map[string]interface{}{
						"NODE_VERSION": "18",
					},
					Script: []string{"npm test"},
				},
				// Job with complex script
				"build complex": { // Should trigger naming issue
					Stage:  "build",
					Script: make([]string, 15), // Should trigger complexity issue
					Cache: &parser.Cache{
						Paths: []string{"node_modules/"}, // Should trigger cache key issue
					},
				},
				// Job with duplicated script
				"duplicate1": {
					Stage:  "test",
					Script: []string{"echo duplicate", "npm run test"},
				},
				"duplicate2": {
					Stage:  "test", 
					Script: []string{"echo duplicate", "npm run test"}, // Same as duplicate1
				},
				// Job with high retry
				"flaky_job": {
					Stage: "test",
					Retry: &parser.Retry{
						Max: 5, // Should trigger retry issue
					},
				},
				// Job with artifacts but no expiration
				"build_artifacts": {
					Stage: "build",
					Artifacts: &parser.Artifacts{
						Paths: []string{"dist/"},
						// Missing ExpireIn - should trigger issue
					},
				},
			},
		}

		result := analyzer.Analyze(config)

		// Verify we get a comprehensive analysis
		if result.TotalIssues == 0 {
			t.Error("Expected issues to be found in complex config")
		}

		// Verify we have different types of issues
		if result.Summary.Performance == 0 {
			t.Error("Expected performance issues")
		}

		if result.Summary.Security == 0 {
			t.Error("Expected security issues")
		}

		if result.Summary.Maintainability == 0 {
			t.Error("Expected maintainability issues")
		}

		if result.Summary.Reliability == 0 {
			t.Error("Expected reliability issues")
		}

		// Check for specific advanced patterns
		foundMatrixOpportunity := false
		foundCacheIssue := false
		foundComplexityIssue := false
		foundDuplicationIssue := false

		for _, issue := range result.Issues {
			switch {
			case contains(issue.Message, "matrix"):
				foundMatrixOpportunity = true
			case contains(issue.Message, "cache") && contains(issue.Message, "key"):
				foundCacheIssue = true
			case contains(issue.Message, "complex") && contains(issue.Message, "script"):
				foundComplexityIssue = true
			case contains(issue.Message, "Duplicated scripts"):
				foundDuplicationIssue = true
			}
		}

		if !foundMatrixOpportunity {
			t.Error("Expected to find matrix opportunity issue")
		}

		if !foundCacheIssue {
			t.Error("Expected to find cache key issue")
		}

		if !foundComplexityIssue {
			t.Error("Expected to find script complexity issue")
		}

		if !foundDuplicationIssue {
			t.Error("Expected to find script duplication issue")
		}
	})

	t.Run("Template inheritance patterns", func(t *testing.T) {
		config := &parser.GitLabConfig{
			Jobs: map[string]*parser.JobConfig{
				".base": {
					Image:        "alpine",
					BeforeScript: []string{"echo base"},
				},
				".extended": {
					Extends:      ".base",
					BeforeScript: []string{"echo base", "echo extended"}, // Redundant "echo base"
				},
				"job1": {
					Extends: ".extended",
					Stage:   "test",
				},
			},
		}

		result := analyzer.Analyze(config)

		// Should find some maintainability issues related to templates
		maintainabilityIssues := result.FilterByType(IssueTypeMaintainability)
		if len(maintainabilityIssues) == 0 {
			t.Error("Expected maintainability issues for template patterns")
		}
	})
}

func TestAnalyzerFiltering(t *testing.T) {
	analyzer := New()

	config := &parser.GitLabConfig{
		Variables: map[string]interface{}{
			"API_PASSWORD": "secret123", // Security issue
		},
		Jobs: map[string]*parser.JobConfig{
			"build project": { // Maintainability issue (naming)
				Stage: "build",
				Cache: &parser.Cache{
					Paths: []string{"node_modules/"}, // Performance issue (missing key)
				},
			},
		},
	}

	t.Run("Filter by performance only", func(t *testing.T) {
		result := analyzer.AnalyzeWithFilter(config, IssueTypePerformance)

		for _, issue := range result.Issues {
			if issue.Type != IssueTypePerformance {
				t.Errorf("Found non-performance issue when filtering by performance: %s", issue.Type)
			}
		}

		if len(result.Issues) == 0 {
			t.Error("Expected to find performance issues")
		}
	})

	t.Run("Filter by security only", func(t *testing.T) {
		result := analyzer.AnalyzeWithFilter(config, IssueTypeSecurity)

		for _, issue := range result.Issues {
			if issue.Type != IssueTypeSecurity {
				t.Errorf("Found non-security issue when filtering by security: %s", issue.Type)
			}
		}

		if len(result.Issues) == 0 {
			t.Error("Expected to find security issues")
		}
	})
}

func TestConfigurationManagement(t *testing.T) {
	t.Run("Custom config with disabled checks", func(t *testing.T) {
		config := DefaultConfig()
		config.DisableCheck("job_naming")

		analyzer := NewWithConfig(config)

		// Test with a config that would trigger job_naming issues
		gitlabConfig := &parser.GitLabConfig{
			Jobs: map[string]*parser.JobConfig{
				"build project": { // Job name with spaces should trigger issue if enabled
					Stage: "build",
				},
			},
		}

		result := analyzer.Analyze(gitlabConfig)

		// Should not contain job_naming issues
		for _, issue := range result.Issues {
			if contains(issue.Message, "spaces") {
				t.Error("Expected job_naming check to be disabled")
			}
		}
	})

	t.Run("List available checks", func(t *testing.T) {
		analyzer := New()
		checks := analyzer.ListChecks()

		if len(checks) == 0 {
			t.Error("Expected available checks to be listed")
		}

		// Verify we have checks from different categories
		hasPerformance := false
		hasSecurity := false
		hasMaintainability := false
		hasReliability := false

		for _, check := range checks {
			switch check.Type {
			case IssueTypePerformance:
				hasPerformance = true
			case IssueTypeSecurity:
				hasSecurity = true
			case IssueTypeMaintainability:
				hasMaintainability = true
			case IssueTypeReliability:
				hasReliability = true
			}
		}

		if !hasPerformance || !hasSecurity || !hasMaintainability || !hasReliability {
			t.Error("Expected checks from all categories to be available")
		}
	})
}

// Helper function
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}