package reliability

import (
	"github.com/emt/gitlab-smith/pkg/analyzer/types"
	"github.com/emt/gitlab-smith/pkg/parser"
)

// CheckRegistry interface to avoid import cycles
type CheckRegistry interface {
	Register(name string, issueType types.IssueType, checkFunc types.CheckFunc)
}

// RegisterChecks registers all reliability-related checks
func RegisterChecks(registry CheckRegistry) {
	registry.Register("retry_configuration", types.IssueTypeReliability, CheckRetryConfiguration)
	registry.Register("missing_stages", types.IssueTypeReliability, CheckMissingStages)
}

func CheckRetryConfiguration(config *parser.GitLabConfig) []types.Issue {
	var issues []types.Issue

	for jobName, job := range config.Jobs {
		if job.Retry != nil && job.Retry.Max > 3 {
			issues = append(issues, types.Issue{
				Type:       types.IssueTypeReliability,
				Severity:   types.SeverityLow,
				Path:       "jobs." + jobName + ".retry.max",
				Message:    "High retry count may mask underlying issues",
				Suggestion: "Consider investigating root cause instead of increasing retries",
				JobName:    jobName,
			})
		}
	}

	return issues
}

func CheckMissingStages(config *parser.GitLabConfig) []types.Issue {
	var issues []types.Issue

	// Check if jobs reference non-existent stages
	definedStages := make(map[string]bool)
	for _, stage := range config.Stages {
		definedStages[stage] = true
	}

	for jobName, job := range config.Jobs {
		if job.Stage != "" && !definedStages[job.Stage] {
			issues = append(issues, types.Issue{
				Type:       types.IssueTypeReliability,
				Severity:   types.SeverityHigh,
				Path:       "jobs." + jobName + ".stage",
				Message:    "Job references undefined stage: " + job.Stage,
				Suggestion: "Add '" + job.Stage + "' to the stages list or use an existing stage",
				JobName:    jobName,
			})
		}
	}

	return issues
}