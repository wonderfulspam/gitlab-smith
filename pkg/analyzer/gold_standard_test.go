package analyzer

import (
	"strings"
	"testing"

	"github.com/wonderfulspam/gitlab-smith/pkg/analyzer/types"
	"github.com/wonderfulspam/gitlab-smith/pkg/parser"
	"github.com/wonderfulspam/gitlab-smith/pkg/validator/testutil"
)

// TestGoldStandardCases tests the analyzer against gold standard CI/CD configurations
// These are high-quality configurations that should produce minimal or no issues
func TestGoldStandardCases(t *testing.T) {
	casesPath := "../../test/gold-standard-cases"

	cases, err := testutil.DiscoverGoldStandardCases(casesPath)
	if err != nil {
		t.Fatalf("Failed to discover gold standard cases: %v", err)
	}

	if len(cases) == 0 {
		t.Skip("No gold standard cases found")
	}

	for _, goldCase := range cases {
		t.Run(goldCase.Name, func(t *testing.T) {
			// Parse the configuration
			config, err := parser.ParseFile(goldCase.ConfigFile)
			if err != nil {
				t.Fatalf("Failed to parse gold standard case %s: %v", goldCase.Name, err)
			}

			// Run analysis
			analysisResult := Analyze(config)

			// Convert analysis result to gold standard result format
			result := convertToGoldStandardResult(analysisResult, config)

			// Validate against expectations
			success, issues, warnings := testutil.ValidateGoldStandardExpectations(result, goldCase.Expectations)

			// Report results
			t.Logf("Gold Standard Case: %s - %s", goldCase.Name, goldCase.Description)
			t.Logf("Success: %t", success)
			t.Logf("Total issues found: %d (max allowed: %d)", result.TotalIssues, goldCase.Expectations.MaxAllowedIssues)

			// Log issue breakdown by category
			for category, count := range result.IssuesByCategory {
				t.Logf("  %s: %d issues", category, count)
			}

			// Log pipeline characteristics
			t.Logf("Pipeline characteristics:")
			t.Logf("  Jobs: %d, Stages: %d", result.JobCount, result.StageCount)
			t.Logf("  Has caching: %t, Has artifacts: %t", result.HasCaching, result.HasArtifacts)
			t.Logf("  Has coverage: %t, Has security scanning: %t", result.HasCoverage, result.HasSecurityScanning)

			// Report warnings (these don't fail the test)
			for _, warning := range warnings {
				t.Logf("Warning: %s", warning)
			}

			// Report issues (these fail the test)
			for _, issue := range issues {
				t.Logf("Issue: %s", issue)
			}

			// Log specific analyzer issues for debugging
			if len(result.Issues) > 0 {
				t.Logf("Analyzer issues found:")
				for _, issue := range result.Issues {
					if issue.Severity == "high" || issue.Severity == "medium" {
						t.Logf("  [%s/%s] %s (Job: %s)", issue.Category, issue.Severity, issue.Message, issue.JobName)
						if issue.Suggestion != "" {
							t.Logf("    Suggestion: %s", issue.Suggestion)
						}
					}
				}
			}

			// Fail if there are validation issues
			if !success && goldCase.Expectations.ShouldSucceed {
				t.Errorf("Gold standard case %s expected to succeed but failed validation", goldCase.Name)
			}
		})
	}
}

// convertToGoldStandardResult converts analyzer result to gold standard result format
func convertToGoldStandardResult(analysisResult *types.AnalysisResult, config *parser.GitLabConfig) *testutil.GoldStandardAnalysisResult {
	result := &testutil.GoldStandardAnalysisResult{
		TotalIssues:         analysisResult.TotalIssues,
		IssuesByCategory:    make(map[string]int),
		JobCount:            len(config.Jobs),
		StageCount:          len(config.Stages),
		HasArtifacts:        hasArtifacts(config),
		HasCaching:          hasCaching(config),
		HasCoverage:         hasCoverage(analysisResult),
		HasSecurityScanning: hasSecurityScanning(config),
		ParallelCapable:     isParallelCapable(config),
		HasDependencies:     hasDependencies(config),
		Issues:              make([]testutil.GoldStandardIssue, 0),
	}

	// Count issues by category
	result.IssuesByCategory["performance"] = analysisResult.Summary.Performance
	result.IssuesByCategory["security"] = analysisResult.Summary.Security
	result.IssuesByCategory["maintainability"] = analysisResult.Summary.Maintainability
	result.IssuesByCategory["reliability"] = analysisResult.Summary.Reliability

	// Convert individual issues
	for _, issue := range analysisResult.Issues {
		result.Issues = append(result.Issues, testutil.GoldStandardIssue{
			Category:   string(issue.Type),
			Severity:   string(issue.Severity),
			Message:    issue.Message,
			JobName:    issue.JobName,
			Path:       issue.Path,
			Suggestion: issue.Suggestion,
		})
	}

	return result
}

// Helper functions to analyze configuration characteristics

func hasArtifacts(config *parser.GitLabConfig) bool {
	for _, job := range config.Jobs {
		if job.Artifacts != nil && (len(job.Artifacts.Paths) > 0 || len(job.Artifacts.Reports) > 0) {
			return true
		}
	}
	return false
}

func hasCaching(config *parser.GitLabConfig) bool {
	// Check for global cache configuration
	if config.Cache != nil {
		return true
	}
	// Check for default cache configuration
	if config.Default != nil && config.Default.Cache != nil {
		return true
	}
	// Check for job-level cache configuration
	for _, job := range config.Jobs {
		if job.Cache != nil {
			return true
		}
	}
	return false
}

func hasCoverage(analysisResult *types.AnalysisResult) bool {
	// Check if any job mentions coverage in the analysis
	for _, issue := range analysisResult.Issues {
		if strings.Contains(strings.ToLower(issue.Message), "coverage") ||
			strings.Contains(strings.ToLower(issue.Path), "coverage") {
			return true
		}
	}
	return false
}

func hasSecurityScanning(config *parser.GitLabConfig) bool {
	for _, job := range config.Jobs {
		// Check for common security scanning tools in scripts
		for _, script := range job.Script {
			scriptLower := strings.ToLower(script)
			if strings.Contains(scriptLower, "gosec") ||
				strings.Contains(scriptLower, "snyk") ||
				strings.Contains(scriptLower, "safety") ||
				strings.Contains(scriptLower, "bandit") ||
				strings.Contains(scriptLower, "dependency") && strings.Contains(scriptLower, "scan") {
				return true
			}
		}

		// Check for security scanning in artifacts reports
		if job.Artifacts != nil {
			for reportType := range job.Artifacts.Reports {
				if strings.Contains(strings.ToLower(reportType), "security") ||
					strings.Contains(strings.ToLower(reportType), "sast") {
					return true
				}
			}
		}
	}
	return false
}

func isParallelCapable(config *parser.GitLabConfig) bool {
	// Check if jobs can run in parallel (multiple jobs in same stage without dependencies)
	stageJobs := make(map[string]int)
	for _, job := range config.Jobs {
		stageJobs[job.Stage]++
	}

	for _, count := range stageJobs {
		if count > 1 {
			return true
		}
	}
	return false
}

func hasDependencies(config *parser.GitLabConfig) bool {
	for _, job := range config.Jobs {
		if len(job.Dependencies) > 0 {
			return true
		}
		// Check if Needs is defined (could be string, array, or map)
		if job.Needs != nil {
			return true
		}
	}
	return false
}
