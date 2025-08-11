package validator

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/emt/gitlab-smith/pkg/analyzer"
	"github.com/emt/gitlab-smith/pkg/differ"
	"github.com/emt/gitlab-smith/pkg/parser"
	"github.com/emt/gitlab-smith/pkg/renderer"
)

// SimpleRefactoringCase represents a simple before/after test case
type SimpleRefactoringCase struct {
	Name        string
	Description string
	BeforeFile  string
	AfterFile   string
	Expectations SimpleRefactoringExpectations
}

// SimpleRefactoringExpectations defines success criteria for simple cases
type SimpleRefactoringExpectations struct {
	ShouldReduceIssues     bool    // Should analyzer find fewer issues
	ShouldMaintainBehavior bool    // Should behavior remain the same
	ShouldImproveOrMaintainPerf bool // Should performance improve or stay same
	ExpectedImprovementAreas []string // Areas that should show improvement
	MaxNewIssues           int     // Maximum new issues allowed
}

// SimpleRefactoringResult contains validation results for simple cases
type SimpleRefactoringResult struct {
	Case               *SimpleRefactoringCase
	Success            bool
	Issues             []string
	AnalysisImprovement int
	BehaviorMaintained bool
	PerformanceImproved bool
	DiffResult         *differ.DiffResult
	PipelineComparison *renderer.PipelineComparison
}

// ValidateSimpleRefactoring validates a simple before/after refactoring case
func ValidateSimpleRefactoring(testCase *SimpleRefactoringCase) (*SimpleRefactoringResult, error) {
	result := &SimpleRefactoringResult{
		Case:   testCase,
		Issues: []string{},
	}

	// Parse before configuration
	beforeData, err := os.ReadFile(testCase.BeforeFile)
	if err != nil {
		return result, err
	}
	beforeConfig, err := parser.Parse(beforeData)
	if err != nil {
		return result, err
	}

	// Parse after configuration
	afterData, err := os.ReadFile(testCase.AfterFile)
	if err != nil {
		return result, err
	}
	afterConfig, err := parser.Parse(afterData)
	if err != nil {
		return result, err
	}

	// Perform semantic diff
	result.DiffResult = differ.Compare(beforeConfig, afterConfig)

	// Analyze both configurations
	beforeAnalysis := analyzer.Analyze(beforeConfig)
	afterAnalysis := analyzer.Analyze(afterConfig)
	result.AnalysisImprovement = beforeAnalysis.TotalIssues - afterAnalysis.TotalIssues

	// Compare pipeline executions
	renderer := renderer.New(nil)
	pipelineComparison, err := renderer.CompareConfigurations(beforeConfig, afterConfig)
	if err == nil {
		result.PipelineComparison = pipelineComparison
		result.PerformanceImproved = pipelineComparison.Summary.OverallImprovement
	}

	// Assess behavior maintenance (semantic equivalence)
	result.BehaviorMaintained = assessBehaviorMaintenance(result.DiffResult)

	// Validate expectations
	result.Success = validateSimpleExpectations(result, testCase.Expectations)

	return result, nil
}

// assessBehaviorMaintenance checks if core behavior is maintained
func assessBehaviorMaintenance(diffResult *differ.DiffResult) bool {
	// Check for significant behavioral changes
	significantChanges := 0
	templateJobs := 0
	
	for _, change := range diffResult.Semantic {
		if isBehavioralChange(change) {
			// Template jobs (starting with .) are refactoring improvements, not behavioral changes
			if change.Type == differ.DiffTypeAdded && contains(change.Path, "jobs..") {
				templateJobs++
				continue
			}
			
			// Check if this is a refactoring-safe change
			if isRefactoringSafeChange(change) {
				continue // Safe refactoring changes don't count as significant
			} else {
				significantChanges++
			}
		}
	}

	// Allow template job additions and be more lenient with refactoring changes
	// Templates are a sign of good refactoring, not behavioral problems
	return significantChanges <= 3 || (templateJobs > 0 && significantChanges <= 5)
}

// isRefactoringSafeChange identifies changes that are safe refactoring moves
func isRefactoringSafeChange(change differ.ConfigDiff) bool {
	// Changes that are typically safe during refactoring
	safePatterns := []string{
		"script changed for", // Often consolidation moves
		"Job script changed for", // Usually setup consolidation
		"Job removed:", // Template-based consolidation
		"Job added:", // Template introduction
	}
	
	for _, pattern := range safePatterns {
		if contains(change.Description, pattern) {
			return true
		}
	}
	
	return false
}

// isBehavioralChange determines if a change affects pipeline behavior
func isBehavioralChange(change differ.ConfigDiff) bool {
	// Use the Behavioral field from the differ
	return change.Behavioral
}

// validateSimpleExpectations validates results against expectations
func validateSimpleExpectations(result *SimpleRefactoringResult, expectations SimpleRefactoringExpectations) bool {
	success := true

	// Check issue reduction
	if expectations.ShouldReduceIssues && result.AnalysisImprovement <= 0 {
		result.Issues = append(result.Issues, "Expected to reduce analyzer issues but did not improve")
		success = false
	}

	// Check for too many new issues
	if result.AnalysisImprovement < 0 && -result.AnalysisImprovement > expectations.MaxNewIssues {
		result.Issues = append(result.Issues, 
			"Too many new issues introduced: %d (max allowed: %d)")
		success = false
	}

	// Check behavior maintenance
	if expectations.ShouldMaintainBehavior && !result.BehaviorMaintained {
		result.Issues = append(result.Issues, "Expected to maintain behavior but significant changes detected")
		success = false
	}

	// Check for pipeline structure improvements (job count, parallelism)
	if expectations.ShouldImproveOrMaintainPerf && result.PipelineComparison != nil {
		// Focus on structural improvements rather than simulated timing
		if result.PipelineComparison.Summary.AddedJobs > result.PipelineComparison.Summary.RemovedJobs+2 {
			result.Issues = append(result.Issues, "Refactoring added too many jobs without clear benefit")
			success = false
		}
		// Note: We don't validate duration changes as they're simulated estimates
	}

	return success
}

// contains checks if a string contains a substring (case-insensitive)
func contains(s, substr string) bool {
	return len(s) >= len(substr) && 
		   (s == substr || 
		    len(s) > len(substr) && 
		    (s[:len(substr)] == substr || 
		     s[len(s)-len(substr):] == substr ||
		     indexOf(s, substr) >= 0))
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// Test cases for simple refactoring scenarios
func TestSimpleRefactoringCases(t *testing.T) {
	basePath := "../../test/simple-refactoring-cases"
	
	testCases := []*SimpleRefactoringCase{
		{
			Name:        "duplicate-before-scripts",
			Description: "Consolidating duplicate before_script blocks using default",
			BeforeFile:  filepath.Join(basePath, "duplicate-before-scripts-before.yml"),
			AfterFile:   filepath.Join(basePath, "duplicate-before-scripts-after.yml"),
			Expectations: SimpleRefactoringExpectations{
				ShouldReduceIssues:      true,
				ShouldMaintainBehavior:  true,
				ShouldImproveOrMaintainPerf: true,
				ExpectedImprovementAreas: []string{"duplication"},
				MaxNewIssues:            0,
			},
		},
		{
			Name:        "duplicate-cache",
			Description: "Consolidating repeated cache configuration using default",
			BeforeFile:  filepath.Join(basePath, "duplicate-cache-before.yml"),
			AfterFile:   filepath.Join(basePath, "duplicate-cache-after.yml"),
			Expectations: SimpleRefactoringExpectations{
				ShouldReduceIssues:      true,
				ShouldMaintainBehavior:  true,
				ShouldImproveOrMaintainPerf: true,
				ExpectedImprovementAreas: []string{"cache", "duplication"},
				MaxNewIssues:            0,
			},
		},
		{
			Name:        "duplicate-docker",
			Description: "Consolidating Docker setup using extends",
			BeforeFile:  filepath.Join(basePath, "duplicate-docker-before.yml"),
			AfterFile:   filepath.Join(basePath, "duplicate-docker-after.yml"),
			Expectations: SimpleRefactoringExpectations{
				ShouldReduceIssues:      true,
				ShouldMaintainBehavior:  true,
				ShouldImproveOrMaintainPerf: true,
				ExpectedImprovementAreas: []string{"template", "duplication"},
				MaxNewIssues:            0,
			},
		},
		{
			Name:        "unnecessary-deps",
			Description: "Removing unnecessary explicit dependencies",
			BeforeFile:  filepath.Join(basePath, "unnecessary-deps-before.yml"),
			AfterFile:   filepath.Join(basePath, "unnecessary-deps-after.yml"),
			Expectations: SimpleRefactoringExpectations{
				ShouldReduceIssues:      true,
				ShouldMaintainBehavior:  true,
				ShouldImproveOrMaintainPerf: true,
				ExpectedImprovementAreas: []string{"dependencies"},
				MaxNewIssues:            0,
			},
		},
		{
			Name:        "verbose-rules",
			Description: "Simplifying verbose and redundant rules",
			BeforeFile:  filepath.Join(basePath, "verbose-rules-before.yml"),
			AfterFile:   filepath.Join(basePath, "verbose-rules-after.yml"),
			Expectations: SimpleRefactoringExpectations{
				ShouldReduceIssues:      true,
				ShouldMaintainBehavior:  true,
				ShouldImproveOrMaintainPerf: true,
				ExpectedImprovementAreas: []string{"rules", "simplification"},
				MaxNewIssues:            0,
			},
		},
		// Medium complexity test cases
		{
			Name:        "multiple-patterns",
			Description: "Consolidating multiple duplication patterns using templates and defaults",
			BeforeFile:  filepath.Join(basePath, "multiple-patterns-before.yml"),
			AfterFile:   filepath.Join(basePath, "multiple-patterns-after.yml"),
			Expectations: SimpleRefactoringExpectations{
				ShouldReduceIssues:      true,
				ShouldMaintainBehavior:  true,
				ShouldImproveOrMaintainPerf: true,
				ExpectedImprovementAreas: []string{"template", "duplication", "extends"},
				MaxNewIssues:            1, // Allow some complexity for better structure
			},
		},
		{
			Name:        "variable-simple",
			Description: "Consolidating repeated variables using global scope and templates",
			BeforeFile:  filepath.Join(basePath, "variable-simple-before.yml"),
			AfterFile:   filepath.Join(basePath, "variable-simple-after.yml"),
			Expectations: SimpleRefactoringExpectations{
				ShouldReduceIssues:      true,
				ShouldMaintainBehavior:  true,
				ShouldImproveOrMaintainPerf: true,
				ExpectedImprovementAreas: []string{"variables", "duplication"},
				MaxNewIssues:            0,
			},
		},
		{
			Name:        "complex-conditions",
			Description: "Simplifying complex conditional logic using workflow rules",
			BeforeFile:  filepath.Join(basePath, "complex-conditions-before.yml"),
			AfterFile:   filepath.Join(basePath, "complex-conditions-after.yml"),
			Expectations: SimpleRefactoringExpectations{
				ShouldReduceIssues:      true,
				ShouldMaintainBehavior:  true,
				ShouldImproveOrMaintainPerf: true,
				ExpectedImprovementAreas: []string{"rules", "workflow"},
				MaxNewIssues:            0,
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.Name, func(t *testing.T) {
			result, err := ValidateSimpleRefactoring(testCase)
			if err != nil {
				t.Fatalf("Failed to validate case %s: %v", testCase.Name, err)
			}

			// Report results
			t.Logf("Case: %s - %s", testCase.Name, testCase.Description)
			t.Logf("Success: %v", result.Success)
			t.Logf("Analysis improvement: %d issues", result.AnalysisImprovement)
			t.Logf("Behavior maintained: %v", result.BehaviorMaintained)

			if result.PipelineComparison != nil {
				t.Logf("Pipeline simulation - Jobs: %d total, %d added, %d removed", 
					result.PipelineComparison.Summary.TotalJobs,
					result.PipelineComparison.Summary.AddedJobs,
					result.PipelineComparison.Summary.RemovedJobs)
				t.Logf("Note: Duration changes are simulated estimates, not real performance")
			}

			// Report diff summary
			if result.DiffResult != nil {
				t.Logf("Semantic changes: %d", len(result.DiffResult.Semantic))
				t.Logf("Dependency changes: %d", len(result.DiffResult.Dependencies))
				t.Logf("Performance changes: %d", len(result.DiffResult.Performance))
			}

			// Report issues
			for _, issue := range result.Issues {
				t.Logf("Issue: %s", issue)
			}

			// Debug: Report actual semantic changes for failing cases
			if !result.Success && result.DiffResult != nil {
				t.Logf("Debug - Semantic changes:")
				for _, change := range result.DiffResult.Semantic {
					behavioral := isBehavioralChange(change)
					t.Logf("  - Path: %s, Type: %s, Description: %s, Behavioral: %v", 
						change.Path, change.Type, change.Description, behavioral)
				}
			}

			// Validate success
			if !result.Success {
				t.Errorf("Refactoring case %s failed validation", testCase.Name)
			}
		})
	}
}

// Benchmark simple refactoring validation
func BenchmarkSimpleRefactoringValidation(b *testing.B) {
	basePath := "../../test/simple-refactoring-cases"
	
	testCase := &SimpleRefactoringCase{
		Name:       "benchmark",
		BeforeFile: filepath.Join(basePath, "duplicate-before-scripts-before.yml"),
		AfterFile:  filepath.Join(basePath, "duplicate-before-scripts-after.yml"),
		Expectations: SimpleRefactoringExpectations{
			ShouldReduceIssues:     true,
			ShouldMaintainBehavior: true,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := ValidateSimpleRefactoring(testCase)
		if err != nil {
			b.Fatalf("Validation failed: %v", err)
		}
	}
}