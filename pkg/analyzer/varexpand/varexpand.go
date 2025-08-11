package varexpand

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/wonderfulspam/gitlab-smith/pkg/parser"
)

// Expander handles GitLab CI variable expansion for analysis
type Expander struct {
	globalVars map[string]string
	commonVars map[string]string
}

// New creates a new variable expander for the given config
func New(config *parser.GitLabConfig) *Expander {
	expander := &Expander{
		globalVars: make(map[string]string),
		commonVars: map[string]string{
			"CI_REGISTRY_IMAGE":  "registry.gitlab.com/group/project",
			"CI_COMMIT_REF_SLUG": "main",
			"CI_COMMIT_SHA":      "abcd1234",
			"CI_PROJECT_PATH":    "group/project",
			"CI_PROJECT_NAME":    "project",
			"CI_PIPELINE_ID":     "12345",
		},
	}

	// Add ALL global variables from the config
	if config.Variables != nil {
		for key, value := range config.Variables {
			if str, ok := value.(string); ok {
				expander.globalVars[key] = str
			} else {
				// Handle non-string values by converting to string
				expander.globalVars[key] = fmt.Sprintf("%v", value)
			}
		}
	}

	return expander
}

// ExpandString expands variables in the given string with optional job-level variables
func (e *Expander) ExpandString(str string, jobVars map[string]interface{}) string {
	if !strings.Contains(str, "$") {
		return str
	}

	// Create job-specific variable context
	jobVariables := make(map[string]string)

	// Add common vars first
	for k, v := range e.commonVars {
		jobVariables[k] = v
	}

	// Add global vars (override common vars if defined)
	for k, v := range e.globalVars {
		jobVariables[k] = v
	}

	// Add job-level vars (override global vars if defined)
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

	expanded := varPattern.ReplaceAllStringFunc(str, func(match string) string {
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

// HasUnresolvedVariables checks if a string still contains unresolved variables
func (e *Expander) HasUnresolvedVariables(str string) bool {
	return strings.Contains(str, "$")
}
