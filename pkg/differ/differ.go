package differ

import (
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/emt/gitlab-smith/pkg/parser"
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
}

type DiffResult struct {
	Semantic     []ConfigDiff `json:"semantic"`
	Dependencies []ConfigDiff `json:"dependencies"`
	Performance  []ConfigDiff `json:"performance"`
	HasChanges   bool         `json:"has_changes"`
	Summary      string       `json:"summary"`
}

func Compare(oldConfig, newConfig *parser.GitLabConfig) *DiffResult {
	result := &DiffResult{
		Semantic:     []ConfigDiff{},
		Dependencies: []ConfigDiff{},
		Performance:  []ConfigDiff{},
	}

	// Compare global configuration
	compareGlobalConfig(oldConfig, newConfig, result)

	// Compare jobs
	compareJobs(oldConfig, newConfig, result)

	// Compare dependency graphs
	compareDependencies(oldConfig, newConfig, result)

	result.HasChanges = len(result.Semantic) > 0 || len(result.Dependencies) > 0 || len(result.Performance) > 0
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
			})
		} else if !existsInOld && existsInNew {
			result.Semantic = append(result.Semantic, ConfigDiff{
				Type:        DiffTypeAdded,
				Path:        "jobs." + jobName,
				Description: "Job added: " + jobName,
				NewValue:    newJob,
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
		})
	}

	if !equalStringSlices(oldJob.Script, newJob.Script) {
		result.Semantic = append(result.Semantic, ConfigDiff{
			Type:        DiffTypeModified,
			Path:        basePath + ".script",
			Description: "Job script changed for " + jobName,
			OldValue:    oldJob.Script,
			NewValue:    newJob.Script,
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
			})
		} else if !existsInOld && existsInNew {
			result.Semantic = append(result.Semantic, ConfigDiff{
				Type:        DiffTypeAdded,
				Path:        path + "." + key,
				Description: "Variable added: " + key,
				NewValue:    newVal,
			})
		} else if existsInOld && existsInNew && !reflect.DeepEqual(oldVal, newVal) {
			result.Semantic = append(result.Semantic, ConfigDiff{
				Type:        DiffTypeModified,
				Path:        path + "." + key,
				Description: "Variable modified: " + key,
				OldValue:    oldVal,
				NewValue:    newVal,
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

	total := len(result.Semantic) + len(result.Dependencies) + len(result.Performance)

	return fmt.Sprintf("%s (%d total changes)", strings.Join(parts, ", "), total)
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
