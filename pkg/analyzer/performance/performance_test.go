package performance

import (
	"testing"

	"github.com/wonderfulspam/gitlab-smith/pkg/analyzer/types"
	"github.com/wonderfulspam/gitlab-smith/pkg/parser"
)

func TestCheckCacheUsage(t *testing.T) {
	t.Run("No cache configured", func(t *testing.T) {
		config := &parser.GitLabConfig{
			Jobs: map[string]*parser.JobConfig{
				"build": {Stage: "build"},
				"test":  {Stage: "test"},
			},
		}

		issues := CheckCacheUsage(config)

		if len(issues) != 1 {
			t.Errorf("Expected 1 issue, got %d", len(issues))
		}

		issue := issues[0]
		if issue.Type != types.IssueTypePerformance {
			t.Errorf("Expected performance issue, got %s", issue.Type)
		}
	})

	t.Run("Cache without key", func(t *testing.T) {
		config := &parser.GitLabConfig{
			Jobs: map[string]*parser.JobConfig{
				"build": {
					Stage: "build",
					Cache: &parser.Cache{
						Paths: []string{"node_modules/"},
					},
				},
			},
		}

		issues := CheckCacheUsage(config)

		if len(issues) != 1 {
			t.Errorf("Expected 1 issue, got %d", len(issues))
		}

		issue := issues[0]
		if issue.Type != types.IssueTypePerformance {
			t.Errorf("Expected performance issue, got %s", issue.Type)
		}

		if issue.Path != "jobs.build.cache.key" {
			t.Errorf("Expected path 'jobs.build.cache.key', got '%s'", issue.Path)
		}
	})
}

func TestCheckArtifactExpiration(t *testing.T) {
	t.Run("Artifacts without expiration", func(t *testing.T) {
		config := &parser.GitLabConfig{
			Jobs: map[string]*parser.JobConfig{
				"build": {
					Stage: "build",
					Artifacts: &parser.Artifacts{
						Paths: []string{"dist/"},
					},
				},
			},
		}

		issues := CheckArtifactExpiration(config)

		if len(issues) != 1 {
			t.Errorf("Expected 1 issue, got %d", len(issues))
		}

		issue := issues[0]
		if issue.Type != types.IssueTypePerformance {
			t.Errorf("Expected performance issue, got %s", issue.Type)
		}

		if issue.Severity != types.SeverityLow {
			t.Errorf("Expected low severity, got %s", issue.Severity)
		}
	})
}

func TestCheckDependencyChains(t *testing.T) {
	t.Run("Long dependency chain", func(t *testing.T) {
		// Create a config with a job that has many dependencies
		config := &parser.GitLabConfig{
			Jobs: map[string]*parser.JobConfig{
				"job_with_many_deps": {
					Dependencies: []string{"dep1", "dep2", "dep3", "dep4", "dep5", "dep6"},
				},
				"normal_job": {
					Dependencies: []string{"dep1", "dep2"},
				},
				"dep1": {},
				"dep2": {},
				"dep3": {},
				"dep4": {},
				"dep5": {},
				"dep6": {},
			},
		}

		issues := CheckDependencyChains(config)

		if len(issues) != 1 {
			t.Errorf("Expected 1 issue, got %d", len(issues))
		}

		issue := issues[0]
		if issue.Type != types.IssueTypePerformance {
			t.Errorf("Expected performance issue, got %s", issue.Type)
		}

		if issue.JobName != "job_with_many_deps" {
			t.Errorf("Expected job name 'job_with_many_deps', got '%s'", issue.JobName)
		}
	})
}

func TestCheckMatrixOpportunities(t *testing.T) {
	t.Run("Similar jobs that could use matrix", func(t *testing.T) {
		config := &parser.GitLabConfig{
			Jobs: map[string]*parser.JobConfig{
				"test_node14": {
					Stage: "test",
					Image: "node:14",
					Variables: map[string]interface{}{
						"NODE_VERSION": "14",
					},
				},
				"test_node16": {
					Stage: "test",
					Image: "node:16",
					Variables: map[string]interface{}{
						"NODE_VERSION": "16",
					},
				},
				"test_node18": {
					Stage: "test",
					Image: "node:18",
					Variables: map[string]interface{}{
						"NODE_VERSION": "18",
					},
				},
			},
		}

		issues := CheckMatrixOpportunities(config)

		// Should detect matrix opportunity for the 3 similar node jobs
		if len(issues) != 1 {
			t.Errorf("Expected 1 issue, got %d", len(issues))
		}

		issue := issues[0]
		if issue.Type != types.IssueTypePerformance {
			t.Errorf("Expected performance issue, got %s", issue.Type)
		}

		if !contains(issue.Message, "matrix") {
			t.Errorf("Expected message to contain 'matrix', got '%s'", issue.Message)
		}
	})
}

func TestVariablesEqual(t *testing.T) {
	tests := []struct {
		name     string
		vars1    map[string]interface{}
		vars2    map[string]interface{}
		expected bool
	}{
		{
			name:     "both nil",
			vars1:    nil,
			vars2:    nil,
			expected: true,
		},
		{
			name:     "one nil",
			vars1:    map[string]interface{}{"a": "1"},
			vars2:    nil,
			expected: false,
		},
		{
			name:     "same values",
			vars1:    map[string]interface{}{"a": "1", "b": "2"},
			vars2:    map[string]interface{}{"a": "1", "b": "2"},
			expected: true,
		},
		{
			name:     "different values",
			vars1:    map[string]interface{}{"a": "1", "b": "2"},
			vars2:    map[string]interface{}{"a": "1", "b": "3"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := variablesEqual(tt.vars1, tt.vars2)
			if result != tt.expected {
				t.Errorf("variablesEqual() = %v, expected %v", result, tt.expected)
			}
		})
	}
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