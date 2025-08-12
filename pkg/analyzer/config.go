package analyzer

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"strings"

	"github.com/wonderfulspam/gitlab-smith/pkg/analyzer/types"
	"gopkg.in/yaml.v2"
)

// Config holds the overall analyzer configuration
type Config struct {
	Version  string                       `yaml:"version" json:"version"`
	Analyzer AnalyzerConfig               `yaml:"analyzer" json:"analyzer"`
	Checks   map[string]types.CheckConfig `yaml:"checks" json:"checks"`
	Differ   DifferConfig                 `yaml:"differ,omitempty" json:"differ,omitempty"`
	Output   OutputConfig                 `yaml:"output,omitempty" json:"output,omitempty"`
}

// AnalyzerConfig holds analyzer-specific configuration
type AnalyzerConfig struct {
	SeverityThreshold types.Severity   `yaml:"severity_threshold,omitempty" json:"severity_threshold,omitempty"`
	GlobalExclusions  GlobalExclusions `yaml:"global_exclusions,omitempty" json:"global_exclusions,omitempty"`
}

// GlobalExclusions defines global exclusion patterns
type GlobalExclusions struct {
	Paths []string `yaml:"paths,omitempty" json:"paths,omitempty"`
	Jobs  []string `yaml:"jobs,omitempty" json:"jobs,omitempty"`
}

// DifferConfig holds differ-specific configuration
type DifferConfig struct {
	IgnoreChanges       []string `yaml:"ignore_changes,omitempty" json:"ignore_changes,omitempty"`
	ImprovementPatterns []string `yaml:"improvement_patterns,omitempty" json:"improvement_patterns,omitempty"`
}

// OutputConfig holds output formatting configuration
type OutputConfig struct {
	Format          string `yaml:"format,omitempty" json:"format,omitempty"`
	Verbose         bool   `yaml:"verbose,omitempty" json:"verbose,omitempty"`
	ShowSuggestions bool   `yaml:"show_suggestions,omitempty" json:"show_suggestions,omitempty"`
	GroupBy         string `yaml:"group_by,omitempty" json:"group_by,omitempty"`
}

// DefaultConfig returns the default analyzer configuration
func DefaultConfig() *Config {
	return &Config{
		Version: "1.0",
		Analyzer: AnalyzerConfig{
			SeverityThreshold: types.SeverityLow,
		},
		Output: OutputConfig{
			Format:          "table",
			ShowSuggestions: true,
		},
		Checks: map[string]types.CheckConfig{
			// Performance checks
			"cache_usage": {
				Name:        "cache_usage",
				Type:        types.IssueTypePerformance,
				Enabled:     true,
				Description: "Checks for proper cache configuration in jobs",
			},
			"artifact_expiration": {
				Name:        "artifact_expiration",
				Type:        types.IssueTypePerformance,
				Enabled:     true,
				Description: "Ensures artifacts have expiration times set",
			},
			"dependency_chains": {
				Name:        "dependency_chains",
				Type:        types.IssueTypePerformance,
				Enabled:     true,
				Description: "Detects overly long dependency chains",
			},
			"unnecessary_dependencies": {
				Name:        "unnecessary_dependencies",
				Type:        types.IssueTypePerformance,
				Enabled:     true,
				Description: "Finds explicit dependencies that could be inferred",
			},
			"matrix_opportunities": {
				Name:        "matrix_opportunities",
				Type:        types.IssueTypePerformance,
				Enabled:     true,
				Description: "Identifies jobs that could benefit from parallel matrix",
			},
			"missing_needs": {
				Name:        "missing_needs",
				Type:        types.IssueTypePerformance,
				Enabled:     true,
				Description: "Suggests using 'needs' for better parallelization",
			},
			"workflow_optimization": {
				Name:        "workflow_optimization",
				Type:        types.IssueTypePerformance,
				Enabled:     true,
				Description: "Identifies workflow optimization opportunities",
			},

			// Security checks
			"image_tags": {
				Name:        "image_tags",
				Type:        types.IssueTypeSecurity,
				Enabled:     true,
				Description: "Ensures Docker images use specific tags",
			},
			"environment_variables": {
				Name:        "environment_variables",
				Type:        types.IssueTypeSecurity,
				Enabled:     true,
				Description: "Detects potential secrets in variable names",
			},

			// Maintainability checks
			"job_naming": {
				Name:        "job_naming",
				Type:        types.IssueTypeMaintainability,
				Enabled:     true,
				Description: "Checks job naming conventions",
			},
			"script_complexity": {
				Name:        "script_complexity",
				Type:        types.IssueTypeMaintainability,
				Enabled:     true,
				Description: "Detects overly complex job scripts",
			},
			"duplicated_code": {
				Name:        "duplicated_code",
				Type:        types.IssueTypeMaintainability,
				Enabled:     true,
				Description: "Finds duplicate script blocks",
			},
			"duplicated_before_scripts": {
				Name:        "duplicated_before_scripts",
				Type:        types.IssueTypeMaintainability,
				Enabled:     true,
				Description: "Detects duplicate before_script configurations",
			},
			"duplicated_cache_config": {
				Name:        "duplicated_cache_config",
				Type:        types.IssueTypeMaintainability,
				Enabled:     true,
				Description: "Finds duplicate cache configurations",
			},
			"duplicated_setup": {
				Name:        "duplicated_setup",
				Type:        types.IssueTypeMaintainability,
				Enabled:     true,
				Description: "Detects duplicate job setup patterns",
			},
			"duplicated_variables": {
				Name:        "duplicated_variables",
				Type:        types.IssueTypeMaintainability,
				Enabled:     true,
				Description: "Finds repeated variable definitions",
			},
			"verbose_rules": {
				Name:        "verbose_rules",
				Type:        types.IssueTypeMaintainability,
				Enabled:     true,
				Description: "Detects overly complex rules configurations",
			},
			"template_complexity": {
				Name:        "template_complexity",
				Type:        types.IssueTypeMaintainability,
				Enabled:     true,
				Description: "Finds overly complex template inheritance",
			},
			"redundant_inheritance": {
				Name:        "redundant_inheritance",
				Type:        types.IssueTypeMaintainability,
				Enabled:     true,
				Description: "Detects redundant code in inheritance chains",
			},
			"include_optimization": {
				Name:        "include_optimization",
				Type:        types.IssueTypeMaintainability,
				Enabled:     true,
				Description: "Suggests include optimization opportunities",
			},
			"external_include_duplication": {
				Name:        "external_include_duplication",
				Type:        types.IssueTypeMaintainability,
				Enabled:     true,
				Description: "Detects redundant external includes",
			},
			"missing_extends": {
				Name:        "missing_extends",
				Type:        types.IssueTypeMaintainability,
				Enabled:     true,
				Description: "Suggests opportunities for using extends/templates",
			},
			"missing_templates": {
				Name:        "missing_templates",
				Type:        types.IssueTypeMaintainability,
				Enabled:     true,
				Description: "Identifies configurations that would benefit from templates",
			},
			"stages_definition": {
				Name:        "stages_definition",
				Type:        types.IssueTypeMaintainability,
				Enabled:     true,
				Description: "Ensures stages are explicitly defined",
			},

			// Reliability checks
			"retry_configuration": {
				Name:        "retry_configuration",
				Type:        types.IssueTypeReliability,
				Enabled:     true,
				Description: "Checks retry configuration for jobs",
			},
			"missing_stages": {
				Name:        "missing_stages",
				Type:        types.IssueTypeReliability,
				Enabled:     true,
				Description: "Detects jobs referencing undefined stages",
			},
		},
	}
}

// LoadConfig loads analyzer configuration from a file
func LoadConfig(filename string) (*Config, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	config := &Config{}

	// Try YAML first, then JSON
	err = yaml.Unmarshal(data, config)
	if err != nil {
		// If YAML fails, try JSON
		err = json.Unmarshal(data, config)
		if err != nil {
			return nil, fmt.Errorf("failed to parse config file as YAML or JSON: %w", err)
		}
	}

	// Merge with defaults for any missing checks
	defaultConfig := DefaultConfig()
	for checkName, defaultCheck := range defaultConfig.Checks {
		if _, exists := config.Checks[checkName]; !exists {
			config.Checks[checkName] = defaultCheck
		}
	}

	return config, nil
}

// SaveConfig saves analyzer configuration to a file
func SaveConfig(config *Config, filename string) error {
	var data []byte
	var err error

	// Determine format based on file extension
	if filename[len(filename)-5:] == ".json" {
		data, err = json.MarshalIndent(config, "", "  ")
	} else {
		data, err = yaml.Marshal(config)
	}

	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	return ioutil.WriteFile(filename, data, 0644)
}

// LoadOrCreateConfig loads config from file or creates default if file doesn't exist
func LoadOrCreateConfig(filename string) (*Config, error) {
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		config := DefaultConfig()
		// Save the default config
		if saveErr := SaveConfig(config, filename); saveErr != nil {
			// If we can't save, just return the default config
			return config, nil
		}
		return config, nil
	}

	return LoadConfig(filename)
}

// IsCheckEnabled returns whether a specific check is enabled
func (c *Config) IsCheckEnabled(checkName string) bool {
	if check, exists := c.Checks[checkName]; exists {
		return check.Enabled
	}
	return false
}

// EnableCheck enables a specific check
func (c *Config) EnableCheck(checkName string) {
	if check, exists := c.Checks[checkName]; exists {
		check.Enabled = true
		c.Checks[checkName] = check
	}
}

// DisableCheck disables a specific check
func (c *Config) DisableCheck(checkName string) {
	if check, exists := c.Checks[checkName]; exists {
		check.Enabled = false
		c.Checks[checkName] = check
	}
}

// GetEnabledChecks returns a list of all enabled check names
func (c *Config) GetEnabledChecks() []string {
	var enabled []string
	for checkName, check := range c.Checks {
		if check.Enabled {
			enabled = append(enabled, checkName)
		}
	}
	return enabled
}

// GetChecksByType returns enabled checks of a specific type
func (c *Config) GetChecksByType(issueType types.IssueType) []string {
	var checks []string
	for checkName, check := range c.Checks {
		if check.Enabled && check.Type == issueType {
			checks = append(checks, checkName)
		}
	}
	return checks
}

// ShouldSkipJob checks if a job should be excluded based on configuration
func (c *Config) ShouldSkipJob(checkName, jobName string) bool {
	// Check global exclusions first
	if c.Analyzer.GlobalExclusions.Jobs != nil {
		for _, pattern := range c.Analyzer.GlobalExclusions.Jobs {
			if matched, _ := matchPattern(pattern, jobName); matched {
				return true
			}
		}
	}

	// Check check-specific exclusions
	if check, exists := c.Checks[checkName]; exists {
		// Check ignore patterns
		for _, pattern := range check.IgnorePatterns {
			if matched, _ := matchPattern(pattern, jobName); matched {
				return true
			}
		}

		// Check specific job exclusions
		for _, excludedJob := range check.Exclusions.Jobs {
			if excludedJob == jobName {
				return true
			}
		}
	}

	return false
}

// ShouldSkipPath checks if a path should be excluded based on configuration
func (c *Config) ShouldSkipPath(checkName, path string) bool {
	// Check global exclusions first
	if c.Analyzer.GlobalExclusions.Paths != nil {
		for _, pattern := range c.Analyzer.GlobalExclusions.Paths {
			if matched, _ := matchPattern(pattern, path); matched {
				return true
			}
		}
	}

	// Check check-specific exclusions
	if check, exists := c.Checks[checkName]; exists {
		for _, excludedPath := range check.Exclusions.Paths {
			if matched, _ := matchPattern(excludedPath, path); matched {
				return true
			}
		}
	}

	return false
}

// GetCheckSeverity returns the configured severity for a check (or default)
func (c *Config) GetCheckSeverity(checkName string, defaultSeverity types.Severity) types.Severity {
	if check, exists := c.Checks[checkName]; exists && check.Severity != "" {
		return check.Severity
	}
	return defaultSeverity
}

// ShouldReportIssue checks if an issue meets the severity threshold
func (c *Config) ShouldReportIssue(severity types.Severity) bool {
	if c.Analyzer.SeverityThreshold == "" {
		return true // No threshold set, report all
	}

	return severityLevel(severity) >= severityLevel(c.Analyzer.SeverityThreshold)
}

// GetCustomParam retrieves a custom parameter for a check
func (c *Config) GetCustomParam(checkName, paramName string, defaultValue interface{}) interface{} {
	if check, exists := c.Checks[checkName]; exists {
		if value, found := check.CustomParams[paramName]; found {
			return value
		}
	}
	return defaultValue
}

// Helper function to convert severity to a comparable level
func severityLevel(s types.Severity) int {
	switch s {
	case types.SeverityLow:
		return 1
	case types.SeverityMedium:
		return 2
	case types.SeverityHigh:
		return 3
	default:
		return 0
	}
}

// Helper function for simple glob pattern matching
func matchPattern(pattern, str string) (bool, error) {
	// Simple implementation - can be enhanced with proper glob library
	if strings.Contains(pattern, "*") {
		// Convert simple glob to regex
		regexPattern := strings.ReplaceAll(pattern, "*", ".*")
		regexPattern = "^" + regexPattern + "$"
		return regexp.MatchString(regexPattern, str)
	}
	return pattern == str, nil
}
