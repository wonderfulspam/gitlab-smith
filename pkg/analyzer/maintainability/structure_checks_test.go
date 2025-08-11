package maintainability

import (
	"strings"
	"testing"

	"github.com/wonderfulspam/gitlab-smith/pkg/analyzer/types"
	"github.com/wonderfulspam/gitlab-smith/pkg/parser"
)

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

		if !strings.Contains(issue.Message, "No stages defined") {
			t.Errorf("Expected no stages message, got: %s", issue.Message)
		}

		if issue.Path != "stages" {
			t.Errorf("Expected path 'stages', got: %s", issue.Path)
		}
	})

	t.Run("Stages properly defined", func(t *testing.T) {
		config := &parser.GitLabConfig{
			Stages: []string{"build", "test", "deploy"},
			Jobs:   make(map[string]*parser.JobConfig),
		}

		issues := CheckStagesDefinition(config)

		if len(issues) != 0 {
			t.Errorf("Expected 0 issues for defined stages, got %d", len(issues))
		}
	})

	t.Run("Single stage defined", func(t *testing.T) {
		config := &parser.GitLabConfig{
			Stages: []string{"build"},
			Jobs:   make(map[string]*parser.JobConfig),
		}

		issues := CheckStagesDefinition(config)

		if len(issues) != 0 {
			t.Errorf("Expected 0 issues for single stage, got %d", len(issues))
		}
	})

	t.Run("Nil stages", func(t *testing.T) {
		config := &parser.GitLabConfig{
			Stages: nil,
			Jobs:   make(map[string]*parser.JobConfig),
		}

		issues := CheckStagesDefinition(config)

		if len(issues) != 1 {
			t.Errorf("Expected 1 issue for nil stages, got %d", len(issues))
		}
	})
}

func TestCheckIncludeOptimization(t *testing.T) {
	t.Run("Many includes", func(t *testing.T) {
		config := &parser.GitLabConfig{
			Include: []parser.Include{
				{Local: "ci/build.yml"},
				{Local: "ci/test.yml"},
				{Local: "ci/deploy.yml"},
				{Local: "ci/security.yml"},
				{Local: "ci/lint.yml"},
				{Local: "ci/docs.yml"},
			},
		}

		issues := CheckIncludeOptimization(config)

		if len(issues) != 2 {
			t.Errorf("Expected 2 issues (many includes + local consolidation), got %d", len(issues))
		}

		hasFragmentedConfig := false
		hasLocalConsolidation := false
		for _, issue := range issues {
			if strings.Contains(issue.Message, "fragmented configuration") {
				hasFragmentedConfig = true
				if issue.Severity != types.SeverityMedium {
					t.Errorf("Expected medium severity for fragmented config, got %s", issue.Severity)
				}
			}
			if strings.Contains(issue.Message, "local includes could be consolidated") {
				hasLocalConsolidation = true
				if issue.Severity != types.SeverityLow {
					t.Errorf("Expected low severity for local consolidation, got %s", issue.Severity)
				}
			}
		}

		if !hasFragmentedConfig {
			t.Error("Expected fragmented configuration issue")
		}
		if !hasLocalConsolidation {
			t.Error("Expected local consolidation issue")
		}
	})

	t.Run("Normal include count", func(t *testing.T) {
		config := &parser.GitLabConfig{
			Include: []parser.Include{
				{Local: "ci/build.yml"},
				{Template: "Security/SAST.gitlab-ci.yml"},
				{Remote: "https://example.com/ci.yml"},
			},
		}

		issues := CheckIncludeOptimization(config)

		if len(issues) != 0 {
			t.Errorf("Expected 0 issues for normal include count, got %d", len(issues))
		}
	})

	t.Run("Exactly 5 includes (boundary)", func(t *testing.T) {
		config := &parser.GitLabConfig{
			Include: []parser.Include{
				{Local: "ci/build.yml"},
				{Local: "ci/test.yml"},
				{Local: "ci/deploy.yml"},
				{Template: "Security/SAST.gitlab-ci.yml"},
				{Remote: "https://example.com/ci.yml"},
			},
		}

		issues := CheckIncludeOptimization(config)

		if len(issues) != 0 {
			t.Errorf("Expected 0 issues for exactly 5 includes, got %d", len(issues))
		}
	})

	t.Run("Exactly 6 includes (boundary)", func(t *testing.T) {
		config := &parser.GitLabConfig{
			Include: []parser.Include{
				{Local: "ci/build.yml"},
				{Local: "ci/test.yml"},
				{Local: "ci/deploy.yml"},
				{Local: "ci/lint.yml"},
				{Template: "Security/SAST.gitlab-ci.yml"},
				{Remote: "https://example.com/ci.yml"},
			},
		}

		issues := CheckIncludeOptimization(config)

		if len(issues) != 2 {
			t.Errorf("Expected 2 issues for 6 includes, got %d", len(issues))
		}
	})

	t.Run("Exactly 3 local includes (boundary)", func(t *testing.T) {
		config := &parser.GitLabConfig{
			Include: []parser.Include{
				{Local: "ci/build.yml"},
				{Local: "ci/test.yml"},
				{Local: "ci/deploy.yml"},
			},
		}

		issues := CheckIncludeOptimization(config)

		if len(issues) != 0 {
			t.Errorf("Expected 0 issues for exactly 3 local includes, got %d", len(issues))
		}
	})

	t.Run("Exactly 4 local includes (boundary)", func(t *testing.T) {
		config := &parser.GitLabConfig{
			Include: []parser.Include{
				{Local: "ci/build.yml"},
				{Local: "ci/test.yml"},
				{Local: "ci/deploy.yml"},
				{Local: "ci/lint.yml"},
			},
		}

		issues := CheckIncludeOptimization(config)

		if len(issues) != 1 {
			t.Errorf("Expected 1 issue for 4 local includes, got %d", len(issues))
		}

		if !strings.Contains(issues[0].Message, "local includes could be consolidated") {
			t.Error("Expected local consolidation message")
		}
	})

	t.Run("Mixed include types", func(t *testing.T) {
		config := &parser.GitLabConfig{
			Include: []parser.Include{
				{Local: "ci/build.yml"},
				{Project: "group/shared", File: []string{"templates/test.yml"}},
				{Template: "Security/SAST.gitlab-ci.yml"},
				{Remote: "https://example.com/ci.yml"},
			},
		}

		issues := CheckIncludeOptimization(config)

		if len(issues) != 0 {
			t.Errorf("Expected 0 issues for mixed include types, got %d", len(issues))
		}
	})

	t.Run("No includes", func(t *testing.T) {
		config := &parser.GitLabConfig{
			Include: []parser.Include{},
		}

		issues := CheckIncludeOptimization(config)

		if len(issues) != 0 {
			t.Errorf("Expected 0 issues for no includes, got %d", len(issues))
		}
	})

	t.Run("Nil includes", func(t *testing.T) {
		config := &parser.GitLabConfig{
			Include: nil,
		}

		issues := CheckIncludeOptimization(config)

		if len(issues) != 0 {
			t.Errorf("Expected 0 issues for nil includes, got %d", len(issues))
		}
	})
}
