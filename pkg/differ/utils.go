package differ

import (
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/wonderfulspam/gitlab-smith/pkg/parser"
)

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
