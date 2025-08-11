package main

import (
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/wonderfulspam/gitlab-smith/pkg/analyzer"
	"github.com/wonderfulspam/gitlab-smith/pkg/parser"
	"github.com/spf13/cobra"
)

var analyzeCmd = &cobra.Command{
	Use:   "analyze [file]",
	Short: "Analyze GitLab CI configuration for issues and improvements",
	Long: `Analyze GitLab CI configuration files to identify potential issues,
optimization opportunities, and suggest improvements for better maintainability,
performance, security, and reliability.`,
	Args: cobra.ExactArgs(1),
	RunE: runAnalyze,
}

var analyzeFormat string

func init() {
	analyzeCmd.Flags().StringVar(&analyzeFormat, "format", "table", "Output format: table, json")
	rootCmd.AddCommand(analyzeCmd)
}

func runAnalyze(cmd *cobra.Command, args []string) error {
	configFile := args[0]

	// Make path absolute for cleaner display
	absPath, err := filepath.Abs(configFile)
	if err != nil {
		absPath = configFile
	}

	// Parse the GitLab CI configuration with includes
	config, err := parser.ParseFile(configFile)
	if err != nil {
		return fmt.Errorf("failed to parse GitLab CI config: %w", err)
	}

	// Run analysis
	result := analyzer.Analyze(config)

	switch analyzeFormat {
	case "json":
		return outputAnalysisJSON(cmd, result, absPath)
	case "table":
		return outputAnalysisTable(cmd, result, absPath)
	default:
		return fmt.Errorf("unsupported format: %s (supported: table, json)", analyzeFormat)
	}
}

func outputAnalysisJSON(cmd *cobra.Command, result *analyzer.AnalysisResult, filePath string) error {
	output := map[string]interface{}{
		"file":     filePath,
		"analysis": result,
	}

	encoder := json.NewEncoder(cmd.OutOrStdout())
	encoder.SetIndent("", "  ")
	return encoder.Encode(output)
}

func outputAnalysisTable(cmd *cobra.Command, result *analyzer.AnalysisResult, filePath string) error {
	out := cmd.OutOrStdout()

	fmt.Fprintf(out, "GitLab CI Analysis Report\n")
	fmt.Fprintf(out, "========================\n")
	fmt.Fprintf(out, "File: %s\n\n", filePath)

	// Summary
	fmt.Fprintf(out, "Summary\n")
	fmt.Fprintf(out, "-------\n")
	fmt.Fprintf(out, "Total Issues: %d\n", result.TotalIssues)
	fmt.Fprintf(out, "  Performance: %d\n", result.Summary.Performance)
	fmt.Fprintf(out, "  Security: %d\n", result.Summary.Security)
	fmt.Fprintf(out, "  Maintainability: %d\n", result.Summary.Maintainability)
	fmt.Fprintf(out, "  Reliability: %d\n", result.Summary.Reliability)
	fmt.Fprintf(out, "\n")

	if len(result.Issues) == 0 {
		fmt.Fprintf(out, "✅ No issues found! Your GitLab CI configuration looks good.\n")
		return nil
	}

	// Group issues by severity
	severityOrder := []analyzer.Severity{analyzer.SeverityHigh, analyzer.SeverityMedium, analyzer.SeverityLow}
	severityLabels := map[analyzer.Severity]string{
		analyzer.SeverityHigh:   "🔴 HIGH",
		analyzer.SeverityMedium: "🟡 MEDIUM",
		analyzer.SeverityLow:    "🟢 LOW",
	}

	for _, severity := range severityOrder {
		issues := result.FilterBySeverity(severity)
		if len(issues) == 0 {
			continue
		}

		fmt.Fprintf(out, "%s SEVERITY (%d issues)\n", severityLabels[severity], len(issues))
		fmt.Fprintf(out, "%s\n", getUnderline(len(severityLabels[severity])+12))

		for _, issue := range issues {
			fmt.Fprintf(out, "• [%s] %s\n", string(issue.Type), issue.Message)
			fmt.Fprintf(out, "  Path: %s\n", issue.Path)
			if issue.JobName != "" {
				fmt.Fprintf(out, "  Job: %s\n", issue.JobName)
			}
			if issue.Suggestion != "" {
				fmt.Fprintf(out, "  💡 %s\n", issue.Suggestion)
			}
			fmt.Fprintf(out, "\n")
		}
	}

	// Tips
	fmt.Fprintf(out, "💡 Tips\n")
	fmt.Fprintf(out, "-------\n")
	if result.Summary.Maintainability > 0 {
		fmt.Fprintf(out, "• Focus on maintainability improvements for long-term benefits\n")
	}
	if result.Summary.Performance > 0 {
		fmt.Fprintf(out, "• Address performance issues to speed up your pipelines\n")
	}
	if result.Summary.Security > 0 {
		fmt.Fprintf(out, "• Review security issues to protect your CI/CD pipeline\n")
	}
	fmt.Fprintf(out, "• Use 'gitlab-smith refactor' to validate configuration changes\n")

	return nil
}

func getUnderline(length int) string {
	underline := ""
	for i := 0; i < length; i++ {
		underline += "-"
	}
	return underline
}
