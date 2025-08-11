package security

import (
	"fmt"
	"regexp"
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

	// Build variable context for expansion
	variables := make(map[string]string)

	// Add ALL global variables from the config
	if config.Variables != nil {
		for key, value := range config.Variables {
			if str, ok := value.(string); ok {
				variables[key] = str
			} else {
				// Handle non-string values by converting to string
				variables[key] = fmt.Sprintf("%v", value)
			}
		}
	}

	// Add common GitLab CI predefined variables with reasonable defaults for analysis
	commonVars := map[string]string{
		"CI_REGISTRY_IMAGE":  "registry.gitlab.com/group/project",
		"CI_COMMIT_REF_SLUG": "main",
		"CI_COMMIT_SHA":      "abcd1234",
		"CI_PROJECT_PATH":    "group/project",
		"CI_PROJECT_NAME":    "project",
		"CI_PIPELINE_ID":     "12345",
	}
	for key, value := range commonVars {
		if _, exists := variables[key]; !exists {
			variables[key] = value
		}
	}

	// Helper function to expand variables in image strings
	expandVariables := func(image string, jobVars map[string]interface{}) string {
		expanded := image

		// Create job-specific variable context
		jobVariables := make(map[string]string)
		for k, v := range variables {
			jobVariables[k] = v
		}
		if jobVars != nil {
			for key, value := range jobVars {
				if str, ok := value.(string); ok {
					jobVariables[key] = str
				} else {
					// Handle non-string values by converting to string
					jobVariables[key] = fmt.Sprintf("%v", value)
				}
			}
		}

		// Regex patterns for variable substitution
		varPattern := regexp.MustCompile(`\$\{?([A-Za-z_][A-Za-z0-9_]*)\}?`)

		expanded = varPattern.ReplaceAllStringFunc(expanded, func(match string) string {
			// Extract variable name (handle both $VAR and ${VAR} formats)
			varName := varPattern.FindStringSubmatch(match)[1]
			if value, exists := jobVariables[varName]; exists {
				return value
			}
			// Return original if variable not found (could be dynamic/runtime variable)
			return match
		})

		return expanded
	}

	checkImage := func(image, path, jobName string, jobVars map[string]interface{}) {
		if image == "" {
			return
		}

		// Expand variables first
		expandedImage := expandVariables(image, jobVars)

		// If expansion didn't resolve all variables, skip tag checking
		if strings.Contains(expandedImage, "$") {
			return
		}

		// Check for missing tag after expansion
		if !strings.Contains(expandedImage, ":") {
			issues = append(issues, types.Issue{
				Type:       types.IssueTypeSecurity,
				Severity:   types.SeverityMedium,
				Path:       path,
				Message:    "Docker image without explicit tag: " + image + " (expands to: " + expandedImage + ")",
				Suggestion: "Use specific tags instead of 'latest' for reproducible builds",
				JobName:    jobName,
			})
		} else if strings.HasSuffix(expandedImage, ":latest") {
			issues = append(issues, types.Issue{
				Type:       types.IssueTypeSecurity,
				Severity:   types.SeverityLow,
				Path:       path,
				Message:    "Using 'latest' tag: " + image + " (expands to: " + expandedImage + ")",
				Suggestion: "Pin to specific version for reproducible builds",
				JobName:    jobName,
			})
		}
	}

	// Check default image
	if config.Default != nil {
		checkImage(config.Default.Image, "default.image", "", config.Default.Variables)
	}

	// Check job-specific images
	for jobName, job := range config.Jobs {
		checkImage(job.Image, "jobs."+jobName+".image", jobName, job.Variables)
	}

	return issues
}

func CheckEnvironmentVariables(config *parser.GitLabConfig) []types.Issue {
	var issues []types.Issue

	// Check for potential security issues in variable names
	checkVars := func(vars map[string]interface{}, path string) {
		for varName, value := range vars {
			varLower := strings.ToLower(varName)

			// Skip test/development variables that are obviously test data
			isTestVar := strings.Contains(varLower, "test") ||
				strings.Contains(varLower, "dev") ||
				strings.Contains(varLower, "example") ||
				strings.Contains(varLower, "demo")

			// Skip variables with obvious test values
			valueStr := fmt.Sprintf("%v", value)
			isTestValue := valueStr == "test" || valueStr == "example" ||
				valueStr == "demo"

			if !isTestVar && !isTestValue &&
				(strings.Contains(varLower, "password") ||
					strings.Contains(varLower, "secret") ||
					strings.Contains(varLower, "token")) {
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
