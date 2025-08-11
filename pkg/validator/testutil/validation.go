package testutil

import (
	"fmt"
	"strings"

	"github.com/wonderfulspam/gitlab-smith/pkg/differ"
	"github.com/wonderfulspam/gitlab-smith/pkg/renderer"
)

// RefactoringResult represents the result of a refactoring validation
// This should match the one in pkg/validator/validator.go
// We define it here to avoid circular imports
type RefactoringResult struct {
	ActualChanges        *differ.DiffResult
	AnalysisImprovement  int
	PipelineComparison   *renderer.PipelineComparison
	BehavioralValidation interface{} // Using interface{} to avoid circular dependency
}

// ValidateExpectations validates the refactoring result against expectations
func ValidateExpectations(result *RefactoringResult, expectations RefactoringExpectations) (bool, []string, []string) {
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
	if expectations.SemanticEquivalence && !IsSemanticallySimilar(result) {
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
		if ContainsChange(result, forbidden) {
			issues = append(issues, fmt.Sprintf("Forbidden change detected: %s", forbidden))
			success = false
		}
	}

	// Check required improvements
	for _, required := range expectations.RequiredImprovements {
		if !ContainsChange(result, required) {
			issues = append(issues, fmt.Sprintf("Required improvement missing: %s", required))
			success = false
		}
	}

	// Check job changes
	if result.PipelineComparison != nil {
		for jobName, expectedChange := range expectations.ExpectedJobChanges {
			actualChange := GetJobChangeType(result.PipelineComparison, jobName)
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
			if !ContainsChange(result, pattern) {
				warnings = append(warnings,
					fmt.Sprintf("Expected issue pattern '%s' not found in analysis", pattern))
			}
		}
	}

	return success, issues, warnings
}

// IsSemanticallySimilar checks if two configurations are semantically similar
func IsSemanticallySimilar(result *RefactoringResult) bool {
	if result.ActualChanges == nil {
		return true // No changes means semantically equivalent
	}

	significantChanges := 0

	for _, change := range result.ActualChanges.Semantic {
		if IsSignificantChange(change) {
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

// IsSignificantChange determines if a change affects pipeline behavior
func IsSignificantChange(change differ.ConfigDiff) bool {
	// Use the Behavioral field from the differ
	return change.Behavioral
}

// ContainsChange checks if the diff contains a specific type of change
func ContainsChange(result *RefactoringResult, changePattern string) bool {
	if result.ActualChanges == nil {
		return false
	}

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

// GetJobChangeType determines what type of change happened to a job
func GetJobChangeType(comparison *renderer.PipelineComparison, jobName string) JobChangeType {
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

// GoldStandardAnalysisResult represents the result of analyzing a gold standard case
type GoldStandardAnalysisResult struct {
	TotalIssues         int
	IssuesByCategory    map[string]int
	JobCount            int
	StageCount          int
	HasArtifacts        bool
	HasCaching          bool
	HasCoverage         bool
	HasSecurityScanning bool
	ParallelCapable     bool
	HasDependencies     bool
	Issues              []GoldStandardIssue
}

// GoldStandardIssue represents an issue found in gold standard analysis
type GoldStandardIssue struct {
	Category   string
	Severity   string
	Message    string
	JobName    string
	Path       string
	Suggestion string
}

// ValidateGoldStandardExpectations validates the analysis result against gold standard expectations
func ValidateGoldStandardExpectations(result *GoldStandardAnalysisResult, expectations GoldStandardExpectations) (bool, []string, []string) {
	var issues []string
	var warnings []string
	success := true

	// Check total issues allowed
	if result.TotalIssues > expectations.MaxAllowedIssues {
		issues = append(issues,
			fmt.Sprintf("Too many issues found: %d (max allowed: %d)",
				result.TotalIssues, expectations.MaxAllowedIssues))
		success = false
	}

	// Check categories that should have zero issues
	for _, category := range expectations.ExpectedZeroCategories {
		if count, exists := result.IssuesByCategory[category]; exists && count > 0 {
			issues = append(issues,
				fmt.Sprintf("Expected zero %s issues, found %d", category, count))
			success = false
		}
	}

	// Check categories that should have minimal issues
	for category, maxAllowed := range expectations.ExpectedMinimalCategories {
		if count, exists := result.IssuesByCategory[category]; exists && count > maxAllowed {
			issues = append(issues,
				fmt.Sprintf("Too many %s issues: %d (max allowed: %d)",
					category, count, maxAllowed))
			success = false
		}
	}

	// Check job metrics expectations
	if expectations.ExpectedJobs.Total > 0 && result.JobCount != expectations.ExpectedJobs.Total {
		warnings = append(warnings,
			fmt.Sprintf("Expected %d jobs, found %d", expectations.ExpectedJobs.Total, result.JobCount))
	}

	if expectations.ExpectedJobs.Stages > 0 && result.StageCount != expectations.ExpectedJobs.Stages {
		warnings = append(warnings,
			fmt.Sprintf("Expected %d stages, found %d", expectations.ExpectedJobs.Stages, result.StageCount))
	}

	if expectations.ExpectedJobs.HasCaching && !result.HasCaching {
		warnings = append(warnings, "Expected caching configuration not found")
	}

	if expectations.ExpectedJobs.HasArtifacts && !result.HasArtifacts {
		warnings = append(warnings, "Expected artifacts configuration not found")
	}

	if expectations.ExpectedJobs.HasCoverage && !result.HasCoverage {
		warnings = append(warnings, "Expected coverage configuration not found")
	}

	if expectations.ExpectedJobs.HasSecurityScanning && !result.HasSecurityScanning {
		warnings = append(warnings, "Expected security scanning not found")
	}

	// Check for unacceptable issues
	for _, issue := range result.Issues {
		isAcceptable := false
		for _, acceptable := range expectations.AcceptableMinorIssues {
			if strings.Contains(strings.ToLower(issue.Message), strings.ToLower(acceptable)) {
				isAcceptable = true
				break
			}
		}

		if !isAcceptable && (issue.Severity == "high" || issue.Severity == "medium") {
			issues = append(issues,
				fmt.Sprintf("Unacceptable %s issue: %s", issue.Category, issue.Message))
			success = false
		}
	}

	return success, issues, warnings
}
