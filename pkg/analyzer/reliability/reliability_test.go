package reliability

import (
	"strings"
	"testing"

	"github.com/wonderfulspam/gitlab-smith/pkg/analyzer/types"
	"github.com/wonderfulspam/gitlab-smith/pkg/parser"
)

func TestCheckRetryConfiguration(t *testing.T) {
	tests := []struct {
		name     string
		config   *parser.GitLabConfig
		expected int
	}{
		{
			name: "no jobs",
			config: &parser.GitLabConfig{
				Jobs: make(map[string]*parser.JobConfig),
			},
			expected: 0,
		},
		{
			name: "job with low retry count",
			config: &parser.GitLabConfig{
				Jobs: map[string]*parser.JobConfig{
					"test": {
						Retry: &parser.Retry{Max: 2},
					},
				},
			},
			expected: 0,
		},
		{
			name: "job with high retry count",
			config: &parser.GitLabConfig{
				Jobs: map[string]*parser.JobConfig{
					"test": {
						Retry: &parser.Retry{Max: 5},
					},
				},
			},
			expected: 1,
		},
		{
			name: "multiple jobs with mixed retry counts",
			config: &parser.GitLabConfig{
				Jobs: map[string]*parser.JobConfig{
					"test1": {
						Retry: &parser.Retry{Max: 2},
					},
					"test2": {
						Retry: &parser.Retry{Max: 4},
					},
					"test3": {
						Retry: &parser.Retry{Max: 6},
					},
				},
			},
			expected: 2,
		},
		{
			name: "job without retry configuration",
			config: &parser.GitLabConfig{
				Jobs: map[string]*parser.JobConfig{
					"test": {},
				},
			},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issues := CheckRetryConfiguration(tt.config)
			if len(issues) != tt.expected {
				t.Errorf("Expected %d issues, got %d", tt.expected, len(issues))
			}

			// Check issue properties for non-zero expected results
			if tt.expected > 0 && len(issues) > 0 {
				issue := issues[0]
				if issue.Type != types.IssueTypeReliability {
					t.Errorf("Expected issue type reliability, got %s", issue.Type)
				}
				if issue.Severity != types.SeverityLow {
					t.Errorf("Expected severity low, got %s", issue.Severity)
				}
				if issue.Message != "High retry count may mask underlying issues" {
					t.Errorf("Unexpected message: %s", issue.Message)
				}
			}
		})
	}
}

func TestCheckMissingStages(t *testing.T) {
	tests := []struct {
		name     string
		config   *parser.GitLabConfig
		expected int
	}{
		{
			name: "no jobs or stages",
			config: &parser.GitLabConfig{
				Jobs:   make(map[string]*parser.JobConfig),
				Stages: []string{},
			},
			expected: 0,
		},
		{
			name: "jobs with defined stages",
			config: &parser.GitLabConfig{
				Stages: []string{"build", "test", "deploy"},
				Jobs: map[string]*parser.JobConfig{
					"build-job": {Stage: "build"},
					"test-job":  {Stage: "test"},
				},
			},
			expected: 0,
		},
		{
			name: "job with undefined stage",
			config: &parser.GitLabConfig{
				Stages: []string{"build", "test"},
				Jobs: map[string]*parser.JobConfig{
					"build-job":  {Stage: "build"},
					"deploy-job": {Stage: "deploy"},
				},
			},
			expected: 1,
		},
		{
			name: "multiple jobs with undefined stages",
			config: &parser.GitLabConfig{
				Stages: []string{"build"},
				Jobs: map[string]*parser.JobConfig{
					"build-job":  {Stage: "build"},
					"test-job":   {Stage: "test"},
					"deploy-job": {Stage: "deploy"},
				},
			},
			expected: 2,
		},
		{
			name: "job with empty stage",
			config: &parser.GitLabConfig{
				Stages: []string{"build", "test"},
				Jobs: map[string]*parser.JobConfig{
					"build-job": {Stage: "build"},
					"test-job":  {Stage: ""},
				},
			},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issues := CheckMissingStages(tt.config)
			if len(issues) != tt.expected {
				t.Errorf("Expected %d issues, got %d", tt.expected, len(issues))
			}

			// Check issue properties for non-zero expected results
			if tt.expected > 0 && len(issues) > 0 {
				issue := issues[0]
				if issue.Type != types.IssueTypeReliability {
					t.Errorf("Expected issue type reliability, got %s", issue.Type)
				}
				if issue.Severity != types.SeverityHigh {
					t.Errorf("Expected severity high, got %s", issue.Severity)
				}
				if !strings.Contains(issue.Message, "Job references undefined stage") {
					t.Errorf("Unexpected message: %s", issue.Message)
				}
			}
		})
	}
}

func TestRegisterChecks(t *testing.T) {
	// Create a mock registry to test registration
	registry := &mockRegistry{
		checks: make(map[string]registeredCheck),
	}

	RegisterChecks(registry)

	// Check that both checks were registered
	if len(registry.checks) != 2 {
		t.Errorf("Expected 2 checks to be registered, got %d", len(registry.checks))
	}

	// Check specific registrations
	if check, exists := registry.checks["retry_configuration"]; !exists {
		t.Error("retry_configuration check not registered")
	} else if check.issueType != types.IssueTypeReliability {
		t.Errorf("Expected reliability issue type for retry_configuration, got %s", check.issueType)
	}

	if check, exists := registry.checks["missing_stages"]; !exists {
		t.Error("missing_stages check not registered")
	} else if check.issueType != types.IssueTypeReliability {
		t.Errorf("Expected reliability issue type for missing_stages, got %s", check.issueType)
	}
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
