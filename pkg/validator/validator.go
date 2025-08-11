package validator

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/emt/gitlab-smith/pkg/analyzer"
	"github.com/emt/gitlab-smith/pkg/differ"
	"github.com/emt/gitlab-smith/pkg/parser"
	"github.com/emt/gitlab-smith/pkg/renderer"
)

// RefactoringResult contains the validation results
type RefactoringResult struct {
	ActualChanges       *differ.DiffResult
	AnalysisImprovement int
	PipelineComparison  *renderer.PipelineComparison
}

// RefactoringValidator performs GitLab CI refactoring analysis
type RefactoringValidator struct{}

// NewRefactoringValidator creates a new refactoring validator
func NewRefactoringValidator() *RefactoringValidator {
	return &RefactoringValidator{}
}

// CompareConfigurations compares before and after GitLab CI configurations
func (rv *RefactoringValidator) CompareConfigurations(beforeDir, afterDir string) (*RefactoringResult, error) {
	// Parse before and after configurations
	beforeConfig, err := rv.parseConfiguration(beforeDir)
	if err != nil {
		return nil, fmt.Errorf("failed to parse before config: %w", err)
	}

	afterConfig, err := rv.parseConfiguration(afterDir)
	if err != nil {
		return nil, fmt.Errorf("failed to parse after config: %w", err)
	}

	result := &RefactoringResult{}

	// Perform semantic diff
	result.ActualChanges = differ.Compare(beforeConfig, afterConfig)

	// Analyze both configurations
	beforeAnalysis := analyzer.Analyze(beforeConfig)
	afterAnalysis := analyzer.Analyze(afterConfig)
	result.AnalysisImprovement = beforeAnalysis.TotalIssues - afterAnalysis.TotalIssues

	// Compare pipeline executions
	renderer := renderer.New(nil)
	pipelineComparison, err := renderer.CompareConfigurations(beforeConfig, afterConfig)
	if err != nil {
		return result, fmt.Errorf("pipeline comparison failed: %w", err)
	}
	result.PipelineComparison = pipelineComparison

	return result, nil
}

// parseConfiguration parses a GitLab CI configuration from a directory
func (rv *RefactoringValidator) parseConfiguration(configDir string) (*parser.GitLabConfig, error) {
	// Look for main CI file
	mainFiles := []string{".gitlab-ci.yml", ".gitlab-ci.yaml", "gitlab-ci.yml", "gitlab-ci.yaml"}

	var mainFile string
	for _, filename := range mainFiles {
		path := filepath.Join(configDir, filename)
		if _, err := os.Stat(path); err == nil {
			mainFile = path
			break
		}
	}

	if mainFile == "" {
		return nil, fmt.Errorf("no GitLab CI main file found in %s", configDir)
	}

	// Parse the configuration with includes
	config, err := parser.ParseFile(mainFile)
	if err != nil {
		return nil, fmt.Errorf("failed to parse configuration: %w", err)
	}

	return config, nil
}
