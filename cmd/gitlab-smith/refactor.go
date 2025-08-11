package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/wonderfulspam/gitlab-smith/pkg/analyzer"
	"github.com/wonderfulspam/gitlab-smith/pkg/analyzer/types"
	"github.com/wonderfulspam/gitlab-smith/pkg/deployer"
	"github.com/wonderfulspam/gitlab-smith/pkg/differ"
	"github.com/wonderfulspam/gitlab-smith/pkg/parser"
	"github.com/wonderfulspam/gitlab-smith/pkg/renderer"
)

var refactorCmd = &cobra.Command{
	Use:   "refactor --old <old-file> --new <new-file>",
	Short: "Compare two GitLab CI configurations and analyze differences",
	Long: `Performs semantic comparison between two GitLab CI configuration files.
Provides analysis of changes, potential issues, and optimization suggestions.`,
	RunE: runRefactor,
}

var (
	oldFile         string
	newFile         string
	outputFile      string
	analyze         bool
	fullTest        bool
	format          string
	pipelineCompare bool
)

func init() {
	refactorCmd.Flags().StringVar(&oldFile, "old", "", "Path to the old GitLab CI configuration file")
	refactorCmd.Flags().StringVar(&newFile, "new", "", "Path to the new GitLab CI configuration file")
	refactorCmd.Flags().StringVar(&outputFile, "output", "", "Output file for results (default: stdout)")
	refactorCmd.Flags().BoolVar(&analyze, "analyze", true, "Perform static analysis on both configurations")
	refactorCmd.Flags().BoolVar(&fullTest, "full-test", false, "Enable full testing mode with local GitLab deployment")
	refactorCmd.Flags().StringVar(&format, "format", "json", "Output format (json, table, dot, mermaid)")
	refactorCmd.Flags().BoolVar(&pipelineCompare, "pipeline-compare", false, "Enable pipeline execution comparison simulation")

	refactorCmd.MarkFlagRequired("old")
	refactorCmd.MarkFlagRequired("new")

	rootCmd.AddCommand(refactorCmd)
}

func runRefactor(cmd *cobra.Command, args []string) error {
	if fullTest {
		return runFullTestMode()
	}

	// Parse old configuration
	oldData, err := os.ReadFile(oldFile)
	if err != nil {
		return fmt.Errorf("reading old file '%s': %w", oldFile, err)
	}

	oldConfig, err := parser.Parse(oldData)
	if err != nil {
		return fmt.Errorf("parsing old GitLab CI config '%s': %w", oldFile, err)
	}

	// Parse new configuration
	newData, err := os.ReadFile(newFile)
	if err != nil {
		return fmt.Errorf("reading new file '%s': %w", newFile, err)
	}

	newConfig, err := parser.Parse(newData)
	if err != nil {
		return fmt.Errorf("parsing new GitLab CI config '%s': %w", newFile, err)
	}

	// Perform comparison
	diffResult := differ.Compare(oldConfig, newConfig)

	// Prepare result structure
	result := RefactorResult{
		Comparison: diffResult,
		Files: FileInfo{
			Old: oldFile,
			New: newFile,
		},
	}

	// Perform pipeline comparison if requested
	if pipelineCompare {
		fmt.Fprintf(os.Stderr, "🔄 Performing pipeline execution comparison...\n")
		r := renderer.New(nil)
		pipelineComparison, err := r.CompareConfigurations(oldConfig, newConfig)
		if err != nil {
			return fmt.Errorf("pipeline comparison failed: %w", err)
		}
		result.PipelineComparison = pipelineComparison
	}

	// Perform static analysis if requested
	if analyze {
		oldAnalysis := analyzer.Analyze(oldConfig)
		newAnalysis := analyzer.Analyze(newConfig)

		result.Analysis = &AnalysisComparison{
			Old: oldAnalysis,
			New: newAnalysis,
		}
	}

	// Generate output
	var output []byte
	switch format {
	case "json":
		output, err = json.MarshalIndent(result, "", "  ")
		if err != nil {
			return fmt.Errorf("marshaling result to JSON: %w", err)
		}
	case "table":
		output = []byte(formatAsTable(&result))
	case "dot", "mermaid":
		// For visual formats, we need to generate the appropriate diagram
		r := renderer.New(nil)
		var visualOutput string
		if pipelineCompare && result.PipelineComparison != nil {
			visualOutput, err = r.RenderVisualComparison(oldConfig, newConfig, result.PipelineComparison, format)
		} else {
			// Default to showing the new configuration structure
			visualOutput, err = r.RenderVisualPipeline(newConfig, format)
		}
		if err != nil {
			return fmt.Errorf("generating visual output: %w", err)
		}
		output = []byte(visualOutput)
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}

	// Write output
	if outputFile != "" {
		err = os.WriteFile(outputFile, output, 0644)
		if err != nil {
			return fmt.Errorf("writing output file: %w", err)
		}
		fmt.Printf("Results written to %s\n", outputFile)
	} else {
		fmt.Println(string(output))
	}

	// Print summary to stderr for visibility
	if diffResult.HasChanges {
		fmt.Fprintf(os.Stderr, "\n✓ Analysis complete: %s\n", diffResult.Summary)

		if result.Analysis != nil {
			oldIssues := result.Analysis.Old.TotalIssues
			newIssues := result.Analysis.New.TotalIssues
			issuesDelta := newIssues - oldIssues

			if issuesDelta > 0 {
				fmt.Fprintf(os.Stderr, "⚠  Static analysis: %d new issues introduced\n", issuesDelta)
			} else if issuesDelta < 0 {
				fmt.Fprintf(os.Stderr, "✓ Static analysis: %d issues resolved\n", -issuesDelta)
			} else {
				fmt.Fprintf(os.Stderr, "→ Static analysis: no change in issue count (%d issues)\n", newIssues)
			}
		}
	} else {
		fmt.Fprintln(os.Stderr, "\n✓ No semantic differences found")
	}

	return nil
}

type RefactorResult struct {
	Comparison         *differ.DiffResult           `json:"comparison"`
	Analysis           *AnalysisComparison          `json:"analysis,omitempty"`
	PipelineComparison *renderer.PipelineComparison `json:"pipeline_comparison,omitempty"`
	Files              FileInfo                     `json:"files"`
}

type AnalysisComparison struct {
	Old *types.AnalysisResult `json:"old"`
	New *types.AnalysisResult `json:"new"`
}

type FileInfo struct {
	Old string `json:"old"`
	New string `json:"new"`
}

func formatAsTable(result *RefactorResult) string {
	output := fmt.Sprintf("GitLab CI Configuration Comparison\n")
	output += fmt.Sprintf("=====================================\n\n")
	output += fmt.Sprintf("Files:\n")
	output += fmt.Sprintf("  Old: %s\n", result.Files.Old)
	output += fmt.Sprintf("  New: %s\n\n", result.Files.New)

	if result.Comparison.HasChanges {
		output += fmt.Sprintf("Summary: %s\n\n", result.Comparison.Summary)

		if len(result.Comparison.Semantic) > 0 {
			output += fmt.Sprintf("Semantic Changes:\n")
			output += fmt.Sprintf("-----------------\n")
			for _, diff := range result.Comparison.Semantic {
				output += fmt.Sprintf("  [%s] %s: %s\n", string(diff.Type), diff.Path, diff.Description)
			}
			output += "\n"
		}

		if len(result.Comparison.Dependencies) > 0 {
			output += fmt.Sprintf("Dependency Changes:\n")
			output += fmt.Sprintf("-------------------\n")
			for _, diff := range result.Comparison.Dependencies {
				output += fmt.Sprintf("  [%s] %s: %s\n", string(diff.Type), diff.Path, diff.Description)
			}
			output += "\n"
		}

		if len(result.Comparison.Performance) > 0 {
			output += fmt.Sprintf("Performance Changes:\n")
			output += fmt.Sprintf("--------------------\n")
			for _, diff := range result.Comparison.Performance {
				output += fmt.Sprintf("  [%s] %s: %s\n", string(diff.Type), diff.Path, diff.Description)
			}
			output += "\n"
		}
	} else {
		output += "No semantic differences found.\n\n"
	}

	if result.Analysis != nil {
		output += fmt.Sprintf("Static Analysis:\n")
		output += fmt.Sprintf("================\n")
		output += fmt.Sprintf("Old config issues: %d\n", result.Analysis.Old.TotalIssues)
		output += fmt.Sprintf("New config issues: %d\n", result.Analysis.New.TotalIssues)

		issuesDelta := result.Analysis.New.TotalIssues - result.Analysis.Old.TotalIssues
		if issuesDelta > 0 {
			output += fmt.Sprintf("Change: +%d issues\n", issuesDelta)
		} else if issuesDelta < 0 {
			output += fmt.Sprintf("Change: %d issues (improved)\n", issuesDelta)
		} else {
			output += "Change: no difference\n"
		}

		output += "\nNew Config Issues by Type:\n"
		output += fmt.Sprintf("  Performance: %d\n", result.Analysis.New.Summary.Performance)
		output += fmt.Sprintf("  Security: %d\n", result.Analysis.New.Summary.Security)
		output += fmt.Sprintf("  Maintainability: %d\n", result.Analysis.New.Summary.Maintainability)
		output += fmt.Sprintf("  Reliability: %d\n", result.Analysis.New.Summary.Reliability)
	}

	// Add pipeline comparison results if available
	if result.PipelineComparison != nil {
		output += "\n"
		r := renderer.New(nil)
		pipelineOutput, err := r.FormatComparison(result.PipelineComparison, "table")
		if err == nil {
			output += pipelineOutput
		} else {
			output += fmt.Sprintf("Error formatting pipeline comparison: %v\n", err)
		}
	}

	return output
}

func runFullTestMode() error {
	fmt.Println("🚀 Starting full testing mode with local GitLab deployment...")

	// Create deployer with default configuration
	deploy := deployer.New(nil)

	fmt.Println("📦 Checking Docker availability...")

	// Deploy GitLab instance
	fmt.Println("🔄 Deploying GitLab instance (this may take several minutes)...")
	if err := deploy.Deploy(); err != nil {
		return fmt.Errorf("failed to deploy GitLab: %w", err)
	}

	fmt.Println("✅ GitLab deployment successful!")

	// Get deployment status
	status, err := deploy.GetStatus()
	if err != nil {
		fmt.Printf("⚠️  Warning: Could not get deployment status: %v\n", err)
	} else {
		fmt.Printf("📍 GitLab URL: %s\n", status.URL)
		fmt.Printf("📊 Container Status: %s\n", status.ContainerStatus)
	}

	// Parse configurations for deployment
	fmt.Println("📋 Parsing GitLab CI configurations...")

	// Parse old configuration
	oldData, err := os.ReadFile(oldFile)
	if err != nil {
		deploy.Destroy() // Cleanup on error
		return fmt.Errorf("reading old file '%s': %w", oldFile, err)
	}

	oldConfig, err := parser.Parse(oldData)
	if err != nil {
		deploy.Destroy() // Cleanup on error
		return fmt.Errorf("parsing old GitLab CI config '%s': %w", oldFile, err)
	}

	// Parse new configuration
	newData, err := os.ReadFile(newFile)
	if err != nil {
		deploy.Destroy() // Cleanup on error
		return fmt.Errorf("reading new file '%s': %w", newFile, err)
	}

	newConfig, err := parser.Parse(newData)
	if err != nil {
		deploy.Destroy() // Cleanup on error
		return fmt.Errorf("parsing new GitLab CI config '%s': %w", newFile, err)
	}

	// Perform static comparison first
	fmt.Println("🔍 Performing semantic comparison...")
	diffResult := differ.Compare(oldConfig, newConfig)

	// Prepare full test result
	result := FullTestResult{
		Comparison: diffResult,
		Files: FileInfo{
			Old: oldFile,
			New: newFile,
		},
		Deployment: DeploymentInfo{
			URL:           status.URL,
			ContainerName: status.ContainerName,
			Status:        status.ContainerStatus,
		},
	}

	// Perform static analysis
	if analyze {
		fmt.Println("📊 Running static analysis...")
		oldAnalysis := analyzer.Analyze(oldConfig)
		newAnalysis := analyzer.Analyze(newConfig)

		result.Analysis = &AnalysisComparison{
			Old: oldAnalysis,
			New: newAnalysis,
		}
	}

	// Perform pipeline execution simulation
	fmt.Println("🎭 Running pipeline execution simulation...")
	r := renderer.New(nil) // GitLab client will be added later for real pipeline testing
	pipelineComparison, err := r.CompareConfigurations(oldConfig, newConfig)
	if err != nil {
		deploy.Destroy()
		return fmt.Errorf("pipeline comparison failed: %w", err)
	}
	result.PipelineComparison = pipelineComparison

	// Generate output
	var output []byte
	switch format {
	case "json":
		output, err = json.MarshalIndent(result, "", "  ")
		if err != nil {
			deploy.Destroy() // Cleanup on error
			return fmt.Errorf("marshaling result to JSON: %w", err)
		}
	case "table":
		output = []byte(formatFullTestAsTable(&result))
	case "dot", "mermaid":
		// For visual formats in full test mode, always show comparison since we have pipeline comparison
		r := renderer.New(nil)
		visualOutput, err := r.RenderVisualComparison(oldConfig, newConfig, result.PipelineComparison, format)
		if err != nil {
			deploy.Destroy() // Cleanup on error
			return fmt.Errorf("generating visual output: %w", err)
		}
		output = []byte(visualOutput)
	default:
		deploy.Destroy() // Cleanup on error
		return fmt.Errorf("unsupported format: %s", format)
	}

	// Write output
	if outputFile != "" {
		err = os.WriteFile(outputFile, output, 0644)
		if err != nil {
			deploy.Destroy() // Cleanup on error
			return fmt.Errorf("writing output file: %w", err)
		}
		fmt.Printf("📁 Results written to %s\n", outputFile)
	} else {
		fmt.Println(string(output))
	}

	// Ask user if they want to keep the deployment running
	fmt.Print("🤔 Keep GitLab deployment running for manual testing? (y/N): ")
	var response string
	fmt.Scanln(&response)

	if response == "y" || response == "Y" || response == "yes" {
		fmt.Printf("✨ GitLab is running at %s\n", status.URL)
		fmt.Println("💡 Use 'docker stop gitlab-smith-test' to stop when done")
		fmt.Println("💡 Default login: root / gitlabsmith123")
	} else {
		fmt.Println("🧹 Cleaning up GitLab deployment...")
		if err := deploy.Destroy(); err != nil {
			fmt.Printf("⚠️  Warning: Failed to destroy deployment: %v\n", err)
		} else {
			fmt.Println("✅ Cleanup complete")
		}
	}

	return nil
}

type FullTestResult struct {
	Comparison         *differ.DiffResult           `json:"comparison"`
	Analysis           *AnalysisComparison          `json:"analysis,omitempty"`
	PipelineComparison *renderer.PipelineComparison `json:"pipeline_comparison,omitempty"`
	Files              FileInfo                     `json:"files"`
	Deployment         DeploymentInfo               `json:"deployment"`
}

type DeploymentInfo struct {
	URL           string `json:"url"`
	ContainerName string `json:"container_name"`
	Status        string `json:"status"`
}

func formatFullTestAsTable(result *FullTestResult) string {
	output := fmt.Sprintf("GitLab CI Full Test Results\n")
	output += fmt.Sprintf("============================\n\n")

	output += fmt.Sprintf("Deployment Info:\n")
	output += fmt.Sprintf("  URL: %s\n", result.Deployment.URL)
	output += fmt.Sprintf("  Container: %s\n", result.Deployment.ContainerName)
	output += fmt.Sprintf("  Status: %s\n\n", result.Deployment.Status)

	output += fmt.Sprintf("Files:\n")
	output += fmt.Sprintf("  Old: %s\n", result.Files.Old)
	output += fmt.Sprintf("  New: %s\n\n", result.Files.New)

	// Reuse the existing table formatting logic for comparison
	refactorResult := &RefactorResult{
		Comparison:         result.Comparison,
		Analysis:           result.Analysis,
		PipelineComparison: result.PipelineComparison,
		Files:              result.Files,
	}

	output += formatAsTable(refactorResult)

	return output
}
