package performance

import (
	"strings"
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

func TestCheckUnnecessaryDependencies(t *testing.T) {
	t.Run("job with unnecessary dependency", func(t *testing.T) {
		config := &parser.GitLabConfig{
			Stages: []string{"build", "test", "deploy"},
			Jobs: map[string]*parser.JobConfig{
				"build-job": {Stage: "build"},
				"test-job": {
					Stage:        "test",
					Dependencies: []string{"build-job"},
				},
			},
		}

		issues := CheckUnnecessaryDependencies(config)

		if len(issues) != 1 {
			t.Errorf("Expected 1 issue, got %d", len(issues))
		}

		issue := issues[0]
		if issue.Type != types.IssueTypePerformance {
			t.Errorf("Expected performance issue, got %s", issue.Type)
		}
		if issue.JobName != "test-job" {
			t.Errorf("Expected job name 'test-job', got '%s'", issue.JobName)
		}
	})
}

func TestCheckMissingNeeds(t *testing.T) {
	t.Run("many jobs with dependencies but no needs", func(t *testing.T) {
		config := &parser.GitLabConfig{
			Jobs: map[string]*parser.JobConfig{
				"job1": {Dependencies: []string{"dep1"}},
				"job2": {Dependencies: []string{"dep1", "dep2"}},
				"job3": {Dependencies: []string{"dep3"}},
				"dep1": {},
				"dep2": {},
				"dep3": {},
			},
		}

		issues := CheckMissingNeeds(config)

		if len(issues) != 1 {
			t.Errorf("Expected 1 issue, got %d", len(issues))
		}

		issue := issues[0]
		if issue.Type != types.IssueTypePerformance {
			t.Errorf("Expected performance issue, got %s", issue.Type)
		}
		if !strings.Contains(issue.Message, "Found 3 jobs") {
			t.Errorf("Expected message to mention 3 jobs, got: %s", issue.Message)
		}
	})
}

func TestCheckWorkflowOptimization(t *testing.T) {
	t.Run("many jobs with branch-specific rules", func(t *testing.T) {
		config := &parser.GitLabConfig{
			Jobs: map[string]*parser.JobConfig{
				"job1": {
					Rules: []parser.Rule{
						{If: "$CI_COMMIT_BRANCH == 'main'"},
					},
				},
				"job2": {
					Rules: []parser.Rule{
						{If: "$CI_COMMIT_BRANCH == 'main'"},
					},
				},
			},
		}

		issues := CheckWorkflowOptimization(config)

		if len(issues) != 1 {
			t.Errorf("Expected 1 issue, got %d", len(issues))
		}

		issue := issues[0]
		if issue.Type != types.IssueTypePerformance {
			t.Errorf("Expected performance issue, got %s", issue.Type)
		}
		if !strings.Contains(issue.Message, "branch-specific rules") {
			t.Errorf("Expected message about branch-specific rules, got: %s", issue.Message)
		}
	})

	t.Run("jobs with MR-specific rules", func(t *testing.T) {
		config := &parser.GitLabConfig{
			Jobs: map[string]*parser.JobConfig{
				"job1": {
					Rules: []parser.Rule{
						{If: "$CI_MERGE_REQUEST_ID"},
					},
				},
				"job2": {
					Rules: []parser.Rule{
						{If: "$CI_MERGE_REQUEST_ID"},
					},
				},
				"job3": {},
			},
		}

		issues := CheckWorkflowOptimization(config)

		if len(issues) != 1 {
			t.Errorf("Expected 1 issue, got %d", len(issues))
		}

		issue := issues[0]
		if issue.Type != types.IssueTypePerformance {
			t.Errorf("Expected performance issue, got %s", issue.Type)
		}
		if !strings.Contains(issue.Message, "merge request specific") {
			t.Errorf("Expected message about MR specific jobs, got: %s", issue.Message)
		}
	})
}

func TestHasBranchSpecificRules(t *testing.T) {
	tests := []struct {
		name     string
		job      *parser.JobConfig
		expected bool
	}{
		{
			name: "job with branch rule",
			job: &parser.JobConfig{
				Rules: []parser.Rule{
					{If: "$CI_COMMIT_BRANCH == 'main'"},
				},
			},
			expected: true,
		},
		{
			name: "job with only main",
			job: &parser.JobConfig{
				Only: "main",
			},
			expected: true,
		},
		{
			name: "job without branch rules",
			job: &parser.JobConfig{
				Rules: []parser.Rule{
					{If: "$CI_MERGE_REQUEST_ID"},
				},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hasBranchSpecificRules(tt.job)
			if result != tt.expected {
				t.Errorf("hasBranchSpecificRules() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestHasMRSpecificRules(t *testing.T) {
	tests := []struct {
		name     string
		job      *parser.JobConfig
		expected bool
	}{
		{
			name: "job with MR rule",
			job: &parser.JobConfig{
				Rules: []parser.Rule{
					{If: "$CI_MERGE_REQUEST_ID"},
				},
			},
			expected: true,
		},
		{
			name: "job with only merge_requests",
			job: &parser.JobConfig{
				Only: "merge_requests",
			},
			expected: true,
		},
		{
			name: "job with only slice containing merge_requests",
			job: &parser.JobConfig{
				Only: []interface{}{"merge_requests"},
			},
			expected: true,
		},
		{
			name: "job without MR rules",
			job: &parser.JobConfig{
				Rules: []parser.Rule{
					{If: "$CI_COMMIT_BRANCH == 'main'"},
				},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hasMRSpecificRules(tt.job)
			if result != tt.expected {
				t.Errorf("hasMRSpecificRules() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestCanUseMatrix(t *testing.T) {
	tests := []struct {
		name     string
		jobNames []string
		jobs     map[string]*parser.JobConfig
		expected bool
	}{
		{
			name:     "too few jobs",
			jobNames: []string{"job1"},
			jobs: map[string]*parser.JobConfig{
				"job1": {Stage: "test"},
			},
			expected: false,
		},
		{
			name:     "different stages",
			jobNames: []string{"job1", "job2"},
			jobs: map[string]*parser.JobConfig{
				"job1": {Stage: "build"},
				"job2": {Stage: "test"},
			},
			expected: false,
		},
		{
			name:     "same stage different images",
			jobNames: []string{"job1", "job2"},
			jobs: map[string]*parser.JobConfig{
				"job1": {Stage: "test", Image: "node:14"},
				"job2": {Stage: "test", Image: "node:16"},
			},
			expected: true,
		},
		{
			name:     "same stage different variables",
			jobNames: []string{"job1", "job2"},
			jobs: map[string]*parser.JobConfig{
				"job1": {
					Stage: "test",
					Variables: map[string]interface{}{
						"VERSION": "14",
					},
				},
				"job2": {
					Stage: "test",
					Variables: map[string]interface{}{
						"VERSION": "16",
					},
				},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := canUseMatrix(tt.jobNames, tt.jobs)
			if result != tt.expected {
				t.Errorf("canUseMatrix() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestRegisterChecks(t *testing.T) {
	registry := &mockRegistry{
		checks: make(map[string]registeredCheck),
	}

	RegisterChecks(registry)

	expectedChecks := []string{
		"cache_usage",
		"artifact_expiration",
		"dependency_chains",
		"unnecessary_dependencies",
		"matrix_opportunities",
		"missing_needs",
		"workflow_optimization",
	}

	if len(registry.checks) != len(expectedChecks) {
		t.Errorf("Expected %d checks to be registered, got %d", len(expectedChecks), len(registry.checks))
	}

	for _, checkName := range expectedChecks {
		if check, exists := registry.checks[checkName]; !exists {
			t.Errorf("Check %s not registered", checkName)
		} else if check.issueType != types.IssueTypePerformance {
			t.Errorf("Expected performance issue type for %s, got %s", checkName, check.issueType)
		}
	}
}

// Additional test for cache configuration edge cases
func TestCheckCacheUsage_EdgeCases(t *testing.T) {
	t.Run("cache with empty paths", func(t *testing.T) {
		config := &parser.GitLabConfig{
			Jobs: map[string]*parser.JobConfig{
				"build": {
					Stage: "build",
					Cache: &parser.Cache{
						Key:   "my-key",
						Paths: []string{},
					},
				},
			},
		}

		issues := CheckCacheUsage(config)

		if len(issues) != 1 {
			t.Errorf("Expected 1 issue, got %d", len(issues))
		}

		issue := issues[0]
		if !strings.Contains(issue.Message, "Cache configured without paths") {
			t.Errorf("Expected message about empty paths, got: %s", issue.Message)
		}
	})
}

// Mock registry for testing
type registeredCheck struct {
	name      string
	issueType types.IssueType
	checkFunc types.CheckFunc
}

type mockRegistry struct {
	checks map[string]registeredCheck
}

func (r *mockRegistry) Register(name string, issueType types.IssueType, checkFunc types.CheckFunc) {
	r.checks[name] = registeredCheck{
		name:      name,
		issueType: issueType,
		checkFunc: checkFunc,
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
