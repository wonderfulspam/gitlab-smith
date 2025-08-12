package main

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/spf13/cobra"
	"github.com/wonderfulspam/gitlab-smith/pkg/analyzer"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage GitLabSmith configuration",
	Long:  `Manage GitLabSmith configuration files, including initialization and validation.`,
}

var configInitCmd = &cobra.Command{
	Use:   "init [file]",
	Short: "Generate a default configuration file",
	Long: `Generate a default GitLabSmith configuration file with all available
checks and their default settings. If no file is specified, creates
.gitlab-smith.yml in the current directory.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runConfigInit,
}

var configValidateCmd = &cobra.Command{
	Use:   "validate [file]",
	Short: "Validate a configuration file",
	Long:  `Validate a GitLabSmith configuration file for correctness.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runConfigValidate,
}

var configListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all available checks",
	Long:  `List all available analysis checks with their types and descriptions.`,
	RunE:  runConfigList,
}

func init() {
	configCmd.AddCommand(configInitCmd)
	configCmd.AddCommand(configValidateCmd)
	configCmd.AddCommand(configListCmd)
	rootCmd.AddCommand(configCmd)
}

func runConfigInit(cmd *cobra.Command, args []string) error {
	outputFile := ".gitlab-smith.yml"
	if len(args) > 0 {
		outputFile = args[0]
	}

	// Check if file already exists
	if _, err := os.Stat(outputFile); err == nil {
		return fmt.Errorf("configuration file %s already exists", outputFile)
	}

	// Create example configuration content
	exampleConfig := `# GitLabSmith Configuration File
# This file configures analysis behavior, severity levels, and filtering rules

version: "1.0"

analyzer:
  # Global severity threshold - only report issues at or above this level
  # Options: low, medium, high
  severity_threshold: low

  # Global exclusions applied to all checks
  global_exclusions:
    # Paths to exclude from analysis
    paths:
      # - "experimental/*"
      # - "third-party/*"

    # Job name patterns to exclude
    jobs:
      # - "*-experimental"
      # - "sandbox-*"

# Configure individual checks
# Each check can be enabled/disabled, have severity overridden, and have custom exclusions
checks:
  job_naming:
    enabled: true
    type: maintainability
    description: "Checks job naming conventions"
    # Override the default severity for this check
    # severity: low

    # Ignore patterns - jobs matching these patterns won't be checked
    ignore_patterns:
      # - "legacy-*"
      # - "*-deprecated"

    # Specific exclusions
    exclusions:
      jobs:
        # - "my job with spaces"
        # - "another legacy job"

  image_tags:
    enabled: true
    type: security
    description: "Ensures Docker images use specific tags"
    # severity: high  # Elevate importance

    # Custom parameters for this check
    # custom_params:
    #   allowed_tags:
    #     - "latest"  # Sometimes needed for internal images
    #     - "stable"

  script_complexity:
    enabled: true
    type: maintainability
    description: "Detects overly complex job scripts"
    # custom_params:
    #   max_lines: 50  # Custom threshold
    #   max_commands: 20

  cache_usage:
    enabled: true
    type: performance
    description: "Checks for proper cache configuration in jobs"
    # custom_params:
    #   required_for_stages:
    #     - build
    #     - test

  # Add configurations for other checks as needed
  artifact_expiration:
    enabled: true
    type: performance
    description: "Ensures artifacts have expiration times set"

  dependency_chains:
    enabled: true
    type: performance
    description: "Detects overly long dependency chains"

  environment_variables:
    enabled: true
    type: security
    description: "Detects potential secrets in variable names"

  duplicated_code:
    enabled: true
    type: maintainability
    description: "Finds duplicate script blocks"

  retry_configuration:
    enabled: true
    type: reliability
    description: "Checks retry configuration for jobs"

# Differ configuration (for refactor command)
differ:
  # Ignore certain types of changes
  ignore_changes:
    # - variable_order  # Don't report reordered variables
    # - comment_changes  # Ignore comment-only changes

  # Treat certain changes as improvements
  improvement_patterns:
    # - consolidation  # Combining duplicate jobs
    # - simplification  # Reducing rule complexity

# Output configuration
output:
  format: table  # Options: table, json, yaml
  verbose: false
  show_suggestions: true
  # group_by: type  # Options: type, severity, job
`

	// Write the configuration file
	err := ioutil.WriteFile(outputFile, []byte(exampleConfig), 0644)
	if err != nil {
		return fmt.Errorf("failed to write configuration file: %w", err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "✅ Configuration file created: %s\n", outputFile)
	fmt.Fprintf(cmd.OutOrStdout(), "\nYou can now:\n")
	fmt.Fprintf(cmd.OutOrStdout(), "1. Edit the file to customize analysis behavior\n")
	fmt.Fprintf(cmd.OutOrStdout(), "2. Use it with: gitlab-smith analyze --config=%s <gitlab-ci.yml>\n", outputFile)
	fmt.Fprintf(cmd.OutOrStdout(), "3. Validate it with: gitlab-smith config validate %s\n", outputFile)

	return nil
}

func runConfigValidate(cmd *cobra.Command, args []string) error {
	configFile := args[0]

	// Try to load the configuration
	config, err := analyzer.LoadConfig(configFile)
	if err != nil {
		return fmt.Errorf("configuration validation failed: %w", err)
	}

	// Check for valid severity values
	validSeverities := map[string]bool{
		"":       true, // Empty is valid (no threshold)
		"low":    true,
		"medium": true,
		"high":   true,
	}

	if !validSeverities[string(config.Analyzer.SeverityThreshold)] {
		return fmt.Errorf("invalid severity threshold: %s (must be: low, medium, or high)", config.Analyzer.SeverityThreshold)
	}

	// Count enabled checks
	enabledCount := 0
	for _, check := range config.Checks {
		if check.Enabled {
			enabledCount++
		}
	}

	fmt.Fprintf(cmd.OutOrStdout(), "✅ Configuration is valid!\n\n")
	fmt.Fprintf(cmd.OutOrStdout(), "Summary:\n")
	fmt.Fprintf(cmd.OutOrStdout(), "  Version: %s\n", config.Version)
	fmt.Fprintf(cmd.OutOrStdout(), "  Severity Threshold: %s\n", config.Analyzer.SeverityThreshold)
	fmt.Fprintf(cmd.OutOrStdout(), "  Total Checks: %d\n", len(config.Checks))
	fmt.Fprintf(cmd.OutOrStdout(), "  Enabled Checks: %d\n", enabledCount)

	if len(config.Analyzer.GlobalExclusions.Jobs) > 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "  Global Job Exclusions: %d patterns\n", len(config.Analyzer.GlobalExclusions.Jobs))
	}
	if len(config.Analyzer.GlobalExclusions.Paths) > 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "  Global Path Exclusions: %d patterns\n", len(config.Analyzer.GlobalExclusions.Paths))
	}

	return nil
}

func runConfigList(cmd *cobra.Command, args []string) error {
	// Create a default analyzer to get all checks
	a := analyzer.New()
	config := a.GetConfig()

	fmt.Fprintf(cmd.OutOrStdout(), "Available GitLabSmith Checks\n")
	fmt.Fprintf(cmd.OutOrStdout(), "============================\n\n")

	// Group checks by type
	typeGroups := map[string][]string{
		"performance":     {},
		"security":        {},
		"maintainability": {},
		"reliability":     {},
	}

	for checkName, check := range config.Checks {
		typeGroups[string(check.Type)] = append(typeGroups[string(check.Type)], checkName)
	}

	// Display checks by type
	typeLabels := map[string]string{
		"performance":     "🚀 Performance",
		"security":        "🔒 Security",
		"maintainability": "🔧 Maintainability",
		"reliability":     "⚡ Reliability",
	}

	for typeName, label := range typeLabels {
		checks := typeGroups[typeName]
		if len(checks) == 0 {
			continue
		}

		fmt.Fprintf(cmd.OutOrStdout(), "%s Checks (%d)\n", label, len(checks))
		fmt.Fprintf(cmd.OutOrStdout(), "%s\n", getUnderline(len(label)+15))

		for _, checkName := range checks {
			check := config.Checks[checkName]
			status := "✅"
			if !check.Enabled {
				status = "❌"
			}
			fmt.Fprintf(cmd.OutOrStdout(), "%s %-30s %s\n", status, checkName, check.Description)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "\n")
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Use 'gitlab-smith config init' to create a configuration file\n")
	fmt.Fprintf(cmd.OutOrStdout(), "Use 'gitlab-smith analyze --disable-check=<name>' to disable specific checks\n")

	return nil
}
