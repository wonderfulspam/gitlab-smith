package performance

import (
	"strings"
	"testing"

	"github.com/wonderfulspam/gitlab-smith/pkg/analyzer/types"
	"github.com/wonderfulspam/gitlab-smith/pkg/parser"
)

func TestCheckCacheUsage_GlobalCache(t *testing.T) {
	tests := []struct {
		name           string
		config         *parser.GitLabConfig
		expectIssues   int
		expectMessages []string
	}{
		{
			name: "global cache covers all jobs",
			config: &parser.GitLabConfig{
				Cache: &parser.Cache{
					Key:   "global-cache",
					Paths: []string{".cache/"},
				},
				Jobs: map[string]*parser.JobConfig{
					"job1": {Stage: "test"},
					"job2": {Stage: "build"},
				},
			},
			expectIssues:   0,
			expectMessages: []string{},
		},
		{
			name: "complex global cache key",
			config: &parser.GitLabConfig{
				Cache: &parser.Cache{
					Key: map[string]interface{}{
						"files": []interface{}{"go.mod", "go.sum"},
					},
					Paths:  []string{".go/pkg/mod/"},
					Policy: "pull-push",
				},
				Jobs: map[string]*parser.JobConfig{
					"job1": {Stage: "test"},
					"job2": {Stage: "build"},
					"job3": {Stage: "deploy"},
				},
			},
			expectIssues:   0,
			expectMessages: []string{},
		},
		{
			name: "default cache covers jobs without explicit cache",
			config: &parser.GitLabConfig{
				Default: &parser.JobConfig{
					Cache: &parser.Cache{
						Key:   "default-cache",
						Paths: []string{".cache/"},
					},
				},
				Jobs: map[string]*parser.JobConfig{
					"job1": {Stage: "test"},
					"job2": {Stage: "build"},
				},
			},
			expectIssues:   0,
			expectMessages: []string{},
		},
		{
			name: "no cache configuration anywhere",
			config: &parser.GitLabConfig{
				Jobs: map[string]*parser.JobConfig{
					"job1": {Stage: "test"},
					"job2": {Stage: "build"},
					"job3": {Stage: "deploy"},
				},
			},
			expectIssues:   1,
			expectMessages: []string{"More than half of jobs don't use caching"},
		},
		{
			name: "mixed cache configuration",
			config: &parser.GitLabConfig{
				Cache: &parser.Cache{
					Key:   "global-cache",
					Paths: []string{".cache/"},
				},
				Jobs: map[string]*parser.JobConfig{
					"job1": {
						Stage: "test",
						Cache: &parser.Cache{
							Key:   "job-specific-cache",
							Paths: []string{".job-cache/"},
						},
					},
					"job2": {Stage: "build"}, // Uses global cache
				},
			},
			expectIssues:   0,
			expectMessages: []string{},
		},
		{
			name: "job cache without key",
			config: &parser.GitLabConfig{
				Jobs: map[string]*parser.JobConfig{
					"job1": {
						Stage: "test",
						Cache: &parser.Cache{
							Paths: []string{".cache/"}, // No key specified
						},
					},
				},
			},
			expectIssues:   1,
			expectMessages: []string{"Cache configured without key"},
		},
		{
			name: "job cache without paths",
			config: &parser.GitLabConfig{
				Jobs: map[string]*parser.JobConfig{
					"job1": {
						Stage: "test",
						Cache: &parser.Cache{
							Key: "test-key",
							// No paths specified
						},
					},
				},
			},
			expectIssues:   1,
			expectMessages: []string{"Cache configured without paths"},
		},
		{
			name: "job cache with complex key (valid)",
			config: &parser.GitLabConfig{
				Jobs: map[string]*parser.JobConfig{
					"job1": {
						Stage: "test",
						Cache: &parser.Cache{
							Key: map[string]interface{}{
								"files": []interface{}{"package.json", "yarn.lock"},
							},
							Paths: []string{"node_modules/"},
						},
					},
				},
			},
			expectIssues:   0,
			expectMessages: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issues := CheckCacheUsage(tt.config)

			if len(issues) != tt.expectIssues {
				t.Errorf("Expected %d issues, got %d", tt.expectIssues, len(issues))
				for i, issue := range issues {
					t.Logf("Issue %d: %s", i+1, issue.Message)
				}
			}

			for _, expectedMsg := range tt.expectMessages {
				found := false
				for _, issue := range issues {
					if strings.Contains(issue.Message, expectedMsg) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected to find issue message containing '%s'", expectedMsg)
				}
			}

			// Verify all issues are performance type
			for _, issue := range issues {
				if issue.Type != types.IssueTypePerformance {
					t.Errorf("Expected all issues to be performance type, got %s", issue.Type)
				}
			}
		})
	}
}

func TestCacheKeyValidation(t *testing.T) {
	tests := []struct {
		name  string
		key   interface{}
		valid bool
	}{
		{"string key", "valid-key", true},
		{"empty string key", "", false},
		{"nil key", nil, false},
		{"complex key with files", map[string]interface{}{
			"files": []interface{}{"go.mod", "go.sum"},
		}, true},
		{"complex key with prefix", map[string]interface{}{
			"prefix": "my-prefix",
		}, true},
		{"invalid type", 123, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &parser.GitLabConfig{
				Jobs: map[string]*parser.JobConfig{
					"test_job": {
						Stage: "test",
						Cache: &parser.Cache{
							Key:   tt.key,
							Paths: []string{".cache/"},
						},
					},
				},
			}

			issues := CheckCacheUsage(config)

			hasKeyIssue := false
			for _, issue := range issues {
				if strings.Contains(issue.Message, "Cache configured without key") {
					hasKeyIssue = true
					break
				}
			}

			if tt.valid && hasKeyIssue {
				t.Errorf("Key %v should be valid but got key validation issue", tt.key)
			}
			if !tt.valid && !hasKeyIssue {
				t.Errorf("Key %v should be invalid but no key validation issue found", tt.key)
			}
		})
	}
}
