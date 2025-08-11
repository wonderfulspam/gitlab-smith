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
	checkDuplicatedBeforeScripts(config, result)
	checkDuplicatedCacheConfig(config, result)
	checkDuplicatedSetup(config, result)
	checkDuplicatedVariables(config, result)
	checkUnnecessaryDependencies(config, result)
	checkVerboseRules(config, result)
	checkDependencyChains(config, result)
	checkEnvironmentVariables(config, result)
	checkRetryConfiguration(config, result)
	checkTemplateComplexity(config, result)
	checkRedundantInheritance(config, result)
	checkMatrixOpportunities(config, result)
	checkIncludeOptimization(config, result)
	checkExternalIncludeDuplication(config, result)
	checkMissingExtends(config, result)
	checkMissingNeeds(config, result)
	checkMissingTemplates(config, result)

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

// checkDuplicatedBeforeScripts detects duplicate before_script blocks across jobs
func checkDuplicatedBeforeScripts(config *parser.GitLabConfig, result *AnalysisResult) {
	beforeScriptSets := make(map[string][]string)
	partialMatches := make(map[string][]string) // Track partial duplications
	templateInheritance := make(map[string][]string) // Track which jobs inherit from same template

	// First, identify template inheritance patterns
	for jobName, job := range config.Jobs {
		if strings.HasPrefix(jobName, ".") {
			continue // Skip template jobs themselves
		}
		if job.Extends != nil {
			extends := extractExtendsNames(job.Extends)
			for _, templateName := range extends {
				templateInheritance[templateName] = append(templateInheritance[templateName], jobName)
			}
		}
	}

	for jobName, job := range config.Jobs {
		// Skip template jobs (starting with .) from duplication analysis
		if strings.HasPrefix(jobName, ".") {
			continue
		}
		if len(job.BeforeScript) > 0 {
			scriptKey := strings.Join(job.BeforeScript, "\n")
			beforeScriptSets[scriptKey] = append(beforeScriptSets[scriptKey], jobName)
			
			// Check for substantial overlap (80%+ common commands)
			for otherJobName, otherJob := range config.Jobs {
				if jobName != otherJobName && !strings.HasPrefix(otherJobName, ".") && len(otherJob.BeforeScript) > 0 {
					overlap := calculateScriptOverlap(job.BeforeScript, otherJob.BeforeScript)
					if overlap > 0.8 && len(job.BeforeScript) > 2 {
						key := fmt.Sprintf("overlap-%s-%s", jobName, otherJobName)
						if _, exists := partialMatches[key]; !exists {
							reverseKey := fmt.Sprintf("overlap-%s-%s", otherJobName, jobName)
							if _, exists := partialMatches[reverseKey]; !exists {
								partialMatches[key] = []string{jobName, otherJobName}
							}
						}
					}
				}
			}
		}
	}

	// Report exact duplicates (but skip if they inherit from same template)
	for _, jobNames := range beforeScriptSets {
		if len(jobNames) > 1 {
			// Check if these jobs inherit from the same template
			inheritFromSameTemplate := false
			for _, templateJobs := range templateInheritance {
				if containsAllJobs(templateJobs, jobNames) {
					inheritFromSameTemplate = true
					break
				}
			}
			
			if !inheritFromSameTemplate {
				result.Issues = append(result.Issues, Issue{
					Type:       IssueTypeMaintainability,
					Severity:   SeverityHigh,
					Path:       "jobs.*.before_script",
					Message:    "Duplicate before_script blocks in jobs: " + strings.Join(jobNames, ", "),
					Suggestion: "Consider consolidating duplicated before_script into default configuration or templates",
				})
			}
		}
	}
	
	// Report substantial overlaps (but skip if they inherit from same template)
	for _, jobNames := range partialMatches {
		if len(jobNames) == 2 {
			// Check if these jobs inherit from the same template
			inheritFromSameTemplate := false
			for _, templateJobs := range templateInheritance {
				if containsAllJobs(templateJobs, jobNames) {
					inheritFromSameTemplate = true
					break
				}
			}
			
			if !inheritFromSameTemplate {
				result.Issues = append(result.Issues, Issue{
					Type:       IssueTypeMaintainability,
					Severity:   SeverityMedium,
					Path:       "jobs.*.before_script",
					Message:    "Similar before_script blocks with high overlap in jobs: " + strings.Join(jobNames, ", "),
					Suggestion: "Consider consolidating common commands into reusable template or default configuration",
				})
			}
		}
	}
}

// checkDuplicatedCacheConfig detects duplicate cache configuration across jobs
func checkDuplicatedCacheConfig(config *parser.GitLabConfig, result *AnalysisResult) {
	cacheSets := make(map[string][]string)
	
	for jobName, job := range config.Jobs {
		if job.Cache != nil {
			cacheKey := job.Cache.Key
			if cacheKey != "" {
				pathsKey := strings.Join(job.Cache.Paths, ",")
				fullCacheKey := cacheKey + "|" + pathsKey
				cacheSets[fullCacheKey] = append(cacheSets[fullCacheKey], jobName)
			}
		}
	}

	for _, jobNames := range cacheSets {
		if len(jobNames) > 1 {
			result.Issues = append(result.Issues, Issue{
				Type:       IssueTypeMaintainability,
				Severity:   SeverityMedium,
				Path:       "jobs.*.cache",
				Message:    "Duplicate cache configuration in jobs: " + strings.Join(jobNames, ", "),
				Suggestion: "Consider consolidating cache configuration into default block",
			})
		}
	}
}

// checkDuplicatedSetup detects duplicate image/services/variables setup patterns
func checkDuplicatedSetup(config *parser.GitLabConfig, result *AnalysisResult) {
	setupSets := make(map[string][]string)
	imageGroups := make(map[string][]string)
	serviceGroups := make(map[string][]string)

	for jobName, job := range config.Jobs {
		// Skip template jobs
		if strings.HasPrefix(jobName, ".") {
			continue
		}
		
		setupKey := ""
		if job.Image != "" {
			setupKey += "image:" + job.Image + "|"
			imageGroups[job.Image] = append(imageGroups[job.Image], jobName)
		}
		if len(job.Services) > 0 {
			servicesKey := strings.Join(job.Services, ",")
			setupKey += "services:" + servicesKey + "|"
			serviceGroups[servicesKey] = append(serviceGroups[servicesKey], jobName)
		}
		
		if setupKey != "" {
			setupSets[setupKey] = append(setupSets[setupKey], jobName)
		}
	}

	// Check for duplicate full setups
	for _, jobNames := range setupSets {
		if len(jobNames) > 2 { // Only flag if 3+ jobs have identical setup
			result.Issues = append(result.Issues, Issue{
				Type:       IssueTypeMaintainability,
				Severity:   SeverityHigh,
				Path:       "jobs",
				Message:    "Duplicate setup configuration in jobs: " + strings.Join(jobNames, ", "),
				Suggestion: "Consider using extends or templates to reduce setup duplication",
			})
		}
	}
	
	// Check for repeated images (potential template opportunity)
	for image, jobNames := range imageGroups {
		if len(jobNames) > 3 && image != "" {
			result.Issues = append(result.Issues, Issue{
				Type:       IssueTypeMaintainability,
				Severity:   SeverityMedium,
				Path:       "jobs.*.image",
				Message:    fmt.Sprintf("Image '%s' used in multiple jobs: %s", image, strings.Join(jobNames, ", ")),
				Suggestion: "Consider using default image or template to reduce duplication",
			})
		}
	}
	
	// Check for repeated services
	for services, jobNames := range serviceGroups {
		if len(jobNames) > 2 && services != "" {
			result.Issues = append(result.Issues, Issue{
				Type:       IssueTypeMaintainability,
				Severity:   SeverityMedium,
				Path:       "jobs.*.services",
				Message:    fmt.Sprintf("Services '%s' used in multiple jobs: %s", services, strings.Join(jobNames, ", ")),
				Suggestion: "Consider using template to share common services configuration",
			})
		}
	}
}

// checkUnnecessaryDependencies detects explicit dependencies that could be inferred
func checkUnnecessaryDependencies(config *parser.GitLabConfig, result *AnalysisResult) {
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
				result.Issues = append(result.Issues, Issue{
					Type:       IssueTypeMaintainability,
					Severity:   SeverityLow,
					Path:       "jobs." + jobName + ".dependencies",
					Message:    "Job may have unnecessary explicit dependencies",
					Suggestion: "Consider letting GitLab auto-infer dependencies from artifacts",
					JobName:    jobName,
				})
			}
		}
	}
}

// checkDuplicatedVariables detects repeated variables across jobs
func checkDuplicatedVariables(config *parser.GitLabConfig, result *AnalysisResult) {
	variableSets := make(map[string][]string) // value -> job names

	for jobName, job := range config.Jobs {
		// Skip template jobs (starting with .) from duplication analysis
		if strings.HasPrefix(jobName, ".") {
			continue
		}
		if job.Variables != nil {
			for varName, varValue := range job.Variables {
				if varValueStr, ok := varValue.(string); ok {
					key := varName + ":" + varValueStr
					variableSets[key] = append(variableSets[key], jobName)
				}
			}
		}
	}

	for varKey, jobNames := range variableSets {
		if len(jobNames) > 2 { // Flag if 3+ jobs have same variable
			result.Issues = append(result.Issues, Issue{
				Type:       IssueTypeMaintainability,
				Severity:   SeverityMedium,
				Path:       "jobs.*.variables",
				Message:    "Duplicate variable definition in jobs: " + strings.Join(jobNames, ", ") + " (variable: " + varKey + ")",
				Suggestion: "Consider moving repeated variables to global scope or template",
			})
		}
	}
}

// checkVerboseRules detects overly complex or redundant rules
func checkVerboseRules(config *parser.GitLabConfig, result *AnalysisResult) {
	for jobName, job := range config.Jobs {
		if len(job.Rules) > 3 {
			result.Issues = append(result.Issues, Issue{
				Type:       IssueTypeMaintainability,
				Severity:   SeverityMedium,
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
				result.Issues = append(result.Issues, Issue{
					Type:       IssueTypeMaintainability,
					Severity:   SeverityLow,
					Path:       "jobs." + jobName + ".rules",
					Message:    "Rules contain contradictory when conditions",
					Suggestion: "Simplify rules by consolidating conditions",
					JobName:    jobName,
				})
			}
		}
	}
}

// checkTemplateComplexity detects overly complex template inheritance chains
func checkTemplateComplexity(config *parser.GitLabConfig, result *AnalysisResult) {
	templates := getTemplates(config)
	templateDepths := make(map[string]int)
	
	// Calculate inheritance depth for each template
	for templateName := range templates {
		depth := calculateTemplateDepth(templateName, templates, make(map[string]bool))
		templateDepths[templateName] = depth
		
		if depth > 3 {
			result.Issues = append(result.Issues, Issue{
				Type:       IssueTypeMaintainability,
				Severity:   SeverityMedium,
				Path:       "templates." + templateName,
				Message:    fmt.Sprintf("Template has deep inheritance chain (depth: %d)", depth),
				Suggestion: "Consider flattening template hierarchy for better maintainability",
			})
		}
	}
	
	// Check for jobs using deeply inherited templates
	for jobName, job := range config.Jobs {
		extends := job.GetExtends()
		if len(extends) > 0 {
			for _, extendedTemplate := range extends {
				if depth, exists := templateDepths[extendedTemplate]; exists && depth > 2 {
					result.Issues = append(result.Issues, Issue{
						Type:       IssueTypeMaintainability,
						Severity:   SeverityLow,
						Path:       "jobs." + jobName + ".extends",
						Message:    fmt.Sprintf("Job extends deeply nested template (depth: %d)", depth),
						Suggestion: "Consider using a flatter template structure",
						JobName:    jobName,
					})
				}
			}
		}
	}
}

// checkRedundantInheritance detects redundant code in inheritance chains
func checkRedundantInheritance(config *parser.GitLabConfig, result *AnalysisResult) {
	templates := getTemplates(config)
	
	// Check for redundant before_script inheritance
	for templateName, template := range templates {
		extends := template.GetExtends()
		if len(extends) > 0 {
			for _, parentName := range extends {
				parentTemplate := templates[parentName]
				if parentTemplate != nil && len(template.BeforeScript) > 0 && len(parentTemplate.BeforeScript) > 0 {
					// Check if child template repeats parent commands
					redundantCommands := findRedundantCommands(template.BeforeScript, parentTemplate.BeforeScript)
					if len(redundantCommands) > 0 {
						result.Issues = append(result.Issues, Issue{
							Type:       IssueTypeMaintainability,
							Severity:   SeverityMedium,
							Path:       "templates." + templateName + ".before_script",
							Message:    fmt.Sprintf("Template repeats commands from parent: %s", strings.Join(redundantCommands, ", ")),
							Suggestion: "Remove redundant commands already defined in parent template",
						})
					}
				}
			}
		}
	}
}

// checkMatrixOpportunities detects jobs that could benefit from parallel matrix
func checkMatrixOpportunities(config *parser.GitLabConfig, result *AnalysisResult) {
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
			result.Issues = append(result.Issues, Issue{
				Type:       IssueTypePerformance,
				Severity:   SeverityMedium,
				Path:       "jobs",
				Message:    fmt.Sprintf("Multiple similar jobs in stage '%s' could use matrix strategy: %s", stage, strings.Join(jobNames, ", ")),
				Suggestion: "Consider consolidating similar jobs using parallel:matrix for better maintainability",
			})
		}
	}
}

// checkIncludeOptimization detects include optimization opportunities
func checkIncludeOptimization(config *parser.GitLabConfig, result *AnalysisResult) {
	if len(config.Include) > 5 {
		result.Issues = append(result.Issues, Issue{
			Type:       IssueTypeMaintainability,
			Severity:   SeverityMedium,
			Path:       "include",
			Message:    fmt.Sprintf("Many include statements (%d) may indicate fragmented configuration", len(config.Include)),
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
		result.Issues = append(result.Issues, Issue{
			Type:       IssueTypeMaintainability,
			Severity:   SeverityLow,
			Path:       "include",
			Message:    fmt.Sprintf("Multiple local includes (%d) could be consolidated", localIncludes),
			Suggestion: "Consider grouping related local includes into fewer files",
		})
	}
}

// checkExternalIncludeDuplication detects redundant external includes
func checkExternalIncludeDuplication(config *parser.GitLabConfig, result *AnalysisResult) {
	projectIncludes := make(map[string][]string)
	
	for _, include := range config.Include {
		if include.Project != "" {
			// Handle both single file and multiple files
			for _, file := range include.File {
				projectIncludes[include.Project] = append(projectIncludes[include.Project], file)
			}
		}
	}
	
	// Check for multiple includes from same external project
	for project, files := range projectIncludes {
		if len(files) > 3 {
			result.Issues = append(result.Issues, Issue{
				Type:       IssueTypeMaintainability,
				Severity:   SeverityMedium,
				Path:       "include",
				Message:    fmt.Sprintf("Multiple includes from same project '%s': %s", project, strings.Join(files, ", ")),
				Suggestion: "Consider using consolidated include files from external projects",
			})
		}
	}
}

// Helper functions

func getTemplates(config *parser.GitLabConfig) map[string]*parser.JobConfig {
	templates := make(map[string]*parser.JobConfig)
	
	for jobName, job := range config.Jobs {
		if strings.HasPrefix(jobName, ".") {
			templates[jobName] = job
		}
	}
	
	return templates
}

func calculateTemplateDepth(templateName string, templates map[string]*parser.JobConfig, visited map[string]bool) int {
	if visited[templateName] {
		return 0 // Circular reference protection
	}
	
	template := templates[templateName]
	if template == nil {
		return 1
	}
	
	extends := template.GetExtends()
	if len(extends) == 0 {
		return 1
	}
	
	visited[templateName] = true
	maxDepth := 0
	for _, parent := range extends {
		depth := calculateTemplateDepth(parent, templates, visited)
		if depth > maxDepth {
			maxDepth = depth
		}
	}
	delete(visited, templateName)
	
	return 1 + maxDepth
}

func findRedundantCommands(childCommands, parentCommands []string) []string {
	var redundant []string
	parentSet := make(map[string]bool)
	
	for _, cmd := range parentCommands {
		parentSet[strings.TrimSpace(cmd)] = true
	}
	
	for _, cmd := range childCommands {
		trimmed := strings.TrimSpace(cmd)
		if parentSet[trimmed] {
			redundant = append(redundant, trimmed)
		}
	}
	
	return redundant
}

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
	scriptSimilarity := 0
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
		
		// Different images often indicate matrix opportunity (node:14, node:16, etc.)
		if job.Image != firstJob.Image && job.Image != "" && firstJob.Image != "" {
			differentImages++
		}
		
		// Check script similarity (if scripts exist)
		if len(job.Script) > 0 && len(firstJob.Script) > 0 {
			overlap := calculateScriptOverlap(job.Script, firstJob.Script)
			if overlap > 0.7 { // 70% script overlap
				scriptSimilarity++
			}
		} else if len(job.Script) == 0 && len(firstJob.Script) == 0 {
			// Both have no scripts - still could be matrix candidates
			scriptSimilarity++
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
	similarStructure := scriptSimilarity >= totalJobs/2 || (len(firstJob.Script) == 0)
	
	return sameStageAllJobs && hasVariations && similarStructure
}

// calculateScriptOverlap calculates the percentage of overlapping commands between two script arrays
func calculateScriptOverlap(script1, script2 []string) float64 {
	if len(script1) == 0 || len(script2) == 0 {
		return 0.0
	}
	
	// Create normalized command sets
	set1 := make(map[string]bool)
	set2 := make(map[string]bool)
	
	for _, cmd := range script1 {
		normalized := strings.TrimSpace(strings.ToLower(cmd))
		if normalized != "" {
			set1[normalized] = true
		}
	}
	
	for _, cmd := range script2 {
		normalized := strings.TrimSpace(strings.ToLower(cmd))
		if normalized != "" {
			set2[normalized] = true
		}
	}
	
	// Count common commands
	common := 0
	for cmd := range set1 {
		if set2[cmd] {
			common++
		}
	}
	
	// Return overlap as percentage of smaller set
	minSize := len(set1)
	if len(set2) < minSize {
		minSize = len(set2)
	}
	
	if minSize == 0 {
		return 0.0
	}
	
	return float64(common) / float64(minSize)
}

// variablesEqual compares two variable maps for equality
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

// checkMissingExtends detects opportunities for using extends/templates
func checkMissingExtends(config *parser.GitLabConfig, result *AnalysisResult) {
	jobGroups := make(map[string][]string) // Similar job patterns -> job names
	templateCount := 0
	
	// Count existing templates
	for jobName := range config.Jobs {
		if strings.HasPrefix(jobName, ".") {
			templateCount++
		}
	}
	
	// Group jobs by similar patterns
	for jobName, job := range config.Jobs {
		if strings.HasPrefix(jobName, ".") {
			continue // Skip templates
		}
		
		// Create job signature based on image, stage, and common patterns
		signature := fmt.Sprintf("stage:%s|image:%s|scripts:%d", job.Stage, job.Image, len(job.Script))
		jobGroups[signature] = append(jobGroups[signature], jobName)
	}
	
	// Check for groups that could benefit from templates
	consolidationOpportunities := 0
	for _, jobNames := range jobGroups {
		if len(jobNames) >= 3 { // 3+ similar jobs
			consolidationOpportunities++
		}
	}
	
	// If there are consolidation opportunities but few/no templates, suggest extends
	if consolidationOpportunities > 0 && templateCount < consolidationOpportunities/2 {
		result.Issues = append(result.Issues, Issue{
			Type:       IssueTypeMaintainability,
			Severity:   SeverityMedium,
			Path:       "jobs",
			Message:    fmt.Sprintf("Found %d groups of similar jobs that could benefit from template extraction", consolidationOpportunities),
			Suggestion: "Consider using extends and templates to reduce duplication and improve maintainability",
		})
	}
}

// checkMissingNeeds detects opportunities for better dependency management
func checkMissingNeeds(config *parser.GitLabConfig, result *AnalysisResult) {
	// Check for jobs that use dependencies but could benefit from needs
	needsOpportunities := 0
	
	for _, job := range config.Jobs {
		if len(job.Dependencies) > 0 && job.Needs == nil {
			needsOpportunities++
		}
	}
	
	if needsOpportunities > 2 {
		result.Issues = append(result.Issues, Issue{
			Type:       IssueTypePerformance,
			Severity:   SeverityMedium,
			Path:       "jobs.*.dependencies",
			Message:    fmt.Sprintf("Found %d jobs using dependencies that could benefit from 'needs' for better parallelization", needsOpportunities),
			Suggestion: "Consider using 'needs' instead of 'dependencies' for more granular job control and better parallelization",
		})
	}
	
	// Check for stage-based dependencies that could be optimized
	stageJobs := make(map[string][]string)
	for jobName, job := range config.Jobs {
		stage := job.Stage
		if stage == "" {
			stage = "test" // Default
		}
		stageJobs[stage] = append(stageJobs[stage], jobName)
	}
	
	// Look for opportunities where jobs could run in parallel
	for stage, jobNames := range stageJobs {
		if len(jobNames) > 3 {
			// Check if these jobs have unnecessary sequential dependencies
			parallelizable := 0
			for _, jobName := range jobNames {
				job := config.Jobs[jobName]
				if len(job.Dependencies) == 0 && job.Needs == nil {
					parallelizable++
				}
			}
			
			if parallelizable >= len(jobNames)-1 { // Most jobs can run in parallel
				result.Issues = append(result.Issues, Issue{
					Type:       IssueTypePerformance,
					Severity:   SeverityLow,
					Path:       "stages." + stage,
					Message:    fmt.Sprintf("Stage '%s' has %d jobs that could potentially run in parallel", stage, len(jobNames)),
					Suggestion: "Consider optimizing job dependencies to improve pipeline parallelization",
				})
			}
		}
	}
}

// checkMissingTemplates detects configurations that would benefit from template extraction
func checkMissingTemplates(config *parser.GitLabConfig, result *AnalysisResult) {
	// Check for repeated patterns that indicate template opportunities
	beforeScriptPatterns := make(map[string]int)
	setupPatterns := make(map[string]int)
	
	for _, job := range config.Jobs {
		// Count before_script patterns
		if len(job.BeforeScript) > 0 {
			pattern := strings.Join(job.BeforeScript, "|")
			beforeScriptPatterns[pattern]++
		}
		
		// Count setup patterns (image + services combo)
		if job.Image != "" || len(job.Services) > 0 {
			setup := fmt.Sprintf("img:%s|svc:%s", job.Image, strings.Join(job.Services, ","))
			setupPatterns[setup]++
		}
	}
	
	// Check for repeated patterns
	templateOpportunities := 0
	for _, count := range beforeScriptPatterns {
		if count >= 3 {
			templateOpportunities++
		}
	}
	
	for _, count := range setupPatterns {
		if count >= 3 {
			templateOpportunities++
		}
	}
	
	if templateOpportunities > 0 {
		result.Issues = append(result.Issues, Issue{
			Type:       IssueTypeMaintainability,
			Severity:   SeverityMedium,
			Path:       "templates",
			Message:    fmt.Sprintf("Found %d patterns that could benefit from template extraction", templateOpportunities),
			Suggestion: "Consider creating reusable templates for common job patterns to improve maintainability",
		})
	}
}

// extractExtendsNames extracts template names from extends field
func extractExtendsNames(extends interface{}) []string {
	if extends == nil {
		return []string{}
	}
	
	switch v := extends.(type) {
	case string:
		return []string{v}
	case []string:
		return v
	case []interface{}:
		names := make([]string, 0, len(v))
		for _, item := range v {
			if str, ok := item.(string); ok {
				names = append(names, str)
			}
		}
		return names
	default:
		return []string{}
	}
}

// containsAllJobs checks if all jobs in jobsToCheck are in templateJobs
func containsAllJobs(templateJobs []string, jobsToCheck []string) bool {
	if len(jobsToCheck) > len(templateJobs) {
		return false
	}
	
	templateJobsSet := make(map[string]bool)
	for _, job := range templateJobs {
		templateJobsSet[job] = true
	}
	
	for _, job := range jobsToCheck {
		if !templateJobsSet[job] {
			return false
		}
	}
	
	return true
}
