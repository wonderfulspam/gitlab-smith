package validator

import (
	"testing"

	"github.com/wonderfulspam/gitlab-smith/pkg/validator/testutil"
)

// Test scenarios - automatically discover scenarios from filesystem
func TestRefactoringScenarios(t *testing.T) {
	scenariosPath := "../../test/refactoring-scenarios"
	validator := NewRefactoringValidator()

	scenarios, err := testutil.DiscoverScenarios(scenariosPath)
	if err != nil {
		t.Fatalf("Failed to discover scenarios: %v", err)
	}

	if len(scenarios) == 0 {
		t.Skip("No scenarios found")
	}

	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			result, err := validator.CompareConfigurations(scenario.BeforeDir, scenario.AfterDir)
			if err != nil {
				t.Fatalf("Failed to analyze scenario %s: %v", scenario.Name, err)
			}

			// Validate against expectations
			success, issues, warnings := testutil.ValidateExpectations(convertResult(result), scenario.Expectations)

			// Report results
			t.Logf("Scenario: %s - %s", scenario.Name, scenario.Description)
			t.Logf("Success: %t", success)
			t.Logf("Analysis improvement: %d issues", result.AnalysisImprovement)
			if result.ActualChanges != nil {
				t.Logf("Improvement tags: %v", result.ActualChanges.ImprovementTags)
				t.Logf("Improvements count: %d", len(result.ActualChanges.Improvements))
			}
			if result.PipelineComparison != nil {
				t.Logf("Pipeline changes: %d total jobs, %d added, %d removed, improvement: %t",
					result.PipelineComparison.Summary.TotalJobs,
					result.PipelineComparison.Summary.AddedJobs,
					result.PipelineComparison.Summary.RemovedJobs,
					result.PipelineComparison.Summary.OverallImprovement)
			}

			// Report warnings
			for _, warning := range warnings {
				t.Logf("Warning: %s", warning)
			}

			// Fail if there are issues
			for _, issue := range issues {
				t.Logf("Issue: %s", issue)
			}

			if !success && scenario.Expectations.ShouldSucceed {
				t.Errorf("Scenario %s expected to succeed but failed", scenario.Name)
			}
		})
	}
}

// Test realistic application scenarios
func TestRealisticAppScenarios(t *testing.T) {
	scenariosPath := "../../test/realistic-app-scenarios"
	validator := NewRefactoringValidator()

	scenarios, err := testutil.DiscoverRealisticScenarios(scenariosPath)
	if err != nil {
		t.Fatalf("Failed to discover realistic scenarios: %v", err)
	}

	if len(scenarios) == 0 {
		t.Skip("No realistic scenarios found")
	}

	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			result, err := validator.CompareConfigurations(scenario.BeforeDir, scenario.AfterDir)
			if err != nil {
				t.Fatalf("Failed to analyze realistic scenario %s: %v", scenario.Name, err)
			}

			// Validate against expectations
			success, issues, warnings := testutil.ValidateExpectations(convertResult(result), scenario.Expectations)

			t.Logf("Realistic Scenario: %s - %s", scenario.Name, scenario.Description)
			t.Logf("Success: %t", success)
			t.Logf("Analysis improvement: %d issues", result.AnalysisImprovement)
			if result.PipelineComparison != nil {
				t.Logf("Pipeline changes: %d total jobs, %d added, %d removed, improvement: %t",
					result.PipelineComparison.Summary.TotalJobs,
					result.PipelineComparison.Summary.AddedJobs,
					result.PipelineComparison.Summary.RemovedJobs,
					result.PipelineComparison.Summary.OverallImprovement)
			}
			t.Logf("App context: realistic microservice pipeline structure")

			// Report warnings
			for _, warning := range warnings {
				t.Logf("Warning: %s", warning)
			}

			// Report issues
			for _, issue := range issues {
				t.Logf("Issue: %s", issue)
			}

			if !success && scenario.Expectations.ShouldSucceed {
				t.Errorf("Realistic scenario %s expected to succeed but failed", scenario.Name)
			}
		})
	}
}

// convertResult converts our internal RefactoringResult to the testutil version
// This avoids circular imports while allowing the testutil to work with our data
func convertResult(result *RefactoringResult) *testutil.RefactoringResult {
	return &testutil.RefactoringResult{
		ActualChanges:        result.ActualChanges,
		AnalysisImprovement:  result.AnalysisImprovement,
		PipelineComparison:   result.PipelineComparison,
		BehavioralValidation: result.BehavioralValidation,
	}
}

// Benchmark test to ensure refactoring doesn't introduce performance regressions
func BenchmarkRefactoringValidation(b *testing.B) {
	validator := NewRefactoringValidator()

	scenariosPath := "../../test/refactoring-scenarios"
	beforeDir := scenariosPath + "/scenario-1/before"
	afterDir := scenariosPath + "/scenario-1/after"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := validator.CompareConfigurations(beforeDir, afterDir)
		if err != nil {
			b.Fatalf("Validation failed: %v", err)
		}
	}
}
