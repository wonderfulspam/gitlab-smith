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

		if !strings.Contains(issue.Message, "contains spaces") {
			t.Errorf("Expected space message, got: %s", issue.Message)
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

		if !strings.Contains(issue.Message, "too long") {
			t.Errorf("Expected length message, got: %s", issue.Message)
		}
	})

	t.Run("Valid job names", func(t *testing.T) {
		config := &parser.GitLabConfig{
			Jobs: map[string]*parser.JobConfig{
				"build": {
					Stage: "build",
				},
				"test_unit": {
					Stage: "test",
				},
				"deploy_production": {
					Stage: "deploy",
				},
				"lint-code": {
					Stage: "test",
				},
			},
		}

		issues := CheckJobNaming(config)

		if len(issues) != 0 {
			t.Errorf("Expected 0 issues for valid job names, got %d", len(issues))
		}
	})

	t.Run("Multiple naming issues", func(t *testing.T) {
		longName := "this_is_a_very_long_job_name_that_definitely_exceeds_sixty_three_characters_limit"
		config := &parser.GitLabConfig{
			Jobs: map[string]*parser.JobConfig{
				"build project": { // Contains space
					Stage: "build",
				},
				longName: { // Too long
					Stage: "test",
				},
				"deploy with spaces and very long name that exceeds character limit": { // Both issues
					Stage: "deploy",
				},
			},
		}

		issues := CheckJobNaming(config)

		if len(issues) != 4 { // 1 space + 1 length + 1 space + 1 length
			t.Errorf("Expected 4 issues (2 space + 2 length), got %d", len(issues))
		}

		spaceIssues := 0
		lengthIssues := 0
		for _, issue := range issues {
			if strings.Contains(issue.Message, "contains spaces") {
				spaceIssues++
			}
			if strings.Contains(issue.Message, "too long") {
				lengthIssues++
			}
		}

		if spaceIssues != 2 {
			t.Errorf("Expected 2 space issues, got %d", spaceIssues)
		}
		if lengthIssues != 2 {
			t.Errorf("Expected 2 length issues, got %d", lengthIssues)
		}
	})

	t.Run("Empty jobs map", func(t *testing.T) {
		config := &parser.GitLabConfig{
			Jobs: make(map[string]*parser.JobConfig),
		}

		issues := CheckJobNaming(config)

		if len(issues) != 0 {
			t.Errorf("Expected 0 issues for empty jobs map, got %d", len(issues))
		}
	})
}