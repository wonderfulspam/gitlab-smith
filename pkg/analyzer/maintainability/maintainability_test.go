package maintainability

import (
	"strings"
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

func TestCheckDuplicatedCacheConfig(t *testing.T) {
	t.Run("Duplicate cache configurations", func(t *testing.T) {
		config := &parser.GitLabConfig{
			Jobs: map[string]*parser.JobConfig{
				"build": {
					Stage: "build",
					Cache: &parser.Cache{
						Key:   "$CI_COMMIT_REF_SLUG",
						Paths: []string{"node_modules/", ".npm/"},
					},
				},
				"test": {
					Stage: "test",
					Cache: &parser.Cache{
						Key:   "$CI_COMMIT_REF_SLUG",
						Paths: []string{"node_modules/", ".npm/"},
					},
				},
				"deploy": {
					Stage: "deploy",
					Cache: &parser.Cache{
						Key:   "$CI_COMMIT_REF_SLUG",
						Paths: []string{"node_modules/", ".npm/"},
					},
				},
			},
		}

		issues := CheckDuplicatedCacheConfig(config)

		if len(issues) != 1 {
			t.Errorf("Expected 1 issue for duplicate cache, got %d", len(issues))
		}

		if !strings.Contains(issues[0].Message, "Duplicate cache configuration") {
			t.Errorf("Expected duplicate cache message, got: %s", issues[0].Message)
		}
	})
}

func TestCheckDuplicatedImageConfig(t *testing.T) {
	t.Run("Duplicate image configurations", func(t *testing.T) {
		config := &parser.GitLabConfig{
			Jobs: map[string]*parser.JobConfig{
				"build": {
					Stage: "build",
					Image: "node:16",
				},
				"test": {
					Stage: "test",
					Image: "node:16",
				},
				"lint": {
					Stage: "test",
					Image: "node:16",
				},
			},
		}

		issues := CheckDuplicatedImageConfig(config)

		if len(issues) != 1 {
			t.Errorf("Expected 1 issue for duplicate image, got %d", len(issues))
		}

		if !strings.Contains(issues[0].Message, "Duplicate image configuration") {
			t.Errorf("Expected duplicate image message, got: %s", issues[0].Message)
		}
	})
}

func TestCheckDuplicatedSetup(t *testing.T) {
	t.Run("Duplicate setup commands", func(t *testing.T) {
		config := &parser.GitLabConfig{
			Jobs: map[string]*parser.JobConfig{
				"build": {
					Stage: "build",
					Script: []string{
						"npm ci --cache .npm",
						"npm run build",
					},
				},
				"test": {
					Stage: "test",
					Script: []string{
						"npm ci --cache .npm",
						"npm test",
					},
				},
			},
		}

		issues := CheckDuplicatedSetup(config)

		if len(issues) == 0 {
			t.Errorf("Expected at least 1 issue for duplicate setup, got %d", len(issues))
		}

		foundDuplicateSetup := false
		for _, issue := range issues {
			if strings.Contains(issue.Message, "Duplicate setup configuration") {
				foundDuplicateSetup = true
				break
			}
		}

		if !foundDuplicateSetup {
			t.Errorf("Expected duplicate setup configuration issue")
		}
	})
}

func TestCheckDuplicatedBeforeScriptsSimilarity(t *testing.T) {
	t.Run("Similar before_script blocks", func(t *testing.T) {
		config := &parser.GitLabConfig{
			Jobs: map[string]*parser.JobConfig{
				"build": {
					Stage: "build",
					BeforeScript: []string{
						"echo 'Starting build'",
						"apt-get update",
						"apt-get install -y git",
						"npm ci",
					},
				},
				"test": {
					Stage: "test",
					BeforeScript: []string{
						"echo 'Starting test'",
						"apt-get update",
						"apt-get install -y git",
						"npm ci",
					},
				},
			},
		}

		issues := CheckDuplicatedBeforeScripts(config)

		if len(issues) == 0 {
			t.Errorf("Expected at least 1 issue for similar before_scripts, got %d", len(issues))
		}

		foundSimilar := false
		for _, issue := range issues {
			if strings.Contains(issue.Message, "Similar before_script blocks") {
				foundSimilar = true
				break
			}
		}

		if !foundSimilar {
			t.Errorf("Expected similar before_script blocks issue")
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
