package maintainability

import (
	"testing"

	"github.com/wonderfulspam/gitlab-smith/pkg/analyzer/types"
	"github.com/wonderfulspam/gitlab-smith/pkg/parser"
)

func TestCheckJobNaming(t *testing.T) {
	t.Run("Job name with spaces", func(t *testing.T) {
		config := &parser.GitLabConfig{
			Jobs: map[string]*parser.JobConfig{
				"build project": {
					Stage: "build",
				},
			},
		}

		issues := CheckJobNaming(config)

		if len(issues) != 1 {
			t.Errorf("Expected 1 issue, got %d", len(issues))
		}

		issue := issues[0]
		if issue.Type != types.IssueTypeMaintainability {
			t.Errorf("Expected maintainability issue, got %s", issue.Type)
		}

		if issue.Severity != types.SeverityLow {
			t.Errorf("Expected low severity, got %s", issue.Severity)
		}
	})

	t.Run("Long job name", func(t *testing.T) {
		longName := "this_is_a_very_long_job_name_that_exceeds_the_sixty_three_character_limit"
		config := &parser.GitLabConfig{
			Jobs: map[string]*parser.JobConfig{
				longName: {
					Stage: "build",
				},
			},
		}

		issues := CheckJobNaming(config)

		if len(issues) != 1 {
			t.Errorf("Expected 1 issue, got %d", len(issues))
		}

		issue := issues[0]
		if issue.Type != types.IssueTypeMaintainability {
			t.Errorf("Expected maintainability issue, got %s", issue.Type)
		}

		if issue.Severity != types.SeverityMedium {
			t.Errorf("Expected medium severity, got %s", issue.Severity)
		}
	})
}

func TestCheckScriptComplexity(t *testing.T) {
	t.Run("Complex script", func(t *testing.T) {
		script := make([]string, 15) // More than 10 lines
		for i := 0; i < 15; i++ {
			script[i] = "echo line " + string(rune(i+'0'))
		}

		config := &parser.GitLabConfig{
			Jobs: map[string]*parser.JobConfig{
				"build": {
					Stage:  "build",
					Script: script,
				},
			},
		}

		issues := CheckScriptComplexity(config)

		if len(issues) != 1 {
			t.Errorf("Expected 1 issue, got %d", len(issues))
		}

		issue := issues[0]
		if issue.Type != types.IssueTypeMaintainability {
			t.Errorf("Expected maintainability issue, got %s", issue.Type)
		}
	})

	t.Run("Hardcoded URL in script", func(t *testing.T) {
		config := &parser.GitLabConfig{
			Jobs: map[string]*parser.JobConfig{
				"deploy": {
					Stage: "deploy",
					Script: []string{
						"curl -X POST https://api.example.com/deploy",
						"echo 'Deployed'",
					},
				},
			},
		}

		issues := CheckScriptComplexity(config)

		if len(issues) != 1 {
			t.Errorf("Expected 1 issue, got %d", len(issues))
		}

		issue := issues[0]
		if issue.Type != types.IssueTypeMaintainability {
			t.Errorf("Expected maintainability issue, got %s", issue.Type)
		}

		if !contains(issue.Message, "Hardcoded URL") {
			t.Errorf("Expected message to contain 'Hardcoded URL', got '%s'", issue.Message)
		}
	})
}

func TestCheckDuplicatedCode(t *testing.T) {
	t.Run("Duplicated scripts", func(t *testing.T) {
		config := &parser.GitLabConfig{
			Jobs: map[string]*parser.JobConfig{
				"test1": {
					Stage:  "test",
					Script: []string{"npm test", "npm run coverage"},
				},
				"test2": {
					Stage:  "test",
					Script: []string{"npm test", "npm run coverage"},
				},
				"build": {
					Stage:  "build",
					Script: []string{"npm run build"},
				},
			},
		}

		issues := CheckDuplicatedCode(config)

		if len(issues) != 1 {
			t.Errorf("Expected 1 issue, got %d", len(issues))
		}

		issue := issues[0]
		if issue.Type != types.IssueTypeMaintainability {
			t.Errorf("Expected maintainability issue, got %s", issue.Type)
		}

		if !contains(issue.Message, "test1") || !contains(issue.Message, "test2") {
			t.Errorf("Expected message to contain both job names, got '%s'", issue.Message)
		}
	})
}

func TestCheckStagesDefinition(t *testing.T) {
	t.Run("No stages defined", func(t *testing.T) {
		config := &parser.GitLabConfig{
			Stages: []string{},
			Jobs:   make(map[string]*parser.JobConfig),
		}

		issues := CheckStagesDefinition(config)

		if len(issues) != 1 {
			t.Errorf("Expected 1 issue, got %d", len(issues))
		}

		issue := issues[0]
		if issue.Type != types.IssueTypeMaintainability {
			t.Errorf("Expected maintainability issue, got %s", issue.Type)
		}

		if issue.Severity != types.SeverityMedium {
			t.Errorf("Expected medium severity, got %s", issue.Severity)
		}
	})
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}