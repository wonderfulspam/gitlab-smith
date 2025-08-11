package validator

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"gopkg.in/yaml.v3"

	"github.com/emt/gitlab-smith/pkg/differ"
	"github.com/emt/gitlab-smith/pkg/renderer"
)

// RefactoringScenario represents a complete refactoring test case
type RefactoringScenario struct {
	Name        string
	Description string
	BeforeDir   string
	AfterDir    string
	IncludesDir string
	Expectations RefactoringExpectations
}

// RefactoringExpectations defines what success looks like for a refactoring
type RefactoringExpectations struct {
	ShouldSucceed          bool                    // Whether the refactor should be considered successful
	ExpectedIssueReduction int                     // Expected reduction in analyzer issues
	MaxAllowedNewIssues    int                     // Maximum new issues that are acceptable
	RequiredImprovements   []string                // Required improvement categories
	ForbiddenChanges       []string                // Changes that should not happen
	SemanticEquivalence    bool                    // Whether pipelines should be semantically equivalent
	PerformanceImprovement bool                    // Whether performance should improve
	ExpectedJobChanges     map[string]JobChangeType // Expected changes per job
	
	// Detailed expectations
	ExpectedIssueTypes     map[string]int          // Expected count per issue type
	ExpectedIssuePatterns  []string                // Expected issue patterns/messages
	MinimumJobsAnalyzed    int                     // Minimum jobs that should be parsed
	ExpectedIncludes       int                     // Expected includes (for include scenarios)
}

type JobChangeType string

const (
	JobAdded      JobChangeType = "added"
	JobRemoved    JobChangeType = "removed"
	JobUnchanged  JobChangeType = "unchanged" 
	JobImproved   JobChangeType = "improved"
	JobRenamed    JobChangeType = "renamed"
)

// ScenarioConfig represents scenario configuration that can be loaded from YAML
type ScenarioConfig struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	Expectations struct {
		ShouldSucceed          bool                       `yaml:"should_succeed"`
		ExpectedIssueReduction int                        `yaml:"expected_issue_reduction"`
		MaxAllowedNewIssues    int                        `yaml:"max_allowed_new_issues"`
		RequiredImprovements   []string                   `yaml:"required_improvements"`
		ForbiddenChanges       []string                   `yaml:"forbidden_changes"`
		SemanticEquivalence    bool                       `yaml:"semantic_equivalence"`
		PerformanceImprovement bool                       `yaml:"performance_improvement"`
		ExpectedJobChanges     map[string]JobChangeType   `yaml:"expected_job_changes"`
		
		// Detailed expectations for specific improvement types
		ExpectedIssueTypes     map[string]int             `yaml:"expected_issue_types"`    // e.g., "maintainability": 5
		ExpectedIssuePatterns  []string                   `yaml:"expected_issue_patterns"` // e.g., "template complexity", "matrix opportunities"
		MinimumJobsAnalyzed    int                        `yaml:"minimum_jobs_analyzed"`   // Ensure parser is working
		ExpectedIncludes       int                        `yaml:"expected_includes"`       // For include consolidation tests
	} `yaml:"expectations"`
}

// Test scenarios - automatically discover scenarios from filesystem
func TestRefactoringScenarios(t *testing.T) {
	scenariosPath := "../../test/refactoring-scenarios"
	validator := NewRefactoringValidator()

	scenarios, err := discoverScenarios(scenariosPath)
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
			success, issues, warnings := validateExpectations(result, scenario.Expectations)

			// Report results
			t.Logf("Scenario: %s - %s", scenario.Name, scenario.Description)
			t.Logf("Success: %v", success)
			t.Logf("Analysis improvement: %d issues", result.AnalysisImprovement)
			t.Logf("Improvement tags: %v", result.ActualChanges.ImprovementTags)
			t.Logf("Improvements count: %d", len(result.ActualChanges.Improvements))
			
			if result.PipelineComparison != nil {
				t.Logf("Pipeline changes: %d total jobs, %d added, %d removed, improvement: %v",
					result.PipelineComparison.Summary.TotalJobs,
					result.PipelineComparison.Summary.AddedJobs,
					result.PipelineComparison.Summary.RemovedJobs,
					result.PipelineComparison.Summary.OverallImprovement)
			}

			// Report issues and warnings
			for _, issue := range issues {
				t.Logf("Issue: %s", issue)
			}
			for _, warning := range warnings {
				t.Logf("Warning: %s", warning)
			}

			// Fail test if scenario expected to succeed but didn't
			if scenario.Expectations.ShouldSucceed && !success {
				t.Errorf("Scenario %s expected to succeed but failed", scenario.Name)
			}
			
			// Pass test if scenario expected to fail and did fail
			if !scenario.Expectations.ShouldSucceed && success {
				t.Errorf("Scenario %s expected to fail but succeeded", scenario.Name)
			}
		})
	}
}

// Test realistic app scenarios - handle more complex directory structures
func TestRealisticAppScenarios(t *testing.T) {
	scenariosPath := "../../test/realistic-app-scenarios"
	validator := NewRefactoringValidator()

	scenarios, err := discoverRealisticScenarios(scenariosPath)
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
			success, issues, warnings := validateExpectations(result, scenario.Expectations)

			// Report results with focus on realistic app metrics
			t.Logf("Realistic Scenario: %s - %s", scenario.Name, scenario.Description)
			t.Logf("Success: %v", success)
			t.Logf("Analysis improvement: %d issues", result.AnalysisImprovement)
			
			if result.PipelineComparison != nil {
				t.Logf("Pipeline changes: %d total jobs, %d added, %d removed, improvement: %v",
					result.PipelineComparison.Summary.TotalJobs,
					result.PipelineComparison.Summary.AddedJobs,
					result.PipelineComparison.Summary.RemovedJobs,
					result.PipelineComparison.Summary.OverallImprovement)
				t.Logf("App context: realistic microservice pipeline structure")
			}

			// Report issues and warnings
			for _, issue := range issues {
				t.Logf("Issue: %s", issue)
			}
			for _, warning := range warnings {
				t.Logf("Warning: %s", warning)
			}

			// Validate expectations
			if scenario.Expectations.ShouldSucceed && !success {
				t.Errorf("Realistic scenario %s expected to succeed but failed", scenario.Name)
			}
			
			if !scenario.Expectations.ShouldSucceed && success {
				t.Errorf("Realistic scenario %s expected to fail but succeeded", scenario.Name)
			}
		})
	}
}

// validateExpectations validates the refactoring result against expectations (test logic)
func validateExpectations(result *RefactoringResult, expectations RefactoringExpectations) (bool, []string, []string) {
	var issues []string
	var warnings []string
	success := true

	// Check issue reduction
	if result.AnalysisImprovement < expectations.ExpectedIssueReduction {
		issues = append(issues, 
			fmt.Sprintf("Expected issue reduction of %d, got %d", 
				expectations.ExpectedIssueReduction, result.AnalysisImprovement))
		success = false
	}

	// Check for too many new issues
	if result.AnalysisImprovement < 0 && -result.AnalysisImprovement > expectations.MaxAllowedNewIssues {
		issues = append(issues,
			fmt.Sprintf("Too many new issues introduced: %d (max allowed: %d)",
				-result.AnalysisImprovement, expectations.MaxAllowedNewIssues))
		success = false
	}

	// Check semantic equivalence
	if expectations.SemanticEquivalence && !isSemanticallySimilar(result) {
		issues = append(issues, "Configurations are not semantically equivalent")
		success = false
	}

	// Check performance improvement
	if expectations.PerformanceImprovement && result.PipelineComparison != nil {
		if !result.PipelineComparison.Summary.OverallImprovement {
			issues = append(issues, "Expected performance improvement but got degradation")
			success = false
		}
	}

	// Check forbidden changes
	for _, forbidden := range expectations.ForbiddenChanges {
		if containsChange(result, forbidden) {
			issues = append(issues, fmt.Sprintf("Forbidden change detected: %s", forbidden))
			success = false
		}
	}

	// Check required improvements
	for _, required := range expectations.RequiredImprovements {
		if !containsChange(result, required) {
			issues = append(issues, fmt.Sprintf("Required improvement missing: %s", required))
			success = false
		}
	}

	// Check job changes
	if result.PipelineComparison != nil {
		for jobName, expectedChange := range expectations.ExpectedJobChanges {
			actualChange := getJobChangeType(result.PipelineComparison, jobName)
			if actualChange != expectedChange {
				issues = append(issues,
					fmt.Sprintf("Job %s: expected %s, got %s", jobName, expectedChange, actualChange))
				success = false
			}
		}
	}

	// Check minimum jobs analyzed
	if expectations.MinimumJobsAnalyzed > 0 && result.PipelineComparison != nil {
		if result.PipelineComparison.Summary.TotalJobs < expectations.MinimumJobsAnalyzed {
			issues = append(issues,
				fmt.Sprintf("Expected at least %d jobs analyzed, got %d", 
					expectations.MinimumJobsAnalyzed, result.PipelineComparison.Summary.TotalJobs))
			success = false
		}
	}

	// Check expected issue patterns
	if len(expectations.ExpectedIssuePatterns) > 0 {
		for _, pattern := range expectations.ExpectedIssuePatterns {
			if !containsChange(result, pattern) {
				warnings = append(warnings,
					fmt.Sprintf("Expected issue pattern '%s' not found in analysis", pattern))
			}
		}
	}

	return success, issues, warnings
}

// isSemanticallySimilar checks if two configurations are semantically similar (test logic)
func isSemanticallySimilar(result *RefactoringResult) bool {
	significantChanges := 0
	
	for _, change := range result.ActualChanges.Semantic {
		if isSignificantChange(change) {
			significantChanges++
		}
	}

	// Be more lenient if there are improvement patterns detected
	maxChanges := 2
	if len(result.ActualChanges.ImprovementTags) > 0 {
		maxChanges = 5 // Allow more changes for good refactoring
	}

	return significantChanges <= maxChanges
}

// isSignificantChange determines if a change affects pipeline behavior (test logic)
func isSignificantChange(change differ.ConfigDiff) bool {
	// Use the Behavioral field from the differ
	return change.Behavioral
}

// containsChange checks if the diff contains a specific type of change (test logic)
func containsChange(result *RefactoringResult, changePattern string) bool {
	allChanges := append(result.ActualChanges.Semantic, result.ActualChanges.Dependencies...)
	allChanges = append(allChanges, result.ActualChanges.Performance...)
	allChanges = append(allChanges, result.ActualChanges.Improvements...)

	pattern := strings.ToLower(changePattern)

	// Check improvement tags directly
	for _, tag := range result.ActualChanges.ImprovementTags {
		if strings.ToLower(tag) == pattern {
			return true
		}
	}
	
	for _, change := range allChanges {
		path := strings.ToLower(change.Path)
		desc := strings.ToLower(change.Description)
		
		if strings.Contains(path, pattern) || strings.Contains(desc, pattern) {
			return true
		}
		
		// Special patterns for common refactoring improvements
		switch pattern {
		case "duplication":
			if strings.Contains(desc, "duplicate") || strings.Contains(desc, "consolidat") ||
			   strings.Contains(path, "default") || strings.Contains(desc, "default") ||
			   strings.Contains(desc, "removed") && (strings.Contains(path, "before_script") || strings.Contains(path, "script")) {
				return true
			}
		case "consolidation":
			if strings.Contains(desc, "consolidat") || strings.Contains(desc, "default") ||
			   strings.Contains(path, "default") || strings.Contains(desc, "configuration has changed") {
				return true
			}
		case "template":
			if strings.Contains(desc, "template") || strings.Contains(path, ".") && strings.Contains(desc, "added") {
				return true
			}
		case "extends":
			if strings.Contains(desc, "extend") || strings.Contains(path, "extend") {
				return true
			}
		case "cache":
			if strings.Contains(path, "cache") || strings.Contains(desc, "cache") {
				return true
			}
		case "variables":
			if strings.Contains(path, "variable") || strings.Contains(desc, "variable") {
				return true
			}
		case "dependencies", "needs":
			if strings.Contains(path, "dependencies") || strings.Contains(path, "needs") ||
			   strings.Contains(desc, "dependencies") || strings.Contains(desc, "needs") {
				return true
			}
		case "matrix":
			if strings.Contains(desc, "matrix") || strings.Contains(path, "matrix") {
				return true
			}
		case "include":
			if strings.Contains(path, "include") || strings.Contains(desc, "include") {
				return true
			}
		}
	}

	return false
}

// getJobChangeType determines what type of change happened to a job (test logic)
func getJobChangeType(comparison *renderer.PipelineComparison, jobName string) JobChangeType {
	for _, jobComp := range comparison.JobComparisons {
		if jobComp.JobName == jobName {
			switch jobComp.Status {
			case renderer.StatusAdded:
				return JobAdded
			case renderer.StatusRemoved:
				return JobRemoved
			case renderer.StatusImproved:
				return JobImproved
			case renderer.StatusIdentical:
				return JobUnchanged
			default:
				return JobRenamed
			}
		}
	}
	return JobRemoved
}

// discoverScenarios automatically finds all scenario directories and creates test scenarios
func discoverScenarios(scenariosPath string) ([]*RefactoringScenario, error) {
	var scenarios []*RefactoringScenario

	entries, err := ioutil.ReadDir(scenariosPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read scenarios directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() || !strings.HasPrefix(entry.Name(), "scenario-") {
			continue
		}

		scenarioName := entry.Name()
		scenarioDir := filepath.Join(scenariosPath, scenarioName)
		
		beforeDir := filepath.Join(scenarioDir, "before")
		afterDir := filepath.Join(scenarioDir, "after")
		includesDir := filepath.Join(scenarioDir, "includes")
		
		if !dirExists(beforeDir) || !dirExists(afterDir) {
			continue
		}

		scenario := &RefactoringScenario{
			Name:        scenarioName,
			Description: generateDescription(scenarioName),
			BeforeDir:   beforeDir,
			AfterDir:    afterDir,
			IncludesDir: includesDir,
			Expectations: getDefaultExpectations(scenarioName),
		}

		configPath := filepath.Join(scenarioDir, "config.yaml")
		if fileExists(configPath) {
			config, err := loadScenarioConfig(configPath)
			if err != nil {
				return nil, fmt.Errorf("failed to load config for %s: %w", scenarioName, err)
			}
			applyScenarioConfig(scenario, config)
		}

		scenarios = append(scenarios, scenario)
	}

	return scenarios, nil
}

// discoverRealisticScenarios discovers realistic application scenarios
func discoverRealisticScenarios(scenariosPath string) ([]*RefactoringScenario, error) {
	var scenarios []*RefactoringScenario

	entries, err := ioutil.ReadDir(scenariosPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read realistic scenarios directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		scenarioName := entry.Name()
		scenarioDir := filepath.Join(scenariosPath, scenarioName)
		
		beforeDir := filepath.Join(scenarioDir, "before")
		afterDir := filepath.Join(scenarioDir, "after")
		
		if !dirExists(beforeDir) || !dirExists(afterDir) {
			continue
		}

		scenario := &RefactoringScenario{
			Name:        scenarioName,
			Description: generateRealisticDescription(scenarioName),
			BeforeDir:   beforeDir,
			AfterDir:    afterDir,
			IncludesDir: filepath.Join(scenarioDir, "includes"),
			Expectations: getRealisticExpectations(scenarioName),
		}

		configPath := filepath.Join(scenarioDir, "config.yaml")
		if fileExists(configPath) {
			config, err := loadScenarioConfig(configPath)
			if err != nil {
				return nil, fmt.Errorf("failed to load config for realistic scenario %s: %w", scenarioName, err)
			}
			applyScenarioConfig(scenario, config)
		}

		scenarios = append(scenarios, scenario)
	}

	return scenarios, nil
}

// Test helper functions
func loadScenarioConfig(configPath string) (*ScenarioConfig, error) {
	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var config ScenarioConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

func applyScenarioConfig(scenario *RefactoringScenario, config *ScenarioConfig) {
	if config.Name != "" {
		scenario.Name = config.Name
	}
	if config.Description != "" {
		scenario.Description = config.Description
	}
	
	scenario.Expectations = RefactoringExpectations{
		ShouldSucceed:          config.Expectations.ShouldSucceed,
		ExpectedIssueReduction: config.Expectations.ExpectedIssueReduction,
		MaxAllowedNewIssues:    config.Expectations.MaxAllowedNewIssues,
		RequiredImprovements:   config.Expectations.RequiredImprovements,
		ForbiddenChanges:       config.Expectations.ForbiddenChanges,
		SemanticEquivalence:    config.Expectations.SemanticEquivalence,
		PerformanceImprovement: config.Expectations.PerformanceImprovement,
		ExpectedJobChanges:     config.Expectations.ExpectedJobChanges,
		ExpectedIssueTypes:     config.Expectations.ExpectedIssueTypes,
		ExpectedIssuePatterns:  config.Expectations.ExpectedIssuePatterns,
		MinimumJobsAnalyzed:    config.Expectations.MinimumJobsAnalyzed,
		ExpectedIncludes:       config.Expectations.ExpectedIncludes,
	}
}

func getDefaultExpectations(scenarioName string) RefactoringExpectations {
	return RefactoringExpectations{
		ShouldSucceed:          true,
		ExpectedIssueReduction: 1,
		MaxAllowedNewIssues:    0,
		SemanticEquivalence:    true,
		PerformanceImprovement: false,
		ExpectedJobChanges:     make(map[string]JobChangeType),
	}
}

func generateDescription(scenarioName string) string {
	descriptions := map[string]string{
		"scenario-1": "Duplicate script blocks consolidation",
		"scenario-2": "Complex include consolidation",
		"scenario-3": "Variable and cache optimization",
		"scenario-4": "Job dependency simplification",
		"scenario-5": "Template extraction and reuse",
		"scenario-6": "Monolithic include breakdown for microservices",
		"scenario-7": "Multi-environment matrix expansion optimization",
		"scenario-8": "Nested template inheritance consolidation",
		"scenario-9": "Cross-repository include optimization",
	}

	if desc, exists := descriptions[scenarioName]; exists {
		return desc
	}
	return fmt.Sprintf("Refactoring scenario %s", scenarioName)
}

func generateRealisticDescription(scenarioName string) string {
	descriptions := map[string]string{
		"flask-microservice": "Realistic Flask microservice CI/CD pipeline optimization",
		"react-frontend":     "Frontend application pipeline with build, test, and deployment stages",
		"go-api":            "Go API service with comprehensive testing and deployment",
		"python-backend":    "Python backend service with database migrations and testing",
	}

	if desc, exists := descriptions[scenarioName]; exists {
		return desc
	}
	return fmt.Sprintf("Realistic application scenario: %s", scenarioName)
}

func getRealisticExpectations(scenarioName string) RefactoringExpectations {
	return RefactoringExpectations{
		ShouldSucceed:          true,
		ExpectedIssueReduction: 3,
		MaxAllowedNewIssues:    2,
		SemanticEquivalence:    false,
		PerformanceImprovement: true,
		ExpectedJobChanges:     make(map[string]JobChangeType),
		MinimumJobsAnalyzed:    5,
	}
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
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