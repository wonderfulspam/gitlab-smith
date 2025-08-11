package performance

import (
	"fmt"
	"strings"

	"github.com/emt/gitlab-smith/pkg/analyzer/types"
	"github.com/emt/gitlab-smith/pkg/parser"
)

// CheckRegistry interface to avoid import cycles
type CheckRegistry interface {
	Register(name string, issueType types.IssueType, checkFunc types.CheckFunc)
}

// RegisterChecks registers all performance-related checks
func RegisterChecks(registry CheckRegistry) {
	registry.Register("cache_usage", types.IssueTypePerformance, CheckCacheUsage)
	registry.Register("artifact_expiration", types.IssueTypePerformance, CheckArtifactExpiration)
	registry.Register("dependency_chains", types.IssueTypePerformance, CheckDependencyChains)
	registry.Register("unnecessary_dependencies", types.IssueTypePerformance, CheckUnnecessaryDependencies)
	registry.Register("matrix_opportunities", types.IssueTypePerformance, CheckMatrixOpportunities)
	registry.Register("missing_needs", types.IssueTypePerformance, CheckMissingNeeds)
	registry.Register("workflow_optimization", types.IssueTypePerformance, CheckWorkflowOptimization)
}

func CheckCacheUsage(config *parser.GitLabConfig) []types.Issue {
	var issues []types.Issue
	jobsWithoutCache := 0
	totalJobs := len(config.Jobs)

	for jobName, job := range config.Jobs {
		if job.Cache == nil && (config.Default == nil || config.Default.Cache == nil) {
			jobsWithoutCache++
		}

		// Check for inefficient cache configuration
		if job.Cache != nil {
			if job.Cache.Key == "" {
				issues = append(issues, types.Issue{
					Type:       types.IssueTypePerformance,
					Severity:   types.SeverityMedium,
					Path:       "jobs." + jobName + ".cache.key",
					Message:    "Cache configured without key - may lead to cache conflicts",
					Suggestion: "Define a specific cache key to avoid conflicts between jobs",
					JobName:    jobName,
				})
			}

			if len(job.Cache.Paths) == 0 {
				issues = append(issues, types.Issue{
					Type:       types.IssueTypePerformance,
					Severity:   types.SeverityMedium,
					Path:       "jobs." + jobName + ".cache.paths",
					Message:    "Cache configured without paths",
					Suggestion: "Specify cache paths to improve build performance",
					JobName:    jobName,
				})
			}
		}
	}

	if totalJobs > 0 && float64(jobsWithoutCache)/float64(totalJobs) > 0.5 {
		issues = append(issues, types.Issue{
			Type:       types.IssueTypePerformance,
			Severity:   types.SeverityMedium,
			Path:       "cache",
			Message:    "More than half of jobs don't use caching",
			Suggestion: "Consider adding cache configuration to improve build performance",
		})
	}

	return issues
}

func CheckArtifactExpiration(config *parser.GitLabConfig) []types.Issue {
	var issues []types.Issue

	for jobName, job := range config.Jobs {
		if job.Artifacts != nil && job.Artifacts.ExpireIn == "" {
			issues = append(issues, types.Issue{
				Type:       types.IssueTypePerformance,
				Severity:   types.SeverityLow,
				Path:       "jobs." + jobName + ".artifacts.expire_in",
				Message:    "Artifacts configured without expiration",
				Suggestion: "Set expire_in to prevent storage bloat",
				JobName:    jobName,
			})
		}
	}

	return issues
}

func CheckDependencyChains(config *parser.GitLabConfig) []types.Issue {
	var issues []types.Issue
	graph := config.GetDependencyGraph()

	// Check for very long dependency chains
	for jobName, deps := range graph {
		if len(deps) > 5 {
			issues = append(issues, types.Issue{
				Type:       types.IssueTypePerformance,
				Severity:   types.SeverityMedium,
				Path:       "jobs." + jobName,
				Message:    fmt.Sprintf("Job has many dependencies (%d)", len(deps)),
				Suggestion: "Consider reducing dependencies or using parallel execution",
				JobName:    jobName,
			})
		}
	}

	return issues
}

func CheckUnnecessaryDependencies(config *parser.GitLabConfig) []types.Issue {
	var issues []types.Issue

	// Create stage order map
	stageOrder := make(map[string]int)
	for i, stage := range config.Stages {
		stageOrder[stage] = i
	}

	for jobName, job := range config.Jobs {
		if len(job.Dependencies) > 0 {
			currentStageOrder := stageOrder[job.Stage]
			unnecessaryDeps := 0

			for _, dep := range job.Dependencies {
				if depJob, exists := config.Jobs[dep]; exists {
					depStageOrder := stageOrder[depJob.Stage]
					// If dependency is from earlier stage, it might be unnecessary
					if depStageOrder < currentStageOrder {
						unnecessaryDeps++
					}
				}
			}

			if unnecessaryDeps > 0 {
				issues = append(issues, types.Issue{
					Type:       types.IssueTypePerformance,
					Severity:   types.SeverityLow,
					Path:       "jobs." + jobName + ".dependencies",
					Message:    "Job may have unnecessary explicit dependencies",
					Suggestion: "Consider letting GitLab auto-infer dependencies from artifacts",
					JobName:    jobName,
				})
			}
		}
	}

	return issues
}

func CheckMatrixOpportunities(config *parser.GitLabConfig) []types.Issue {
	var issues []types.Issue

	// Group jobs by stage (potential matrix candidates)
	stageGroups := make(map[string][]string)

	for jobName, job := range config.Jobs {
		// Skip templates
		if strings.HasPrefix(jobName, ".") {
			continue
		}

		stage := job.Stage
		if stage == "" {
			stage = "test" // Default stage
		}
		stageGroups[stage] = append(stageGroups[stage], jobName)
	}

	// Look for stages with multiple similar jobs
	for stage, jobNames := range stageGroups {
		if len(jobNames) >= 3 && canUseMatrix(jobNames, config.Jobs) {
			issues = append(issues, types.Issue{
				Type:       types.IssueTypePerformance,
				Severity:   types.SeverityMedium,
				Path:       "jobs",
				Message:    fmt.Sprintf("Multiple similar jobs in stage '%s' could use matrix strategy: %s", stage, strings.Join(jobNames, ", ")),
				Suggestion: "Consider consolidating similar jobs using parallel:matrix for better maintainability",
			})
		}
	}

	return issues
}

func CheckMissingNeeds(config *parser.GitLabConfig) []types.Issue {
	var issues []types.Issue

	// Check for jobs that use dependencies but could benefit from needs
	needsOpportunities := 0

	for _, job := range config.Jobs {
		if len(job.Dependencies) > 0 && job.Needs == nil {
			needsOpportunities++
		}
	}

	if needsOpportunities > 2 {
		issues = append(issues, types.Issue{
			Type:       types.IssueTypePerformance,
			Severity:   types.SeverityMedium,
			Path:       "jobs.*.dependencies",
			Message:    fmt.Sprintf("Found %d jobs using dependencies that could benefit from 'needs' for better parallelization", needsOpportunities),
			Suggestion: "Consider using 'needs' instead of 'dependencies' for more granular job control and better parallelization",
		})
	}

	return issues
}

func CheckWorkflowOptimization(config *parser.GitLabConfig) []types.Issue {
	var issues []types.Issue

	// Check if workflow is missing but jobs have different rules
	if config.Workflow == nil {
		branchSpecificJobs := 0
		mrSpecificJobs := 0

		for _, job := range config.Jobs {
			if hasBranchSpecificRules(job) {
				branchSpecificJobs++
			}
			if hasMRSpecificRules(job) {
				mrSpecificJobs++
			}
		}

		totalJobs := len(config.Jobs)
		if branchSpecificJobs > totalJobs/2 && totalJobs > 1 {
			issues = append(issues, types.Issue{
				Type:       types.IssueTypePerformance,
				Severity:   types.SeverityMedium,
				Path:       "workflow",
				Message:    fmt.Sprintf("%d out of %d jobs have branch-specific rules", branchSpecificJobs, totalJobs),
				Suggestion: "Consider using workflow: rules to control pipeline creation instead of individual job rules",
			})
		}

		if mrSpecificJobs > totalJobs/3 && totalJobs > 2 {
			issues = append(issues, types.Issue{
				Type:       types.IssueTypePerformance,
				Severity:   types.SeverityMedium,
				Path:       "workflow",
				Message:    fmt.Sprintf("%d jobs appear to be merge request specific", mrSpecificJobs),
				Suggestion: "Consider using workflow: rules to create separate main and MR pipelines",
			})
		}
	}

	return issues
}

// Helper functions

func canUseMatrix(jobNames []string, jobs map[string]*parser.JobConfig) bool {
	if len(jobNames) < 2 {
		return false
	}

	// Check if jobs have similar structure but different variables/configurations
	firstJob := jobs[jobNames[0]]
	if firstJob == nil {
		return false
	}

	// Look for patterns that indicate matrix potential
	commonStage := 0
	differentImages := 0
	differentVariables := 0

	for i := 1; i < len(jobNames); i++ {
		job := jobs[jobNames[i]]
		if job == nil {
			return false
		}

		// Jobs should have same stage
		if job.Stage != firstJob.Stage {
			return false
		}
		commonStage++

		// Different images often indicate matrix opportunity
		if job.Image != firstJob.Image && job.Image != "" && firstJob.Image != "" {
			differentImages++
		}

		// Different variables suggest matrix opportunity
		if job.Variables != nil || firstJob.Variables != nil {
			if !variablesEqual(job.Variables, firstJob.Variables) {
				differentVariables++
			}
		}
	}

	totalJobs := len(jobNames) - 1

	// Matrix is beneficial if jobs share same stage and have variations in setup
	sameStageAllJobs := commonStage == totalJobs
	hasVariations := differentImages > 0 || differentVariables > 0

	return sameStageAllJobs && hasVariations
}

func variablesEqual(vars1, vars2 map[string]interface{}) bool {
	if vars1 == nil && vars2 == nil {
		return true
	}
	if vars1 == nil || vars2 == nil {
		return false
	}
	if len(vars1) != len(vars2) {
		return false
	}

	for key, val1 := range vars1 {
		val2, exists := vars2[key]
		if !exists {
			return false
		}
		if fmt.Sprintf("%v", val1) != fmt.Sprintf("%v", val2) {
			return false
		}
	}

	return true
}

func hasBranchSpecificRules(job *parser.JobConfig) bool {
	for _, rule := range job.Rules {
		if strings.Contains(rule.If, "$CI_COMMIT_BRANCH") ||
			strings.Contains(rule.If, "main") ||
			strings.Contains(rule.If, "master") {
			return true
		}
	}

	// Check only/except for branch references
	if job.Only != nil {
		if onlyStr, ok := job.Only.(string); ok {
			if onlyStr == "main" || onlyStr == "master" {
				return true
			}
		}
	}

	return false
}

func hasMRSpecificRules(job *parser.JobConfig) bool {
	for _, rule := range job.Rules {
		if strings.Contains(rule.If, "$CI_MERGE_REQUEST_ID") ||
			strings.Contains(rule.If, "merge_request_event") {
			return true
		}
	}

	// Check only/except for MR references
	if job.Only != nil {
		if onlyStr, ok := job.Only.(string); ok {
			if onlyStr == "merge_requests" {
				return true
			}
		}
		if onlySlice, ok := job.Only.([]interface{}); ok {
			for _, item := range onlySlice {
				if str, ok := item.(string); ok && str == "merge_requests" {
					return true
				}
			}
		}
	}

	return false
}