package maintainability

import (
	"strings"
	"testing"

	"github.com/wonderfulspam/gitlab-smith/pkg/analyzer/types"
	"github.com/wonderfulspam/gitlab-smith/pkg/parser"
)

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

		if !strings.Contains(issue.Message, "Hardcoded URL") {
			t.Errorf("Expected message to contain 'Hardcoded URL', got '%s'", issue.Message)
		}
	})

	t.Run("Job without script", func(t *testing.T) {
		config := &parser.GitLabConfig{
			Jobs: map[string]*parser.JobConfig{
				"trigger": {
					Stage: "deploy",
					// No script
				},
			},
		}

		issues := CheckScriptComplexity(config)

		if len(issues) != 0 {
			t.Errorf("Expected 0 issues for job without script, got %d", len(issues))
		}
	})
}

func TestCheckVerboseRules(t *testing.T) {
	t.Run("Job with many rules (>3)", func(t *testing.T) {
		config := &parser.GitLabConfig{
			Jobs: map[string]*parser.JobConfig{
				"complex_job": {
					Stage: "deploy",
					Rules: []parser.Rule{
						{If: "$CI_COMMIT_BRANCH == 'main'"},
						{If: "$CI_COMMIT_BRANCH =~ /^feature\\/.*$/"},
						{If: "$CI_COMMIT_BRANCH =~ /^hotfix\\/.*$/"},
						{If: "$CI_COMMIT_BRANCH == 'develop'"},
					},
				},
			},
		}

		issues := CheckVerboseRules(config)

		if len(issues) == 0 {
			t.Error("Expected at least 1 issue for job with >3 rules")
		}

		found := false
		for _, issue := range issues {
			if strings.Contains(issue.Message, "complex rules configuration") {
				found = true
				break
			}
		}

		if !found {
			t.Error("Expected complex rules configuration issue")
		}
	})

	t.Run("Job with simple rules", func(t *testing.T) {
		config := &parser.GitLabConfig{
			Jobs: map[string]*parser.JobConfig{
				"simple_job": {
					Stage: "test",
					Rules: []parser.Rule{
						{
							If: "$CI_COMMIT_BRANCH == 'main'",
						},
					},
				},
			},
		}

		issues := CheckVerboseRules(config)

		if len(issues) != 0 {
			t.Errorf("Expected 0 issues for simple rules, got %d", len(issues))
		}
	})

	t.Run("Job with contradictory when conditions", func(t *testing.T) {
		config := &parser.GitLabConfig{
			Jobs: map[string]*parser.JobConfig{
				"contradictory_job": {
					Stage: "deploy",
					Rules: []parser.Rule{
						{If: "$CI_COMMIT_BRANCH == 'main'", When: "always"},
						{If: "$CI_COMMIT_BRANCH == 'develop'", When: "never"},
					},
				},
			},
		}

		issues := CheckVerboseRules(config)

		found := false
		for _, issue := range issues {
			if strings.Contains(issue.Message, "contradictory when conditions") {
				found = true
				break
			}
		}

		if !found {
			t.Error("Expected contradictory when conditions issue")
		}
	})

	t.Run("Job without rules", func(t *testing.T) {
		config := &parser.GitLabConfig{
			Jobs: map[string]*parser.JobConfig{
				"no_rules_job": {
					Stage:  "build",
					Script: []string{"echo 'build'"},
				},
			},
		}

		issues := CheckVerboseRules(config)

		if len(issues) != 0 {
			t.Errorf("Expected 0 issues for job without rules, got %d", len(issues))
		}
	})
}