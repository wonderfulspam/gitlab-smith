package security

import (
	"strings"
	"testing"

	"github.com/wonderfulspam/gitlab-smith/pkg/analyzer/types"
	"github.com/wonderfulspam/gitlab-smith/pkg/parser"
)

func TestCheckImageTags(t *testing.T) {
	tests := []struct {
		name     string
		config   *parser.GitLabConfig
		expected int
	}{
		{
			name: "no images configured",
			config: &parser.GitLabConfig{
				Jobs: make(map[string]*parser.JobConfig),
			},
			expected: 0,
		},
		{
			name: "images with proper tags",
			config: &parser.GitLabConfig{
				Default: &parser.JobConfig{Image: "alpine:3.14"},
				Jobs: map[string]*parser.JobConfig{
					"test": {Image: "node:16.14.0"},
				},
			},
			expected: 0,
		},
		{
			name: "image without tag",
			config: &parser.GitLabConfig{
				Default: &parser.JobConfig{Image: "alpine"},
			},
			expected: 1,
		},
		{
			name: "image with latest tag",
			config: &parser.GitLabConfig{
				Jobs: map[string]*parser.JobConfig{
					"test": {Image: "node:latest"},
				},
			},
			expected: 1,
		},
		{
			name: "mixed image configurations",
			config: &parser.GitLabConfig{
				Default: &parser.JobConfig{Image: "alpine"},
				Jobs: map[string]*parser.JobConfig{
					"test1": {Image: "node:latest"},
					"test2": {Image: "python:3.9"},
					"test3": {Image: "ubuntu"},
				},
			},
			expected: 3,
		},
		{
			name: "empty image field",
			config: &parser.GitLabConfig{
				Default: &parser.JobConfig{Image: ""},
				Jobs: map[string]*parser.JobConfig{
					"test": {Image: ""},
				},
			},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issues := CheckImageTags(tt.config)
			if len(issues) != tt.expected {
				t.Errorf("Expected %d issues, got %d", tt.expected, len(issues))
			}

			// Check issue properties for non-zero expected results
			for _, issue := range issues {
				if issue.Type != types.IssueTypeSecurity {
					t.Errorf("Expected issue type security, got %s", issue.Type)
				}
				if issue.Severity != types.SeverityMedium && issue.Severity != types.SeverityLow {
					t.Errorf("Expected severity medium or low, got %s", issue.Severity)
				}
				if !strings.Contains(issue.Message, "Docker image") && !strings.Contains(issue.Message, "Using 'latest' tag") {
					t.Errorf("Expected message to contain 'Docker image' or 'Using 'latest' tag', got: %s", issue.Message)
				}
			}
		})
	}
}

func TestCheckEnvironmentVariables(t *testing.T) {
	tests := []struct {
		name     string
		config   *parser.GitLabConfig
		expected int
	}{
		{
			name: "no variables",
			config: &parser.GitLabConfig{
				Jobs: make(map[string]*parser.JobConfig),
			},
			expected: 0,
		},
		{
			name: "safe variables",
			config: &parser.GitLabConfig{
				Variables: map[string]interface{}{
					"NODE_ENV": "production",
					"VERSION":  "1.0.0",
				},
			},
			expected: 0,
		},
		{
			name: "global sensitive variables",
			config: &parser.GitLabConfig{
				Variables: map[string]interface{}{
					"DB_PASSWORD": "secret123",
					"API_SECRET":  "mysecret",
					"AUTH_TOKEN":  "token123",
				},
			},
			expected: 3,
		},
		{
			name: "job-level sensitive variables",
			config: &parser.GitLabConfig{
				Jobs: map[string]*parser.JobConfig{
					"deploy": {
						Variables: map[string]interface{}{
							"DEPLOY_PASSWORD": "secret",
							"GITHUB_TOKEN":    "ghp_123",
						},
					},
				},
			},
			expected: 2,
		},
		{
			name: "mixed case sensitive variables",
			config: &parser.GitLabConfig{
				Variables: map[string]interface{}{
					"DATABASE_PASSWORD": "secret",
					"api_secret":        "secret",
					"Auth_Token":        "token",
				},
			},
			expected: 3,
		},
		{
			name: "variables containing sensitive substrings",
			config: &parser.GitLabConfig{
				Variables: map[string]interface{}{
					"USER_PASSWORD_POLICY": "strong",
					"SECRET_CONFIG_PATH":   "/etc/secrets",
					"TOKEN_EXPIRY":         "3600",
				},
			},
			expected: 3,
		},
		{
			name: "mixed global and job variables",
			config: &parser.GitLabConfig{
				Variables: map[string]interface{}{
					"API_KEY":  "key123",
					"NODE_ENV": "production",
				},
				Jobs: map[string]*parser.JobConfig{
					"test": {
						Variables: map[string]interface{}{
							"DB_PASSWORD": "secret",
							"VERSION":     "1.0.0",
						},
					},
				},
			},
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issues := CheckEnvironmentVariables(tt.config)
			if len(issues) != tt.expected {
				t.Errorf("Expected %d issues, got %d", tt.expected, len(issues))
			}

			// Check issue properties for non-zero expected results
			for _, issue := range issues {
				if issue.Type != types.IssueTypeSecurity {
					t.Errorf("Expected issue type security, got %s", issue.Type)
				}
				if issue.Severity != types.SeverityHigh {
					t.Errorf("Expected severity high, got %s", issue.Severity)
				}
				if !strings.Contains(issue.Message, "Potential secret") {
					t.Errorf("Expected message to contain 'Potential secret', got: %s", issue.Message)
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
	if check, exists := registry.checks["image_tags"]; !exists {
		t.Error("image_tags check not registered")
	} else if check.issueType != types.IssueTypeSecurity {
		t.Errorf("Expected security issue type for image_tags, got %s", check.issueType)
	}

	if check, exists := registry.checks["environment_variables"]; !exists {
		t.Error("environment_variables check not registered")
	} else if check.issueType != types.IssueTypeSecurity {
		t.Errorf("Expected security issue type for environment_variables, got %s", check.issueType)
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
