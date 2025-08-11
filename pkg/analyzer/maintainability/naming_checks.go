package maintainability

import (
	"strings"

	"github.com/wonderfulspam/gitlab-smith/pkg/analyzer/types"
	"github.com/wonderfulspam/gitlab-smith/pkg/parser"
)

func CheckJobNaming(config *parser.GitLabConfig) []types.Issue {
	var issues []types.Issue

	for jobName := range config.Jobs {
		if strings.Contains(jobName, " ") {
			issues = append(issues, types.Issue{
				Type:       types.IssueTypeMaintainability,
				Severity:   types.SeverityLow,
				Path:       "jobs." + jobName,
				Message:    "Job name contains spaces: " + jobName,
				Suggestion: "Use underscores or hyphens instead of spaces in job names",
				JobName:    jobName,
			})
		}

		if len(jobName) > 63 {
			issues = append(issues, types.Issue{
				Type:       types.IssueTypeMaintainability,
				Severity:   types.SeverityMedium,
				Path:       "jobs." + jobName,
				Message:    "Job name is too long (>63 characters): " + jobName,
				Suggestion: "Shorten job name to improve readability and avoid potential issues",
				JobName:    jobName,
			})
		}
	}

	return issues
}
