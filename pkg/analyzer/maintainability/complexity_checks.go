package maintainability

import (
	"strings"

	"github.com/wonderfulspam/gitlab-smith/pkg/analyzer/types"
	"github.com/wonderfulspam/gitlab-smith/pkg/parser"
)

func CheckScriptComplexity(config *parser.GitLabConfig) []types.Issue {
	var issues []types.Issue

	for jobName, job := range config.Jobs {
		scriptLines := len(job.Script)
		if scriptLines > 10 {
			issues = append(issues, types.Issue{
				Type:       types.IssueTypeMaintainability,
				Severity:   types.SeverityMedium,
				Path:       "jobs." + jobName + ".script",
				Message:    "Job script is complex (>10 lines)",
				Suggestion: "Consider breaking into smaller jobs or using external scripts",
				JobName:    jobName,
			})
		}

		// Check for hardcoded values in scripts
		for _, line := range job.Script {
			if strings.Contains(line, "http://") || strings.Contains(line, "https://") {
				issues = append(issues, types.Issue{
					Type:       types.IssueTypeMaintainability,
					Severity:   types.SeverityLow,
					Path:       "jobs." + jobName + ".script",
					Message:    "Hardcoded URL in script",
					Suggestion: "Consider using variables for URLs",
					JobName:    jobName,
				})
				break
			}
		}
	}

	return issues
}

func CheckVerboseRules(config *parser.GitLabConfig) []types.Issue {
	var issues []types.Issue

	for jobName, job := range config.Jobs {
		if len(job.Rules) > 3 {
			issues = append(issues, types.Issue{
				Type:       types.IssueTypeMaintainability,
				Severity:   types.SeverityMedium,
				Path:       "jobs." + jobName + ".rules",
				Message:    "Job has complex rules configuration (>3 rules)",
				Suggestion: "Consider simplifying rules or using workflow rules",
				JobName:    jobName,
			})
		}

		// Check for redundant rules patterns
		if len(job.Rules) > 1 {
			// Look for complementary if/when patterns that could be simplified
			hasAlways := false
			hasNever := false

			for _, rule := range job.Rules {
				if rule.When == "always" {
					hasAlways = true
				}
				if rule.When == "never" {
					hasNever = true
				}
			}

			if hasAlways && hasNever {
				issues = append(issues, types.Issue{
					Type:       types.IssueTypeMaintainability,
					Severity:   types.SeverityLow,
					Path:       "jobs." + jobName + ".rules",
					Message:    "Rules contain contradictory when conditions",
					Suggestion: "Simplify rules by consolidating conditions",
					JobName:    jobName,
				})
			}
		}
	}

	return issues
}
