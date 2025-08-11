package maintainability

import (
	"strings"

	"github.com/emt/gitlab-smith/pkg/analyzer/types"
	"github.com/emt/gitlab-smith/pkg/parser"
)

// CheckRegistry interface to avoid import cycles
type CheckRegistry interface {
	Register(name string, issueType types.IssueType, checkFunc types.CheckFunc)
}

// RegisterChecks registers all maintainability-related checks
func RegisterChecks(registry CheckRegistry) {
	registry.Register("job_naming", types.IssueTypeMaintainability, CheckJobNaming)
	registry.Register("script_complexity", types.IssueTypeMaintainability, CheckScriptComplexity)
	registry.Register("duplicated_code", types.IssueTypeMaintainability, CheckDuplicatedCode)
	registry.Register("stages_definition", types.IssueTypeMaintainability, CheckStagesDefinition)
	registry.Register("duplicated_before_scripts", types.IssueTypeMaintainability, CheckDuplicatedBeforeScripts)
	registry.Register("verbose_rules", types.IssueTypeMaintainability, CheckVerboseRules)
	registry.Register("include_optimization", types.IssueTypeMaintainability, CheckIncludeOptimization)
}

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

func CheckDuplicatedCode(config *parser.GitLabConfig) []types.Issue {
	var issues []types.Issue
	scriptSets := make(map[string][]string)

	for jobName, job := range config.Jobs {
		scriptKey := strings.Join(job.Script, "\n")
		if scriptKey != "" {
			scriptSets[scriptKey] = append(scriptSets[scriptKey], jobName)
		}
	}

	for _, jobNames := range scriptSets {
		if len(jobNames) > 1 {
			issues = append(issues, types.Issue{
				Type:       types.IssueTypeMaintainability,
				Severity:   types.SeverityMedium,
				Path:       "jobs",
				Message:    "Duplicated scripts in jobs: " + strings.Join(jobNames, ", "),
				Suggestion: "Consider using extends or before_script to reduce duplication",
			})
		}
	}

	return issues
}

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

func CheckDuplicatedBeforeScripts(config *parser.GitLabConfig) []types.Issue {
	var issues []types.Issue
	beforeScriptSets := make(map[string][]string)

	for jobName, job := range config.Jobs {
		// Skip template jobs (starting with .) from duplication analysis
		if strings.HasPrefix(jobName, ".") {
			continue
		}
		if len(job.BeforeScript) > 0 {
			scriptKey := strings.Join(job.BeforeScript, "\n")
			beforeScriptSets[scriptKey] = append(beforeScriptSets[scriptKey], jobName)
		}
	}

	// Report exact duplicates
	for _, jobNames := range beforeScriptSets {
		if len(jobNames) > 1 {
			issues = append(issues, types.Issue{
				Type:       types.IssueTypeMaintainability,
				Severity:   types.SeverityHigh,
				Path:       "jobs.*.before_script",
				Message:    "Duplicate before_script blocks in jobs: " + strings.Join(jobNames, ", "),
				Suggestion: "Consider consolidating duplicated before_script into default configuration or templates",
			})
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