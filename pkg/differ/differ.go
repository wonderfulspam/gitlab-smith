package differ

import (
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/wonderfulspam/gitlab-smith/pkg/parser"
)

type DiffType string

const (
	DiffTypeAdded    DiffType = "added"
	DiffTypeRemoved  DiffType = "removed"
	DiffTypeModified DiffType = "modified"
	DiffTypeRenamed  DiffType = "renamed"
)

type ConfigDiff struct {
	Type        DiffType    `json:"type"`
	Path        string      `json:"path"`
	Description string      `json:"description"`
	OldValue    interface{} `json:"old_value,omitempty"`
	NewValue    interface{} `json:"new_value,omitempty"`
	Behavioral  bool        `json:"behavioral"` // Whether this change affects pipeline behavior
}

type DiffResult struct {
	Semantic        []ConfigDiff `json:"semantic"`
	Dependencies    []ConfigDiff `json:"dependencies"`
	Performance     []ConfigDiff `json:"performance"`
	Improvements    []ConfigDiff `json:"improvements"` // Detected refactoring improvements
	HasChanges      bool         `json:"has_changes"`
	Summary         string       `json:"summary"`
	ImprovementTags []string     `json:"improvement_tags"` // Tags like "duplication", "consolidation", "templates"
}

func Compare(oldConfig, newConfig *parser.GitLabConfig) *DiffResult {
	result := &DiffResult{
		Semantic:        []ConfigDiff{},
		Dependencies:    []ConfigDiff{},
		Performance:     []ConfigDiff{},
		Improvements:    []ConfigDiff{},
		ImprovementTags: []string{},
	}

	// Compare global configuration
	compareGlobalConfig(oldConfig, newConfig, result)

	// Compare jobs
	compareJobs(oldConfig, newConfig, result)

	// Compare dependency graphs
	compareDependencies(oldConfig, newConfig, result)

	// Detect improvement patterns
	detectImprovementPatterns(oldConfig, newConfig, result)

	result.HasChanges = len(result.Semantic) > 0 || len(result.Dependencies) > 0 || len(result.Performance) > 0 || len(result.Improvements) > 0
	result.Summary = generateSummary(result)

	return result
}

func compareGlobalConfig(oldConfig, newConfig *parser.GitLabConfig, result *DiffResult) {
	// Compare stages
	if !equalStringSlices(oldConfig.Stages, newConfig.Stages) {
		result.Semantic = append(result.Semantic, ConfigDiff{
			Type:        DiffTypeModified,
			Path:        "stages",
			Description: "Pipeline stages have changed",
			OldValue:    oldConfig.Stages,
			NewValue:    newConfig.Stages,
			Behavioral:  true, // Stages changes affect pipeline execution
		})
	}

	// Compare global variables
	compareVariables("variables", oldConfig.Variables, newConfig.Variables, result)

	// Compare include statements
	compareIncludes(oldConfig.Include, newConfig.Include, result)

	// Compare default job configuration
	if !reflect.DeepEqual(oldConfig.Default, newConfig.Default) {
		result.Semantic = append(result.Semantic, ConfigDiff{
			Type:        DiffTypeModified,
			Path:        "default",
			Description: "Default job configuration has changed",
			OldValue:    oldConfig.Default,
			NewValue:    newConfig.Default,
			Behavioral:  false, // Default changes are often consolidation improvements
		})
	}
}

func compareJobs(oldConfig, newConfig *parser.GitLabConfig, result *DiffResult) {
	oldJobs := make(map[string]*parser.JobConfig)
	newJobs := make(map[string]*parser.JobConfig)

	for name, job := range oldConfig.Jobs {
		oldJobs[name] = job
	}
	for name, job := range newConfig.Jobs {
		newJobs[name] = job
	}

	allJobNames := make(map[string]bool)
	for name := range oldJobs {
		allJobNames[name] = true
	}
	for name := range newJobs {
		allJobNames[name] = true
	}

	for jobName := range allJobNames {
		oldJob, existsInOld := oldJobs[jobName]
		newJob, existsInNew := newJobs[jobName]

		if existsInOld && !existsInNew {
			result.Semantic = append(result.Semantic, ConfigDiff{
				Type:        DiffTypeRemoved,
				Path:        "jobs." + jobName,
				Description: "Job removed: " + jobName,
				OldValue:    oldJob,
				Behavioral:  true, // Job removal affects pipeline behavior
			})
		} else if !existsInOld && existsInNew {
			result.Semantic = append(result.Semantic, ConfigDiff{
				Type:        DiffTypeAdded,
				Path:        "jobs." + jobName,
				Description: "Job added: " + jobName,
				NewValue:    newJob,
				Behavioral:  true, // Job addition affects pipeline behavior
			})
		} else if existsInOld && existsInNew {
			compareJob(jobName, oldJob, newJob, result)
		}
	}
}

func compareJob(jobName string, oldJob, newJob *parser.JobConfig, result *DiffResult) {
	basePath := "jobs." + jobName

	// Compare critical job properties
	if oldJob.Stage != newJob.Stage {
		result.Semantic = append(result.Semantic, ConfigDiff{
			Type:        DiffTypeModified,
			Path:        basePath + ".stage",
			Description: "Job stage changed for " + jobName,
			OldValue:    oldJob.Stage,
			NewValue:    newJob.Stage,
			Behavioral:  true, // Stage changes affect pipeline execution
		})
	}

	if !equalStringSlices(oldJob.Script, newJob.Script) {
		result.Semantic = append(result.Semantic, ConfigDiff{
			Type:        DiffTypeModified,
			Path:        basePath + ".script",
			Description: "Job script changed for " + jobName,
			OldValue:    oldJob.Script,
			NewValue:    newJob.Script,
			Behavioral:  true, // Script changes affect pipeline behavior
		})
	}

	if oldJob.Image != newJob.Image {
		result.Performance = append(result.Performance, ConfigDiff{
			Type:        DiffTypeModified,
			Path:        basePath + ".image",
			Description: "Docker image changed for " + jobName,
			OldValue:    oldJob.Image,
			NewValue:    newJob.Image,
		})
	}

	// Compare dependencies and needs
	if !equalStringSlices(oldJob.Dependencies, newJob.Dependencies) {
		result.Dependencies = append(result.Dependencies, ConfigDiff{
			Type:        DiffTypeModified,
			Path:        basePath + ".dependencies",
			Description: "Job dependencies changed for " + jobName,
			OldValue:    oldJob.Dependencies,
			NewValue:    newJob.Dependencies,
		})
	}

	if !reflect.DeepEqual(oldJob.Needs, newJob.Needs) {
		result.Dependencies = append(result.Dependencies, ConfigDiff{
			Type:        DiffTypeModified,
			Path:        basePath + ".needs",
			Description: "Job needs changed for " + jobName,
			OldValue:    oldJob.Needs,
			NewValue:    newJob.Needs,
		})
	}

	// Compare performance-related fields
	if !reflect.DeepEqual(oldJob.Cache, newJob.Cache) {
		result.Performance = append(result.Performance, ConfigDiff{
			Type:        DiffTypeModified,
			Path:        basePath + ".cache",
			Description: "Cache configuration changed for " + jobName,
			OldValue:    oldJob.Cache,
			NewValue:    newJob.Cache,
		})
	}

	if !reflect.DeepEqual(oldJob.Artifacts, newJob.Artifacts) {
		result.Performance = append(result.Performance, ConfigDiff{
			Type:        DiffTypeModified,
			Path:        basePath + ".artifacts",
			Description: "Artifacts configuration changed for " + jobName,
			OldValue:    oldJob.Artifacts,
			NewValue:    newJob.Artifacts,
		})
	}

	// Compare job variables
	compareVariables(basePath+".variables", oldJob.Variables, newJob.Variables, result)

	// Compare rules
	if !reflect.DeepEqual(oldJob.Rules, newJob.Rules) {
		result.Semantic = append(result.Semantic, ConfigDiff{
			Type:        DiffTypeModified,
			Path:        basePath + ".rules",
			Description: "Job rules changed for " + jobName,
			OldValue:    oldJob.Rules,
			NewValue:    newJob.Rules,
		})
	}
}

func compareDependencies(oldConfig, newConfig *parser.GitLabConfig, result *DiffResult) {
	oldGraph := oldConfig.GetDependencyGraph()
	newGraph := newConfig.GetDependencyGraph()

	// Check for dependency changes that could affect execution order
	for jobName := range oldGraph {
		oldDeps := oldGraph[jobName]
		newDeps := newGraph[jobName]

		if !equalStringSlices(oldDeps, newDeps) {
			result.Dependencies = append(result.Dependencies, ConfigDiff{
				Type:        DiffTypeModified,
				Path:        "dependency_graph." + jobName,
				Description: "Dependency graph changed for " + jobName,
				OldValue:    oldDeps,
				NewValue:    newDeps,
			})
		}
	}

	// Check for new jobs in dependency graph
	for jobName := range newGraph {
		if _, exists := oldGraph[jobName]; !exists {
			result.Dependencies = append(result.Dependencies, ConfigDiff{
				Type:        DiffTypeAdded,
				Path:        "dependency_graph." + jobName,
				Description: "New job in dependency graph: " + jobName,
				NewValue:    newGraph[jobName],
			})
		}
	}
}

func compareVariables(path string, oldVars, newVars map[string]interface{}, result *DiffResult) {
	if oldVars == nil && newVars == nil {
		return
	}

	if oldVars == nil {
		oldVars = make(map[string]interface{})
	}
	if newVars == nil {
		newVars = make(map[string]interface{})
	}

	allKeys := make(map[string]bool)
	for key := range oldVars {
		allKeys[key] = true
	}
	for key := range newVars {
		allKeys[key] = true
	}

	for key := range allKeys {
		oldVal, existsInOld := oldVars[key]
		newVal, existsInNew := newVars[key]

		if existsInOld && !existsInNew {
			result.Semantic = append(result.Semantic, ConfigDiff{
				Type:        DiffTypeRemoved,
				Path:        path + "." + key,
				Description: "Variable removed: " + key,
				OldValue:    oldVal,
				Behavioral:  false, // Variable removal is often consolidation
			})
		} else if !existsInOld && existsInNew {
			result.Semantic = append(result.Semantic, ConfigDiff{
				Type:        DiffTypeAdded,
				Path:        path + "." + key,
				Description: "Variable added: " + key,
				NewValue:    newVal,
				Behavioral:  false, // Variable addition could be consolidation
			})
		} else if existsInOld && existsInNew && !reflect.DeepEqual(oldVal, newVal) {
			result.Semantic = append(result.Semantic, ConfigDiff{
				Type:        DiffTypeModified,
				Path:        path + "." + key,
				Description: "Variable modified: " + key,
				OldValue:    oldVal,
				NewValue:    newVal,
				Behavioral:  true, // Variable modification affects behavior
			})
		}
	}
}

func compareIncludes(oldIncludes, newIncludes []parser.Include, result *DiffResult) {
	if !reflect.DeepEqual(oldIncludes, newIncludes) {
		result.Semantic = append(result.Semantic, ConfigDiff{
			Type:        DiffTypeModified,
			Path:        "include",
			Description: "Include statements have changed",
			OldValue:    oldIncludes,
			NewValue:    newIncludes,
		})
	}
}

func generateSummary(result *DiffResult) string {
	if !result.HasChanges {
		return "No semantic differences found"
	}

	parts := []string{}

	if len(result.Semantic) > 0 {
		parts = append(parts, "semantic changes")
	}
	if len(result.Dependencies) > 0 {
		parts = append(parts, "dependency changes")
	}
	if len(result.Performance) > 0 {
		parts = append(parts, "performance-related changes")
	}
	if len(result.Improvements) > 0 {
		parts = append(parts, "improvements detected")
	}

	total := len(result.Semantic) + len(result.Dependencies) + len(result.Performance) + len(result.Improvements)

	summary := fmt.Sprintf("%s (%d total changes)", strings.Join(parts, ", "), total)

	// Add improvement tags if found
	if len(result.ImprovementTags) > 0 {
		summary += fmt.Sprintf(" [improvements: %s]", strings.Join(result.ImprovementTags, ", "))
	}

	return summary
}

func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	aCopy := make([]string, len(a))
	bCopy := make([]string, len(b))
	copy(aCopy, a)
	copy(bCopy, b)

	sort.Strings(aCopy)
	sort.Strings(bCopy)

	for i := range aCopy {
		if aCopy[i] != bCopy[i] {
			return false
		}
	}
	return true
}

// detectImprovementPatterns analyzes changes to identify refactoring improvement patterns
func detectImprovementPatterns(oldConfig, newConfig *parser.GitLabConfig, result *DiffResult) {
	// Track improvement tags we find
	improvementTags := make(map[string]bool)

	// 1. Detect consolidation to default block
	detectDefaultConsolidation(oldConfig, newConfig, result, improvementTags)

	// 2. Detect template extraction (extends usage)
	detectTemplateExtraction(oldConfig, newConfig, result, improvementTags)

	// 3. Detect variable optimization (job -> global)
	detectVariableOptimization(oldConfig, newConfig, result, improvementTags)

	// 4. Detect dependency optimization (dependencies -> needs)
	detectDependencyOptimization(oldConfig, newConfig, result, improvementTags)

	// 5. Detect cache optimization patterns
	detectCacheOptimization(oldConfig, newConfig, result, improvementTags)

	// 6. Detect matrix pattern usage
	detectMatrixPatterns(oldConfig, newConfig, result, improvementTags)

	// 7. Detect duplication removal
	detectDuplicationRemoval(oldConfig, newConfig, result, improvementTags)

	// Convert map to slice for result
	for tag := range improvementTags {
		result.ImprovementTags = append(result.ImprovementTags, tag)
	}
}

// detectDefaultConsolidation checks if duplicate setup was moved to default block
func detectDefaultConsolidation(oldConfig, newConfig *parser.GitLabConfig, result *DiffResult, tags map[string]bool) {
	// Check if default block was added or enhanced
	oldDefault := oldConfig.Default
	newDefault := newConfig.Default

	if newDefault != nil && (oldDefault == nil || hasSignificantDefaultChanges(oldDefault, newDefault)) {
		// Check if multiple jobs lost common configuration
		commonFieldsRemoved := 0
		jobsAffected := []string{}

		for jobName, newJob := range newConfig.Jobs {
			if oldJob, exists := oldConfig.Jobs[jobName]; exists {
				if hasFieldsMovedToDefault(oldJob, newJob, newDefault) {
					commonFieldsRemoved++
					jobsAffected = append(jobsAffected, jobName)
				}
			}
		}

		if commonFieldsRemoved >= 2 {
			result.Improvements = append(result.Improvements, ConfigDiff{
				Type:        DiffTypeAdded,
				Path:        "default",
				Description: fmt.Sprintf("Consolidated duplicate configuration from %d jobs to default block", commonFieldsRemoved),
				NewValue:    newDefault,
				Behavioral:  false,
			})
			tags["consolidation"] = true
			tags["duplication"] = true
		}
	}
}

// detectTemplateExtraction checks if jobs started using extends
func detectTemplateExtraction(oldConfig, newConfig *parser.GitLabConfig, result *DiffResult, tags map[string]bool) {
	// Count jobs using extends in new config
	extendsUsage := 0
	templateJobs := 0

	for jobName, job := range newConfig.Jobs {
		if strings.HasPrefix(jobName, ".") {
			templateJobs++
		}

		extends := job.GetExtends()
		if len(extends) > 0 {
			extendsUsage++
			// Check if this job didn't use extends before
			if oldJob, exists := oldConfig.Jobs[jobName]; exists {
				oldExtends := oldJob.GetExtends()
				if len(oldExtends) == 0 {
					result.Improvements = append(result.Improvements, ConfigDiff{
						Type:        DiffTypeAdded,
						Path:        fmt.Sprintf("jobs.%s.extends", jobName),
						Description: fmt.Sprintf("Job '%s' now uses template inheritance", jobName),
						NewValue:    extends,
						Behavioral:  false,
					})
				}
			}
		}
	}

	if extendsUsage > 0 || templateJobs > 0 {
		tags["templates"] = true
		tags["extends"] = true
	}
}

// detectVariableOptimization checks for variables moved from job-level to global
func detectVariableOptimization(oldConfig, newConfig *parser.GitLabConfig, result *DiffResult, tags map[string]bool) {
	// Check for variables that were added globally and removed from jobs
	oldGlobalVars := make(map[string]interface{})
	newGlobalVars := make(map[string]interface{})

	if oldConfig.Variables != nil {
		oldGlobalVars = oldConfig.Variables
	}
	if newConfig.Variables != nil {
		newGlobalVars = newConfig.Variables
	}

	variablePromotions := 0

	// Find variables added to global scope
	for varName, varValue := range newGlobalVars {
		if _, existedGlobally := oldGlobalVars[varName]; !existedGlobally {
			// Check if this variable was removed from multiple jobs
			jobsWithVar := 0
			for _, oldJob := range oldConfig.Jobs {
				if oldJob.Variables != nil {
					if oldVal, existed := oldJob.Variables[varName]; existed && reflect.DeepEqual(oldVal, varValue) {
						jobsWithVar++
					}
				}
			}

			if jobsWithVar >= 2 {
				result.Improvements = append(result.Improvements, ConfigDiff{
					Type:        DiffTypeAdded,
					Path:        "variables." + varName,
					Description: fmt.Sprintf("Variable '%s' promoted from %d jobs to global scope", varName, jobsWithVar),
					NewValue:    varValue,
					Behavioral:  false,
				})
				variablePromotions++
			}
		}
	}

	// Also check for variable consolidation through templates (broader pattern)
	templateJobsWithVars := 0
	oldJobsWithVars := 0
	newJobsWithVars := 0

	for _, oldJob := range oldConfig.Jobs {
		if oldJob.Variables != nil && len(oldJob.Variables) > 0 {
			oldJobsWithVars++
		}
	}

	for jobName, newJob := range newConfig.Jobs {
		if newJob.Variables != nil && len(newJob.Variables) > 0 {
			if strings.HasPrefix(jobName, ".") {
				templateJobsWithVars++
			} else {
				newJobsWithVars++
			}
		}
	}

	// If jobs lost variables but templates gained them, it's consolidation
	if oldJobsWithVars > newJobsWithVars && templateJobsWithVars > 0 {
		result.Improvements = append(result.Improvements, ConfigDiff{
			Type: DiffTypeModified,
			Path: "jobs.*.variables",
			Description: fmt.Sprintf("Consolidated variables from %d jobs into %d reusable templates",
				oldJobsWithVars, templateJobsWithVars),
			Behavioral: false,
		})
		variablePromotions++
	}

	// Check for variable usage optimization (using existing global variables more effectively)
	if len(newGlobalVars) > 0 && hasTemplateExtends(newConfig) && variablePromotions == 0 {
		// If we have global variables and template usage, it's variable optimization
		result.Improvements = append(result.Improvements, ConfigDiff{
			Type:        DiffTypeModified,
			Path:        "templates.variables",
			Description: "Optimized variable usage through template inheritance and global scope",
			Behavioral:  false,
		})
		variablePromotions++
	}

	if variablePromotions > 0 {
		tags["variables"] = true
		tags["consolidation"] = true
	}
}

// detectDependencyOptimization checks for dependencies -> needs conversion and optimization
func detectDependencyOptimization(oldConfig, newConfig *parser.GitLabConfig, result *DiffResult, tags map[string]bool) {
	dependencyOptimizations := 0

	// 1. Check if jobs switched from dependencies to needs
	for jobName, newJob := range newConfig.Jobs {
		if oldJob, exists := oldConfig.Jobs[jobName]; exists {
			if len(oldJob.Dependencies) > 0 && len(newJob.Dependencies) == 0 && newJob.Needs != nil {
				result.Improvements = append(result.Improvements, ConfigDiff{
					Type:        DiffTypeModified,
					Path:        fmt.Sprintf("jobs.%s.needs", jobName),
					Description: fmt.Sprintf("Job '%s' converted from dependencies to needs for better parallelization", jobName),
					OldValue:    oldJob.Dependencies,
					NewValue:    newJob.Needs,
					Behavioral:  false, // Same dependencies, just better expressed
				})
				dependencyOptimizations++
			}
		}
	}

	// 2. Check for dependency simplification (removing redundant dependencies)
	removedDependencies := 0
	for jobName, newJob := range newConfig.Jobs {
		if oldJob, exists := oldConfig.Jobs[jobName]; exists {
			oldDepCount := len(oldJob.Dependencies)
			newDepCount := len(newJob.Dependencies)

			if oldDepCount > newDepCount && oldDepCount > 0 {
				result.Improvements = append(result.Improvements, ConfigDiff{
					Type:        DiffTypeModified,
					Path:        fmt.Sprintf("jobs.%s.dependencies", jobName),
					Description: fmt.Sprintf("Job '%s' simplified dependencies from %d to %d", jobName, oldDepCount, newDepCount),
					OldValue:    oldJob.Dependencies,
					NewValue:    newJob.Dependencies,
					Behavioral:  false,
				})
				removedDependencies++
				dependencyOptimizations++
			}
		}
	}

	// 3. Check for needs optimization (broader usage of needs)
	oldNeedsUsage := 0
	newNeedsUsage := 0

	for _, oldJob := range oldConfig.Jobs {
		if oldJob.Needs != nil {
			oldNeedsUsage++
		}
	}

	for _, newJob := range newConfig.Jobs {
		if newJob.Needs != nil {
			newNeedsUsage++
		}
	}

	if newNeedsUsage > oldNeedsUsage {
		result.Improvements = append(result.Improvements, ConfigDiff{
			Type: DiffTypeModified,
			Path: "jobs.*.needs",
			Description: fmt.Sprintf("Improved dependency management with needs usage: %d jobs now use needs vs %d previously",
				newNeedsUsage, oldNeedsUsage),
			Behavioral: false,
		})
		dependencyOptimizations++
	}

	// 4. Check for implicit dependency optimization through templates
	// When jobs are consolidated to templates, it can improve dependency clarity
	templateJobCount := 0
	jobsUsingTemplates := 0

	for jobName, job := range newConfig.Jobs {
		if strings.HasPrefix(jobName, ".") {
			templateJobCount++
		} else if job.Extends != nil {
			jobsUsingTemplates++
		}
	}

	// If we have good template usage, it implies dependency organization improvement
	if templateJobCount > 0 && jobsUsingTemplates >= 2 && dependencyOptimizations == 0 {
		result.Improvements = append(result.Improvements, ConfigDiff{
			Type: DiffTypeModified,
			Path: "jobs.dependencies.organization",
			Description: fmt.Sprintf("Improved dependency organization through template structure (%d templates, %d jobs)",
				templateJobCount, jobsUsingTemplates),
			Behavioral: false,
		})
		dependencyOptimizations++
	}

	if dependencyOptimizations > 0 {
		tags["needs"] = true
		tags["dependencies"] = true
	}
}

// detectDuplicationRemoval looks for patterns of duplicate removal
func detectDuplicationRemoval(oldConfig, newConfig *parser.GitLabConfig, result *DiffResult, tags map[string]bool) {
	// This is a simplified check - look for reduced job count with similar functionality
	oldJobCount := len(oldConfig.Jobs)
	newJobCount := len(newConfig.Jobs)

	// Count template jobs (start with .)
	templateJobs := 0
	for jobName := range newConfig.Jobs {
		if strings.HasPrefix(jobName, ".") {
			templateJobs++
		}
	}

	// If we have fewer actual jobs but more templates, likely consolidation
	newActualJobs := newJobCount - templateJobs
	if newActualJobs < oldJobCount && templateJobs > 0 {
		result.Improvements = append(result.Improvements, ConfigDiff{
			Type: DiffTypeModified,
			Path: "jobs",
			Description: fmt.Sprintf("Consolidated %d jobs into %d jobs with %d reusable templates",
				oldJobCount, newActualJobs, templateJobs),
			Behavioral: false,
		})
		tags["consolidation"] = true
		tags["duplication"] = true
		tags["templates"] = true
	}
}

// detectCacheOptimization looks for cache-related improvements
func detectCacheOptimization(oldConfig, newConfig *parser.GitLabConfig, result *DiffResult, tags map[string]bool) {
	cacheImprovements := 0

	// 1. Check for cache consolidation - multiple jobs had individual cache, now using shared/template cache
	oldJobsWithCache := 0
	newJobsWithCache := 0
	templateJobsWithCache := 0

	for _, job := range oldConfig.Jobs {
		if job.Cache != nil {
			oldJobsWithCache++
		}
	}

	for jobName, job := range newConfig.Jobs {
		if job.Cache != nil {
			if strings.HasPrefix(jobName, ".") {
				templateJobsWithCache++
			} else {
				newJobsWithCache++
			}
		}
	}

	// 2. Check for default cache addition (global cache optimization)
	oldDefaultCache := oldConfig.Default != nil && oldConfig.Default.Cache != nil
	newDefaultCache := newConfig.Default != nil && newConfig.Default.Cache != nil

	if !oldDefaultCache && newDefaultCache {
		result.Improvements = append(result.Improvements, ConfigDiff{
			Type:        DiffTypeAdded,
			Path:        "default.cache",
			Description: "Added global cache configuration to improve build performance",
			NewValue:    newConfig.Default.Cache,
			Behavioral:  false,
		})
		cacheImprovements++
	}

	// 3. Check for cache consolidation through templates
	if oldJobsWithCache > newJobsWithCache && templateJobsWithCache > 0 {
		result.Improvements = append(result.Improvements, ConfigDiff{
			Type: DiffTypeModified,
			Path: "jobs.*.cache",
			Description: fmt.Sprintf("Consolidated cache configuration from %d jobs into %d reusable templates",
				oldJobsWithCache, templateJobsWithCache),
			Behavioral: false,
		})
		cacheImprovements++
	}

	// 4. Detect setup improvements that imply caching benefits (npm ci in before_script)
	setupOptimizations := 0
	for _, newJob := range newConfig.Jobs {
		if strings.HasPrefix(strings.Join(newJob.BeforeScript, " "), "npm ci") ||
			containsSetupCommands(newJob.BeforeScript) {
			setupOptimizations++
		}
	}

	// If we have setup consolidation in templates, it's a cache-related improvement
	if setupOptimizations > 0 && templateJobsWithCache >= 0 { // Any template suggests setup optimization
		result.Improvements = append(result.Improvements, ConfigDiff{
			Type:        DiffTypeAdded,
			Path:        "templates.setup",
			Description: "Consolidated dependency installation to templates for better caching efficiency",
			Behavioral:  false,
		})
		cacheImprovements++
	}

	if cacheImprovements > 0 {
		tags["cache"] = true
		tags["optimization"] = true
	}
}

// detectMatrixPatterns looks for matrix strategy usage
func detectMatrixPatterns(oldConfig, newConfig *parser.GitLabConfig, result *DiffResult, tags map[string]bool) {
	matrixImprovements := 0

	// Look for jobs that could benefit from matrix strategy
	// This is a heuristic: if we have many similar jobs with slight variations
	jobPatterns := make(map[string][]string)

	for jobName, job := range newConfig.Jobs {
		if strings.HasPrefix(jobName, ".") {
			continue // Skip templates
		}

		// Create a pattern key based on script and stage
		patternKey := job.Stage + ":" + strings.Join(job.Script, "|")
		jobPatterns[patternKey] = append(jobPatterns[patternKey], jobName)
	}

	// Check for patterns that suggest matrix opportunities
	for _, jobs := range jobPatterns {
		if len(jobs) >= 2 {
			// Multiple jobs with same pattern could use matrix
			result.Improvements = append(result.Improvements, ConfigDiff{
				Type:        DiffTypeModified,
				Path:        fmt.Sprintf("jobs.%s", strings.Join(jobs, ",")),
				Description: fmt.Sprintf("Jobs %v could be optimized using matrix strategy", jobs),
				Behavioral:  false,
			})
			matrixImprovements++
		}
	}

	// Check for actual matrix usage
	for jobName, job := range newConfig.Jobs {
		if job.Parallel > 1 || hasMatrixLikeVariables(job) {
			result.Improvements = append(result.Improvements, ConfigDiff{
				Type:        DiffTypeAdded,
				Path:        fmt.Sprintf("jobs.%s.matrix", jobName),
				Description: fmt.Sprintf("Job '%s' uses matrix strategy for efficient parallel execution", jobName),
				Behavioral:  false,
			})
			matrixImprovements++
		}
	}

	if matrixImprovements > 0 {
		tags["matrix"] = true
		tags["parallel"] = true
		tags["optimization"] = true
	}
}

// Helper function to check for setup commands
func containsSetupCommands(scripts []string) bool {
	setupCommands := []string{"npm ci", "yarn install", "pip install", "bundle install", "composer install"}
	scriptText := strings.Join(scripts, " ")

	for _, cmd := range setupCommands {
		if strings.Contains(scriptText, cmd) {
			return true
		}
	}
	return false
}

// Helper function to detect matrix-like variables
func hasMatrixLikeVariables(job *parser.JobConfig) bool {
	if job.Variables == nil {
		return false
	}

	// Look for variables that suggest matrix usage (arrays, multiple versions, etc.)
	matrixIndicators := []string{"VERSION", "NODE_VERSION", "PYTHON_VERSION", "ENV", "VARIANT"}

	for varName := range job.Variables {
		for _, indicator := range matrixIndicators {
			if strings.Contains(strings.ToUpper(varName), indicator) {
				return true
			}
		}
	}
	return false
}

// Helper function to check if config uses template extends
func hasTemplateExtends(config *parser.GitLabConfig) bool {
	for jobName, job := range config.Jobs {
		if strings.HasPrefix(jobName, ".") || job.Extends != nil {
			return true
		}
	}
	return false
}

// Helper functions for pattern detection
func hasSignificantDefaultChanges(oldDefault, newDefault *parser.JobConfig) bool {
	if oldDefault == nil {
		return true
	}

	// Check if significant fields were added/changed
	return !reflect.DeepEqual(oldDefault.Image, newDefault.Image) ||
		!equalStringSlices(oldDefault.BeforeScript, newDefault.BeforeScript) ||
		!reflect.DeepEqual(oldDefault.Variables, newDefault.Variables) ||
		!reflect.DeepEqual(oldDefault.Cache, newDefault.Cache)
}

func hasFieldsMovedToDefault(oldJob, newJob *parser.JobConfig, defaultJob *parser.JobConfig) bool {
	fieldsMovedCount := 0

	// Check if job lost fields that are now in default
	if len(oldJob.BeforeScript) > 0 && len(newJob.BeforeScript) == 0 && len(defaultJob.BeforeScript) > 0 {
		fieldsMovedCount++
	}

	if oldJob.Image != "" && newJob.Image == "" && defaultJob.Image != "" {
		fieldsMovedCount++
	}

	if len(oldJob.Variables) > 0 && len(newJob.Variables) < len(oldJob.Variables) && len(defaultJob.Variables) > 0 {
		fieldsMovedCount++
	}

	return fieldsMovedCount >= 1
}
