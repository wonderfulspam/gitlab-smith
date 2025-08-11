package security

import (
	"strings"

	"github.com/wonderfulspam/gitlab-smith/pkg/analyzer/types"
	"github.com/wonderfulspam/gitlab-smith/pkg/parser"
)

// CheckRegistry interface to avoid import cycles
type CheckRegistry interface {
	Register(name string, issueType types.IssueType, checkFunc types.CheckFunc)
}

// RegisterChecks registers all security-related checks
func RegisterChecks(registry CheckRegistry) {
	registry.Register("image_tags", types.IssueTypeSecurity, CheckImageTags)
	registry.Register("environment_variables", types.IssueTypeSecurity, CheckEnvironmentVariables)
}

func CheckImageTags(config *parser.GitLabConfig) []types.Issue {
	var issues []types.Issue

	checkImage := func(image, path, jobName string) {
		if image != "" && !strings.Contains(image, ":") {
			issues = append(issues, types.Issue{
				Type:       types.IssueTypeSecurity,
				Severity:   types.SeverityMedium,
				Path:       path,
				Message:    "Docker image without explicit tag: " + image,
				Suggestion: "Use specific tags instead of 'latest' for reproducible builds",
				JobName:    jobName,
			})
		} else if strings.HasSuffix(image, ":latest") {
			issues = append(issues, types.Issue{
				Type:       types.IssueTypeSecurity,
				Severity:   types.SeverityLow,
				Path:       path,
				Message:    "Using 'latest' tag: " + image,
				Suggestion: "Pin to specific version for reproducible builds",
				JobName:    jobName,
			})
		}
	}

	// Check default image
	if config.Default != nil {
		checkImage(config.Default.Image, "default.image", "")
	}

	// Check job-specific images
	for jobName, job := range config.Jobs {
		checkImage(job.Image, "jobs."+jobName+".image", jobName)
	}

	return issues
}

func CheckEnvironmentVariables(config *parser.GitLabConfig) []types.Issue {
	var issues []types.Issue

	// Check for potential security issues in variable names
	checkVars := func(vars map[string]interface{}, path string) {
		for varName := range vars {
			if strings.Contains(strings.ToLower(varName), "password") ||
				strings.Contains(strings.ToLower(varName), "secret") ||
				strings.Contains(strings.ToLower(varName), "token") {
				issues = append(issues, types.Issue{
					Type:       types.IssueTypeSecurity,
					Severity:   types.SeverityHigh,
					Path:       path + "." + varName,
					Message:    "Potential secret in variable name: " + varName,
					Suggestion: "Use protected variables or external secret management",
				})
			}
		}
	}

	if config.Variables != nil {
		checkVars(config.Variables, "variables")
	}

	for jobName, job := range config.Jobs {
		if job.Variables != nil {
			checkVars(job.Variables, "jobs."+jobName+".variables")
		}
	}

	return issues
}