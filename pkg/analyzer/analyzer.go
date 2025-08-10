package analyzer

import (
	"fmt"
	"strings"

	"github.com/emt/gitlab-smith/pkg/parser"
)

type IssueType string

const (
	IssueTypePerformance     IssueType = "performance"
	IssueTypeSecurity        IssueType = "security"
	IssueTypeMaintainability IssueType = "maintainability"
	IssueTypeReliability     IssueType = "reliability"
)

type Severity string

const (
	SeverityLow    Severity = "low"
	SeverityMedium Severity = "medium"
	SeverityHigh   Severity = "high"
)

type Issue struct {
	Type       IssueType `json:"type"`
	Severity   Severity  `json:"severity"`
	Path       string    `json:"path"`
	Message    string    `json:"message"`
	Suggestion string    `json:"suggestion,omitempty"`
	JobName    string    `json:"job_name,omitempty"`
}

type AnalysisResult struct {
	Issues      []Issue `json:"issues"`
	TotalIssues int     `json:"total_issues"`
	Summary     Summary `json:"summary"`
}

type Summary struct {
	Performance     int `json:"performance"`
	Security        int `json:"security"`
	Maintainability int `json:"maintainability"`
	Reliability     int `json:"reliability"`
}

func Analyze(config *parser.GitLabConfig) *AnalysisResult {
	result := &AnalysisResult{
		Issues: []Issue{},
	}

	// Run all analysis rules
	checkMissingStages(config, result)
	checkJobNaming(config, result)
	checkCacheUsage(config, result)
	checkArtifactExpiration(config, result)
	checkImageTags(config, result)
	checkScriptComplexity(config, result)
	checkDuplicatedCode(config, result)
	checkDependencyChains(config, result)
	checkEnvironmentVariables(config, result)
	checkRetryConfiguration(config, result)

	result.TotalIssues = len(result.Issues)
	result.Summary = calculateSummary(result.Issues)

	return result
}

func checkMissingStages(config *parser.GitLabConfig, result *AnalysisResult) {
	if len(config.Stages) == 0 {
		result.Issues = append(result.Issues, Issue{
			Type:       IssueTypeMaintainability,
			Severity:   SeverityMedium,
			Path:       "stages",
			Message:    "No stages defined - using implicit stages",
			Suggestion: "Define explicit stages for better pipeline organization",
		})
	}

	// Check if jobs reference non-existent stages
	definedStages := make(map[string]bool)
	for _, stage := range config.Stages {
		definedStages[stage] = true
	}

	for jobName, job := range config.Jobs {
		if job.Stage != "" && !definedStages[job.Stage] {
			result.Issues = append(result.Issues, Issue{
				Type:       IssueTypeReliability,
				Severity:   SeverityHigh,
				Path:       "jobs." + jobName + ".stage",
				Message:    "Job references undefined stage: " + job.Stage,
				Suggestion: "Add '" + job.Stage + "' to the stages list or use an existing stage",
				JobName:    jobName,
			})
		}
	}
}

func checkJobNaming(config *parser.GitLabConfig, result *AnalysisResult) {
	for jobName := range config.Jobs {
		if strings.Contains(jobName, " ") {
			result.Issues = append(result.Issues, Issue{
				Type:       IssueTypeMaintainability,
				Severity:   SeverityLow,
				Path:       "jobs." + jobName,
				Message:    "Job name contains spaces: " + jobName,
				Suggestion: "Use underscores or hyphens instead of spaces in job names",
				JobName:    jobName,
			})
		}

		if len(jobName) > 63 {
			result.Issues = append(result.Issues, Issue{
				Type:       IssueTypeReliability,
				Severity:   SeverityMedium,
				Path:       "jobs." + jobName,
				Message:    "Job name is too long (>63 characters): " + jobName,
				Suggestion: "Shorten job name to improve readability and avoid potential issues",
				JobName:    jobName,
			})
		}
	}
}

func checkCacheUsage(config *parser.GitLabConfig, result *AnalysisResult) {
	jobsWithoutCache := 0
	totalJobs := len(config.Jobs)

	for jobName, job := range config.Jobs {
		if job.Cache == nil && (config.Default == nil || config.Default.Cache == nil) {
			jobsWithoutCache++
		}

		// Check for inefficient cache configuration
		if job.Cache != nil {
			if job.Cache.Key == "" {
				result.Issues = append(result.Issues, Issue{
					Type:       IssueTypePerformance,
					Severity:   SeverityMedium,
					Path:       "jobs." + jobName + ".cache.key",
					Message:    "Cache configured without key - may lead to cache conflicts",
					Suggestion: "Define a specific cache key to avoid conflicts between jobs",
					JobName:    jobName,
				})
			}

			if len(job.Cache.Paths) == 0 {
				result.Issues = append(result.Issues, Issue{
					Type:       IssueTypePerformance,
					Severity:   SeverityMedium,
					Path:       "jobs." + jobName + ".cache.paths",
					Message:    "Cache configured without paths",
					Suggestion: "Specify cache paths to improve build performance",
					JobName:    jobName,
				})
			}
		}
	}

	if totalJobs > 0 && float64(jobsWithoutCache)/float64(totalJobs) > 0.5 {
		result.Issues = append(result.Issues, Issue{
			Type:       IssueTypePerformance,
			Severity:   SeverityMedium,
			Path:       "cache",
			Message:    "More than half of jobs don't use caching",
			Suggestion: "Consider adding cache configuration to improve build performance",
		})
	}
}

func checkArtifactExpiration(config *parser.GitLabConfig, result *AnalysisResult) {
	for jobName, job := range config.Jobs {
		if job.Artifacts != nil && job.Artifacts.ExpireIn == "" {
			result.Issues = append(result.Issues, Issue{
				Type:       IssueTypePerformance,
				Severity:   SeverityLow,
				Path:       "jobs." + jobName + ".artifacts.expire_in",
				Message:    "Artifacts configured without expiration",
				Suggestion: "Set expire_in to prevent storage bloat",
				JobName:    jobName,
			})
		}
	}
}

func checkImageTags(config *parser.GitLabConfig, result *AnalysisResult) {
	checkImage := func(image, path, jobName string) {
		if image != "" && !strings.Contains(image, ":") {
			result.Issues = append(result.Issues, Issue{
				Type:       IssueTypeSecurity,
				Severity:   SeverityMedium,
				Path:       path,
				Message:    "Docker image without explicit tag: " + image,
				Suggestion: "Use specific tags instead of 'latest' for reproducible builds",
				JobName:    jobName,
			})
		} else if strings.HasSuffix(image, ":latest") {
			result.Issues = append(result.Issues, Issue{
				Type:       IssueTypeSecurity,
				Severity:   SeverityLow,
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
}

func checkScriptComplexity(config *parser.GitLabConfig, result *AnalysisResult) {
	for jobName, job := range config.Jobs {
		scriptLines := len(job.Script)
		if scriptLines > 10 {
			result.Issues = append(result.Issues, Issue{
				Type:       IssueTypeMaintainability,
				Severity:   SeverityMedium,
				Path:       "jobs." + jobName + ".script",
				Message:    "Job script is complex (>10 lines)",
				Suggestion: "Consider breaking into smaller jobs or using external scripts",
				JobName:    jobName,
			})
		}

		// Check for hardcoded values in scripts
		for _, line := range job.Script {
			if strings.Contains(line, "http://") || strings.Contains(line, "https://") {
				result.Issues = append(result.Issues, Issue{
					Type:       IssueTypeMaintainability,
					Severity:   SeverityLow,
					Path:       "jobs." + jobName + ".script",
					Message:    "Hardcoded URL in script",
					Suggestion: "Consider using variables for URLs",
					JobName:    jobName,
				})
				break
			}
		}
	}
}

func checkDuplicatedCode(config *parser.GitLabConfig, result *AnalysisResult) {
	scriptSets := make(map[string][]string)

	for jobName, job := range config.Jobs {
		scriptKey := strings.Join(job.Script, "\n")
		if scriptKey != "" {
			scriptSets[scriptKey] = append(scriptSets[scriptKey], jobName)
		}
	}

	for _, jobNames := range scriptSets {
		if len(jobNames) > 1 {
			result.Issues = append(result.Issues, Issue{
				Type:       IssueTypeMaintainability,
				Severity:   SeverityMedium,
				Path:       "jobs",
				Message:    "Duplicated scripts in jobs: " + strings.Join(jobNames, ", "),
				Suggestion: "Consider using extends or before_script to reduce duplication",
			})
		}
	}
}

func checkDependencyChains(config *parser.GitLabConfig, result *AnalysisResult) {
	graph := config.GetDependencyGraph()

	// Check for very long dependency chains
	for jobName, deps := range graph {
		if len(deps) > 5 {
			result.Issues = append(result.Issues, Issue{
				Type:       IssueTypePerformance,
				Severity:   SeverityMedium,
				Path:       "jobs." + jobName,
				Message:    fmt.Sprintf("Job has many dependencies (%d)", len(deps)),
				Suggestion: "Consider reducing dependencies or using parallel execution",
				JobName:    jobName,
			})
		}
	}
}

func checkEnvironmentVariables(config *parser.GitLabConfig, result *AnalysisResult) {
	// Check for potential security issues in variable names
	checkVars := func(vars map[string]interface{}, path string) {
		for varName := range vars {
			if strings.Contains(strings.ToLower(varName), "password") ||
				strings.Contains(strings.ToLower(varName), "secret") ||
				strings.Contains(strings.ToLower(varName), "token") {
				result.Issues = append(result.Issues, Issue{
					Type:       IssueTypeSecurity,
					Severity:   SeverityHigh,
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
}

func checkRetryConfiguration(config *parser.GitLabConfig, result *AnalysisResult) {
	for jobName, job := range config.Jobs {
		if job.Retry != nil && job.Retry.Max > 3 {
			result.Issues = append(result.Issues, Issue{
				Type:       IssueTypeReliability,
				Severity:   SeverityLow,
				Path:       "jobs." + jobName + ".retry.max",
				Message:    "High retry count may mask underlying issues",
				Suggestion: "Consider investigating root cause instead of increasing retries",
				JobName:    jobName,
			})
		}
	}
}

func calculateSummary(issues []Issue) Summary {
	summary := Summary{}

	for _, issue := range issues {
		switch issue.Type {
		case IssueTypePerformance:
			summary.Performance++
		case IssueTypeSecurity:
			summary.Security++
		case IssueTypeMaintainability:
			summary.Maintainability++
		case IssueTypeReliability:
			summary.Reliability++
		}
	}

	return summary
}

func (r *AnalysisResult) FilterBySeverity(severity Severity) []Issue {
	var filtered []Issue
	for _, issue := range r.Issues {
		if issue.Severity == severity {
			filtered = append(filtered, issue)
		}
	}
	return filtered
}

func (r *AnalysisResult) FilterByType(issueType IssueType) []Issue {
	var filtered []Issue
	for _, issue := range r.Issues {
		if issue.Type == issueType {
			filtered = append(filtered, issue)
		}
	}
	return filtered
}
