package maintainability

import (
	"github.com/wonderfulspam/gitlab-smith/pkg/analyzer/types"
	"github.com/wonderfulspam/gitlab-smith/pkg/parser"
)

func CheckStagesDefinition(config *parser.GitLabConfig) []types.Issue {
	var issues []types.Issue

	if len(config.Stages) == 0 {
		issues = append(issues, types.Issue{
			Type:       types.IssueTypeMaintainability,
			Severity:   types.SeverityMedium,
			Path:       "stages",
			Message:    "No stages defined - using implicit stages",
			Suggestion: "Define explicit stages for better pipeline organization",
		})
	}

	return issues
}

func CheckIncludeOptimization(config *parser.GitLabConfig) []types.Issue {
	var issues []types.Issue

	if len(config.Include) > 5 {
		issues = append(issues, types.Issue{
			Type:       types.IssueTypeMaintainability,
			Severity:   types.SeverityMedium,
			Path:       "include",
			Message:    "Many include statements may indicate fragmented configuration",
			Suggestion: "Consider consolidating related includes into fewer, more comprehensive files",
		})
	}

	// Check for potential consolidation of local includes
	localIncludes := 0
	for _, include := range config.Include {
		if include.Local != "" {
			localIncludes++
		}
	}

	if localIncludes > 3 {
		issues = append(issues, types.Issue{
			Type:       types.IssueTypeMaintainability,
			Severity:   types.SeverityLow,
			Path:       "include",
			Message:    "Multiple local includes could be consolidated",
			Suggestion: "Consider grouping related local includes into fewer files",
		})
	}

	return issues
}
