package analyzer

import (
	"testing"

	"github.com/emt/gitlab-smith/pkg/parser"
)

func TestAnalyze_EmptyConfig(t *testing.T) {
	config := &parser.GitLabConfig{
		Jobs: make(map[string]*parser.JobConfig),
	}

	result := Analyze(config)

	if result.TotalIssues != len(result.Issues) {
		t.Errorf("TotalIssues (%d) doesn't match actual issues count (%d)", result.TotalIssues, len(result.Issues))
	}

	// Empty config should have at least the missing stages issue
	if result.TotalIssues == 0 {
		t.Error("Expected at least one issue for empty config")
	}
}

func TestCheckMissingStages(t *testing.T) {
	t.Run("No stages defined", func(t *testing.T) {
		config := &parser.GitLabConfig{
			Stages: []string{},
			Jobs:   make(map[string]*parser.JobConfig),
		}

		result := &AnalysisResult{Issues: []Issue{}}
		checkMissingStages(config, result)

		if len(result.Issues) != 1 {
			t.Errorf("Expected 1 issue, got %d", len(result.Issues))
		}

		issue := result.Issues[0]
		if issue.Type != IssueTypeMaintainability {
			t.Errorf("Expected maintainability issue, got %s", issue.Type)
		}

		if issue.Severity != SeverityMedium {
			t.Errorf("Expected medium severity, got %s", issue.Severity)
		}
	})

	t.Run("Job references undefined stage", func(t *testing.T) {
		config := &parser.GitLabConfig{
			Stages: []string{"build", "test"},
			Jobs: map[string]*parser.JobConfig{
				"deploy": {
					Stage: "deploy", // Undefined stage
				},
			},
		}

		result := &AnalysisResult{Issues: []Issue{}}
		checkMissingStages(config, result)

		if len(result.Issues) != 1 {
			t.Errorf("Expected 1 issue, got %d", len(result.Issues))
		}

		issue := result.Issues[0]
		if issue.Type != IssueTypeReliability {
			t.Errorf("Expected reliability issue, got %s", issue.Type)
		}

		if issue.Severity != SeverityHigh {
			t.Errorf("Expected high severity, got %s", issue.Severity)
		}

		if issue.JobName != "deploy" {
			t.Errorf("Expected job name 'deploy', got '%s'", issue.JobName)
		}
	})
}

func TestCheckJobNaming(t *testing.T) {
	t.Run("Job name with spaces", func(t *testing.T) {
		config := &parser.GitLabConfig{
			Jobs: map[string]*parser.JobConfig{
				"build project": {
					Stage: "build",
				},
			},
		}

		result := &AnalysisResult{Issues: []Issue{}}
		checkJobNaming(config, result)

		if len(result.Issues) != 1 {
			t.Errorf("Expected 1 issue, got %d", len(result.Issues))
		}

		issue := result.Issues[0]
		if issue.Type != IssueTypeMaintainability {
			t.Errorf("Expected maintainability issue, got %s", issue.Type)
		}

		if issue.Severity != SeverityLow {
			t.Errorf("Expected low severity, got %s", issue.Severity)
		}
	})

	t.Run("Long job name", func(t *testing.T) {
		longName := "this_is_a_very_long_job_name_that_exceeds_the_sixty_three_character_limit"
		config := &parser.GitLabConfig{
			Jobs: map[string]*parser.JobConfig{
				longName: {
					Stage: "build",
				},
			},
		}

		result := &AnalysisResult{Issues: []Issue{}}
		checkJobNaming(config, result)

		if len(result.Issues) != 1 {
			t.Errorf("Expected 1 issue, got %d", len(result.Issues))
		}

		issue := result.Issues[0]
		if issue.Type != IssueTypeReliability {
			t.Errorf("Expected reliability issue, got %s", issue.Type)
		}

		if issue.Severity != SeverityMedium {
			t.Errorf("Expected medium severity, got %s", issue.Severity)
		}
	})

	t.Run("Valid job names", func(t *testing.T) {
		config := &parser.GitLabConfig{
			Jobs: map[string]*parser.JobConfig{
				"build": {
					Stage: "build",
				},
				"test-unit": {
					Stage: "test",
				},
				"deploy_staging": {
					Stage: "deploy",
				},
			},
		}

		result := &AnalysisResult{Issues: []Issue{}}
		checkJobNaming(config, result)

		if len(result.Issues) != 0 {
			t.Errorf("Expected 0 issues, got %d", len(result.Issues))
		}
	})
}

func TestCheckCacheUsage(t *testing.T) {
	t.Run("No cache configured", func(t *testing.T) {
		config := &parser.GitLabConfig{
			Jobs: map[string]*parser.JobConfig{
				"build": {Stage: "build"},
				"test":  {Stage: "test"},
			},
		}

		result := &AnalysisResult{Issues: []Issue{}}
		checkCacheUsage(config, result)

		if len(result.Issues) != 1 {
			t.Errorf("Expected 1 issue, got %d", len(result.Issues))
		}

		issue := result.Issues[0]
		if issue.Type != IssueTypePerformance {
			t.Errorf("Expected performance issue, got %s", issue.Type)
		}
	})

	t.Run("Cache without key", func(t *testing.T) {
		config := &parser.GitLabConfig{
			Jobs: map[string]*parser.JobConfig{
				"build": {
					Stage: "build",
					Cache: &parser.Cache{
						Paths: []string{"node_modules/"},
					},
				},
			},
		}

		result := &AnalysisResult{Issues: []Issue{}}
		checkCacheUsage(config, result)

		if len(result.Issues) != 1 {
			t.Errorf("Expected 1 issue, got %d", len(result.Issues))
		}

		issue := result.Issues[0]
		if issue.Type != IssueTypePerformance {
			t.Errorf("Expected performance issue, got %s", issue.Type)
		}

		if issue.Path != "jobs.build.cache.key" {
			t.Errorf("Expected path 'jobs.build.cache.key', got '%s'", issue.Path)
		}
	})

	t.Run("Cache without paths", func(t *testing.T) {
		config := &parser.GitLabConfig{
			Jobs: map[string]*parser.JobConfig{
				"build": {
					Stage: "build",
					Cache: &parser.Cache{
						Key: "build-cache",
					},
				},
			},
		}

		result := &AnalysisResult{Issues: []Issue{}}
		checkCacheUsage(config, result)

		if len(result.Issues) != 1 {
			t.Errorf("Expected 1 issue, got %d", len(result.Issues))
		}

		issue := result.Issues[0]
		if issue.Type != IssueTypePerformance {
			t.Errorf("Expected performance issue, got %s", issue.Type)
		}

		if issue.Path != "jobs.build.cache.paths" {
			t.Errorf("Expected path 'jobs.build.cache.paths', got '%s'", issue.Path)
		}
	})
}

func TestCheckImageTags(t *testing.T) {
	t.Run("Image without tag", func(t *testing.T) {
		config := &parser.GitLabConfig{
			Jobs: map[string]*parser.JobConfig{
				"build": {
					Stage: "build",
					Image: "node",
				},
			},
		}

		result := &AnalysisResult{Issues: []Issue{}}
		checkImageTags(config, result)

		if len(result.Issues) != 1 {
			t.Errorf("Expected 1 issue, got %d", len(result.Issues))
		}

		issue := result.Issues[0]
		if issue.Type != IssueTypeSecurity {
			t.Errorf("Expected security issue, got %s", issue.Type)
		}

		if issue.Severity != SeverityMedium {
			t.Errorf("Expected medium severity, got %s", issue.Severity)
		}
	})

	t.Run("Image with latest tag", func(t *testing.T) {
		config := &parser.GitLabConfig{
			Jobs: map[string]*parser.JobConfig{
				"build": {
					Stage: "build",
					Image: "node:latest",
				},
			},
		}

		result := &AnalysisResult{Issues: []Issue{}}
		checkImageTags(config, result)

		if len(result.Issues) != 1 {
			t.Errorf("Expected 1 issue, got %d", len(result.Issues))
		}

		issue := result.Issues[0]
		if issue.Type != IssueTypeSecurity {
			t.Errorf("Expected security issue, got %s", issue.Type)
		}

		if issue.Severity != SeverityLow {
			t.Errorf("Expected low severity, got %s", issue.Severity)
		}
	})

	t.Run("Image with specific tag", func(t *testing.T) {
		config := &parser.GitLabConfig{
			Jobs: map[string]*parser.JobConfig{
				"build": {
					Stage: "build",
					Image: "node:16.14.0",
				},
			},
		}

		result := &AnalysisResult{Issues: []Issue{}}
		checkImageTags(config, result)

		if len(result.Issues) != 0 {
			t.Errorf("Expected 0 issues, got %d", len(result.Issues))
		}
	})

	t.Run("Default image issues", func(t *testing.T) {
		config := &parser.GitLabConfig{
			Default: &parser.JobConfig{
				Image: "ubuntu",
			},
			Jobs: make(map[string]*parser.JobConfig),
		}

		result := &AnalysisResult{Issues: []Issue{}}
		checkImageTags(config, result)

		if len(result.Issues) != 1 {
			t.Errorf("Expected 1 issue, got %d", len(result.Issues))
		}

		issue := result.Issues[0]
		if issue.Path != "default.image" {
			t.Errorf("Expected path 'default.image', got '%s'", issue.Path)
		}
	})
}

func TestCheckScriptComplexity(t *testing.T) {
	t.Run("Complex script", func(t *testing.T) {
		script := make([]string, 15) // More than 10 lines
		for i := 0; i < 15; i++ {
			script[i] = "echo line " + string(rune(i+'0'))
		}

		config := &parser.GitLabConfig{
			Jobs: map[string]*parser.JobConfig{
				"build": {
					Stage:  "build",
					Script: script,
				},
			},
		}

		result := &AnalysisResult{Issues: []Issue{}}
		checkScriptComplexity(config, result)

		if len(result.Issues) != 1 {
			t.Errorf("Expected 1 issue, got %d", len(result.Issues))
		}

		issue := result.Issues[0]
		if issue.Type != IssueTypeMaintainability {
			t.Errorf("Expected maintainability issue, got %s", issue.Type)
		}
	})

	t.Run("Hardcoded URL in script", func(t *testing.T) {
		config := &parser.GitLabConfig{
			Jobs: map[string]*parser.JobConfig{
				"deploy": {
					Stage: "deploy",
					Script: []string{
						"curl -X POST https://api.example.com/deploy",
						"echo 'Deployed'",
					},
				},
			},
		}

		result := &AnalysisResult{Issues: []Issue{}}
		checkScriptComplexity(config, result)

		if len(result.Issues) != 1 {
			t.Errorf("Expected 1 issue, got %d", len(result.Issues))
		}

		issue := result.Issues[0]
		if issue.Type != IssueTypeMaintainability {
			t.Errorf("Expected maintainability issue, got %s", issue.Type)
		}

		if !contains(issue.Message, "Hardcoded URL") {
			t.Errorf("Expected message to contain 'Hardcoded URL', got '%s'", issue.Message)
		}
	})
}

func TestCheckEnvironmentVariables(t *testing.T) {
	t.Run("Potential secret in variable name", func(t *testing.T) {
		config := &parser.GitLabConfig{
			Variables: map[string]interface{}{
				"API_PASSWORD": "secret123",
				"DB_SECRET":    "dbsecret",
				"AUTH_TOKEN":   "token123",
			},
			Jobs: make(map[string]*parser.JobConfig),
		}

		result := &AnalysisResult{Issues: []Issue{}}
		checkEnvironmentVariables(config, result)

		if len(result.Issues) != 3 {
			t.Errorf("Expected 3 issues, got %d", len(result.Issues))
		}

		for _, issue := range result.Issues {
			if issue.Type != IssueTypeSecurity {
				t.Errorf("Expected security issue, got %s", issue.Type)
			}

			if issue.Severity != SeverityHigh {
				t.Errorf("Expected high severity, got %s", issue.Severity)
			}
		}
	})

	t.Run("Job-level variable issues", func(t *testing.T) {
		config := &parser.GitLabConfig{
			Jobs: map[string]*parser.JobConfig{
				"deploy": {
					Stage: "deploy",
					Variables: map[string]interface{}{
						"DEPLOY_PASSWORD": "secret",
					},
				},
			},
		}

		result := &AnalysisResult{Issues: []Issue{}}
		checkEnvironmentVariables(config, result)

		if len(result.Issues) != 1 {
			t.Errorf("Expected 1 issue, got %d", len(result.Issues))
		}

		issue := result.Issues[0]
		if issue.Path != "jobs.deploy.variables.DEPLOY_PASSWORD" {
			t.Errorf("Expected path 'jobs.deploy.variables.DEPLOY_PASSWORD', got '%s'", issue.Path)
		}
	})
}

func TestCheckRetryConfiguration(t *testing.T) {
	t.Run("High retry count", func(t *testing.T) {
		config := &parser.GitLabConfig{
			Jobs: map[string]*parser.JobConfig{
				"flaky_test": {
					Stage: "test",
					Retry: &parser.Retry{
						Max: 5,
					},
				},
			},
		}

		result := &AnalysisResult{Issues: []Issue{}}
		checkRetryConfiguration(config, result)

		if len(result.Issues) != 1 {
			t.Errorf("Expected 1 issue, got %d", len(result.Issues))
		}

		issue := result.Issues[0]
		if issue.Type != IssueTypeReliability {
			t.Errorf("Expected reliability issue, got %s", issue.Type)
		}

		if issue.Severity != SeverityLow {
			t.Errorf("Expected low severity, got %s", issue.Severity)
		}
	})

	t.Run("Acceptable retry count", func(t *testing.T) {
		config := &parser.GitLabConfig{
			Jobs: map[string]*parser.JobConfig{
				"test": {
					Stage: "test",
					Retry: &parser.Retry{
						Max: 2,
					},
				},
			},
		}

		result := &AnalysisResult{Issues: []Issue{}}
		checkRetryConfiguration(config, result)

		if len(result.Issues) != 0 {
			t.Errorf("Expected 0 issues, got %d", len(result.Issues))
		}
	})
}

func TestCheckDuplicatedCode(t *testing.T) {
	t.Run("Duplicated scripts", func(t *testing.T) {
		config := &parser.GitLabConfig{
			Jobs: map[string]*parser.JobConfig{
				"test1": {
					Stage:  "test",
					Script: []string{"npm test", "npm run coverage"},
				},
				"test2": {
					Stage:  "test",
					Script: []string{"npm test", "npm run coverage"},
				},
				"build": {
					Stage:  "build",
					Script: []string{"npm run build"},
				},
			},
		}

		result := &AnalysisResult{Issues: []Issue{}}
		checkDuplicatedCode(config, result)

		if len(result.Issues) != 1 {
			t.Errorf("Expected 1 issue, got %d", len(result.Issues))
		}

		issue := result.Issues[0]
		if issue.Type != IssueTypeMaintainability {
			t.Errorf("Expected maintainability issue, got %s", issue.Type)
		}

		if !contains(issue.Message, "test1") || !contains(issue.Message, "test2") {
			t.Errorf("Expected message to contain both job names, got '%s'", issue.Message)
		}
	})
}

func TestCheckDependencyChains(t *testing.T) {
	t.Run("Long dependency chain", func(t *testing.T) {
		// Create a config with a job that has many dependencies
		config := &parser.GitLabConfig{
			Jobs: map[string]*parser.JobConfig{
				"job_with_many_deps": {
					Dependencies: []string{"dep1", "dep2", "dep3", "dep4", "dep5", "dep6"},
				},
				"normal_job": {
					Dependencies: []string{"dep1", "dep2"},
				},
				"dep1": {},
				"dep2": {},
				"dep3": {},
				"dep4": {},
				"dep5": {},
				"dep6": {},
			},
		}

		result := &AnalysisResult{Issues: []Issue{}}
		checkDependencyChains(config, result)

		if len(result.Issues) != 1 {
			t.Errorf("Expected 1 issue, got %d", len(result.Issues))
		}

		issue := result.Issues[0]
		if issue.Type != IssueTypePerformance {
			t.Errorf("Expected performance issue, got %s", issue.Type)
		}

		if issue.JobName != "job_with_many_deps" {
			t.Errorf("Expected job name 'job_with_many_deps', got '%s'", issue.JobName)
		}
	})
}

func TestCalculateSummary(t *testing.T) {
	issues := []Issue{
		{Type: IssueTypePerformance},
		{Type: IssueTypePerformance},
		{Type: IssueTypeSecurity},
		{Type: IssueTypeMaintainability},
		{Type: IssueTypeMaintainability},
		{Type: IssueTypeMaintainability},
		{Type: IssueTypeReliability},
	}

	summary := calculateSummary(issues)

	if summary.Performance != 2 {
		t.Errorf("Expected 2 performance issues, got %d", summary.Performance)
	}

	if summary.Security != 1 {
		t.Errorf("Expected 1 security issue, got %d", summary.Security)
	}

	if summary.Maintainability != 3 {
		t.Errorf("Expected 3 maintainability issues, got %d", summary.Maintainability)
	}

	if summary.Reliability != 1 {
		t.Errorf("Expected 1 reliability issue, got %d", summary.Reliability)
	}
}

func TestFilterBySeverity(t *testing.T) {
	result := &AnalysisResult{
		Issues: []Issue{
			{Severity: SeverityHigh, Type: IssueTypeSecurity},
			{Severity: SeverityMedium, Type: IssueTypePerformance},
			{Severity: SeverityHigh, Type: IssueTypeReliability},
			{Severity: SeverityLow, Type: IssueTypeMaintainability},
		},
	}

	highSeverityIssues := result.FilterBySeverity(SeverityHigh)

	if len(highSeverityIssues) != 2 {
		t.Errorf("Expected 2 high severity issues, got %d", len(highSeverityIssues))
	}

	for _, issue := range highSeverityIssues {
		if issue.Severity != SeverityHigh {
			t.Errorf("Expected high severity, got %s", issue.Severity)
		}
	}
}

func TestFilterByType(t *testing.T) {
	result := &AnalysisResult{
		Issues: []Issue{
			{Type: IssueTypeSecurity, Severity: SeverityHigh},
			{Type: IssueTypePerformance, Severity: SeverityMedium},
			{Type: IssueTypeSecurity, Severity: SeverityLow},
			{Type: IssueTypeMaintainability, Severity: SeverityMedium},
		},
	}

	securityIssues := result.FilterByType(IssueTypeSecurity)

	if len(securityIssues) != 2 {
		t.Errorf("Expected 2 security issues, got %d", len(securityIssues))
	}

	for _, issue := range securityIssues {
		if issue.Type != IssueTypeSecurity {
			t.Errorf("Expected security issue, got %s", issue.Type)
		}
	}
}

func TestAnalyze_ComprehensiveConfig(t *testing.T) {
	config := &parser.GitLabConfig{
		Stages: []string{"build", "test", "deploy"},
		Variables: map[string]interface{}{
			"NODE_VERSION": "16",
			"API_SECRET":   "secret123", // Should trigger security issue
		},
		Jobs: map[string]*parser.JobConfig{
			"build project": { // Should trigger naming issue
				Stage: "build",
				Image: "node", // Should trigger image tag issue
				Script: []string{
					"npm install",
					"npm run build",
					"curl https://api.example.com/notify", // Should trigger hardcoded URL issue
				},
				Cache: &parser.Cache{
					Paths: []string{"node_modules/"}, // Should trigger cache key issue
				},
			},
			"test": {
				Stage:  "test",
				Script: make([]string, 15), // Should trigger complexity issue
				Retry: &parser.Retry{
					Max: 5, // Should trigger retry issue
				},
			},
		},
	}

	result := Analyze(config)

	if result.TotalIssues == 0 {
		t.Error("Expected issues to be found in comprehensive config")
	}

	if result.Summary.Performance == 0 {
		t.Error("Expected performance issues")
	}

	if result.Summary.Security == 0 {
		t.Error("Expected security issues")
	}

	if result.Summary.Maintainability == 0 {
		t.Error("Expected maintainability issues")
	}

	if result.Summary.Reliability == 0 {
		t.Error("Expected reliability issues")
	}

	// Verify total matches sum
	expectedTotal := result.Summary.Performance + result.Summary.Security + result.Summary.Maintainability + result.Summary.Reliability
	if result.TotalIssues != expectedTotal {
		t.Errorf("TotalIssues (%d) doesn't match summary totals (%d)", result.TotalIssues, expectedTotal)
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsSubstring(s, substr)))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestCheckWorkflowOptimization(t *testing.T) {
	t.Run("Missing workflow with branch-specific rules", func(t *testing.T) {
		config := &parser.GitLabConfig{
			Jobs: map[string]*parser.JobConfig{
				"job1": {
					Rules: []parser.Rule{
						{If: `$CI_COMMIT_BRANCH == "main"`},
					},
				},
				"job2": {
					Rules: []parser.Rule{
						{If: `$CI_COMMIT_BRANCH == "main"`},
					},
				},
			},
		}

		result := &AnalysisResult{Issues: []Issue{}}
		checkWorkflowOptimization(config, result)

		found := false
		for _, issue := range result.Issues {
			if issue.Path == "workflow" && issue.Type == IssueTypePerformance {
				found = true
				break
			}
		}
		if !found {
			t.Error("Expected workflow optimization issue for branch-specific rules")
		}
	})

	t.Run("Missing workflow with MR-specific rules", func(t *testing.T) {
		config := &parser.GitLabConfig{
			Jobs: map[string]*parser.JobConfig{
				"job1": {
					Rules: []parser.Rule{
						{If: `$CI_MERGE_REQUEST_ID`},
					},
				},
				"job2": {
					Rules: []parser.Rule{
						{If: `$CI_MERGE_REQUEST_ID`},
					},
				},
				"job3": {
					Rules: []parser.Rule{
						{If: `$CI_MERGE_REQUEST_ID`},
					},
				},
			},
		}

		result := &AnalysisResult{Issues: []Issue{}}
		checkWorkflowOptimization(config, result)

		found := false
		for _, issue := range result.Issues {
			if issue.Path == "workflow" && issue.Type == IssueTypePerformance {
				found = true
				break
			}
		}
		if !found {
			t.Error("Expected workflow optimization issue for MR-specific rules")
		}
	})

	t.Run("Redundant job rules", func(t *testing.T) {
		config := &parser.GitLabConfig{
			Jobs: map[string]*parser.JobConfig{
				"job1": {
					Rules: []parser.Rule{
						{If: `$CI_PIPELINE_SOURCE == "push"`},
					},
				},
				"job2": {
					Rules: []parser.Rule{
						{If: `$CI_PIPELINE_SOURCE == "push"`},
					},
				},
				"job3": {
					Rules: []parser.Rule{
						{If: `$CI_PIPELINE_SOURCE == "push"`},
					},
				},
			},
		}

		result := &AnalysisResult{Issues: []Issue{}}
		checkWorkflowOptimization(config, result)

		found := false
		for _, issue := range result.Issues {
			if issue.Path == "jobs" && issue.Type == IssueTypeMaintainability {
				found = true
				break
			}
		}
		if !found {
			t.Error("Expected maintainability issue for redundant job rules")
		}
	})

	t.Run("Branch-specific optimization with workflow", func(t *testing.T) {
		config := &parser.GitLabConfig{
			Workflow: &parser.Workflow{
				Rules: []parser.Rule{
					{When: "always"},
				},
			},
			Jobs: map[string]*parser.JobConfig{
				"main-job-1": {
					Rules: []parser.Rule{
						{If: `$CI_COMMIT_BRANCH == "main"`},
					},
				},
				"main-job-2": {
					Rules: []parser.Rule{
						{If: `$CI_COMMIT_BRANCH == "main"`},
					},
				},
				"main-job-3": {
					Rules: []parser.Rule{
						{If: `$CI_COMMIT_BRANCH == "main"`},
					},
				},
				"mr-job": {
					Rules: []parser.Rule{
						{If: `$CI_MERGE_REQUEST_ID`},
					},
				},
				"common-job": {},
			},
		}

		result := &AnalysisResult{Issues: []Issue{}}
		checkWorkflowOptimization(config, result)

		// Should detect optimization opportunity due to different job counts
		// Main branch: 4 jobs (3 main-jobs + common-job), MR: 2 jobs (mr-job + common-job)
		// Difference: |4-2| = 2, which is > 5/3 = 1.67, so should trigger
		found := false
		for _, issue := range result.Issues {
			if issue.Path == "workflow" && issue.Type == IssueTypePerformance {
				found = true
				break
			}
		}
		if !found {
			t.Error("Expected performance issue for branch-specific optimization")
		}
	})

	t.Run("No issues with optimal workflow", func(t *testing.T) {
		config := &parser.GitLabConfig{
			Workflow: &parser.Workflow{
				Rules: []parser.Rule{
					{If: `$CI_PIPELINE_SOURCE == "push"`, When: "always"},
				},
			},
			Jobs: map[string]*parser.JobConfig{
				"test": {
					Script: []string{"echo test"},
				},
			},
		}

		result := &AnalysisResult{Issues: []Issue{}}
		checkWorkflowOptimization(config, result)

		for _, issue := range result.Issues {
			if issue.Path == "workflow" {
				t.Errorf("Unexpected workflow issue: %s", issue.Message)
			}
		}
	})
}

func TestHasBranchSpecificRules(t *testing.T) {
	tests := []struct {
		name     string
		job      *parser.JobConfig
		expected bool
	}{
		{
			name: "Job with branch-specific if condition",
			job: &parser.JobConfig{
				Rules: []parser.Rule{
					{If: `$CI_COMMIT_BRANCH == "main"`},
				},
			},
			expected: true,
		},
		{
			name: "Job with main branch only",
			job: &parser.JobConfig{
				Only: "main",
			},
			expected: true,
		},
		{
			name: "Job with master branch only",
			job: &parser.JobConfig{
				Only: "master",
			},
			expected: true,
		},
		{
			name: "Job without branch-specific rules",
			job: &parser.JobConfig{
				Rules: []parser.Rule{
					{If: `$CI_PIPELINE_SOURCE == "push"`},
				},
			},
			expected: false,
		},
		{
			name:     "Job with no rules",
			job:      &parser.JobConfig{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hasBranchSpecificRules(tt.job)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestHasMRSpecificRules(t *testing.T) {
	tests := []struct {
		name     string
		job      *parser.JobConfig
		expected bool
	}{
		{
			name: "Job with MR ID check",
			job: &parser.JobConfig{
				Rules: []parser.Rule{
					{If: `$CI_MERGE_REQUEST_ID`},
				},
			},
			expected: true,
		},
		{
			name: "Job with merge request event check",
			job: &parser.JobConfig{
				Rules: []parser.Rule{
					{If: `$CI_PIPELINE_SOURCE == "merge_request_event"`},
				},
			},
			expected: true,
		},
		{
			name: "Job with merge_requests only",
			job: &parser.JobConfig{
				Only: "merge_requests",
			},
			expected: true,
		},
		{
			name: "Job with merge_requests in array",
			job: &parser.JobConfig{
				Only: []interface{}{"merge_requests", "pushes"},
			},
			expected: true,
		},
		{
			name: "Job without MR-specific rules",
			job: &parser.JobConfig{
				Rules: []parser.Rule{
					{If: `$CI_COMMIT_BRANCH == "main"`},
				},
			},
			expected: false,
		},
		{
			name:     "Job with no rules",
			job:      &parser.JobConfig{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hasMRSpecificRules(tt.job)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}
