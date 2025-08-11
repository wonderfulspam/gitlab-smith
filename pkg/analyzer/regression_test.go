package analyzer

import (
	"testing"

	"github.com/emt/gitlab-smith/pkg/parser"
)

// TestNoRegressions tests that we have comprehensive coverage and don't miss
// important issues that users would expect to be caught
func TestNoRegressions(t *testing.T) {
	analyzer := New()

	t.Run("Comprehensive real-world config analysis", func(t *testing.T) {
		config := &parser.GitLabConfig{
			Stages: []string{"build", "test", "deploy"},
			Variables: map[string]interface{}{
				"NODE_VERSION": "16",
				"API_SECRET":   "secret123", // Security issue
				"DB_PASSWORD":  "password",  // Security issue
			},
			Include: []parser.Include{
				{Local: "ci/build.yml"},
				{Local: "ci/test.yml"},
				{Local: "ci/deploy.yml"},
				{Local: "ci/lint.yml"},
				{Local: "ci/security.yml"}, // Should trigger include optimization (>3 local)
			},
			Jobs: map[string]*parser.JobConfig{
				// Job with naming issue
				"build project": {
					Stage:        "build",
					Image:        "node", // Should trigger image tag issue
					BeforeScript: []string{"npm install", "npm ci"},
					Script:       []string{"npm run build", "npm run test", "echo done"},
					Cache: &parser.Cache{
						Paths: []string{"node_modules/"}, // Should trigger cache key issue
					},
					Artifacts: &parser.Artifacts{
						Paths: []string{"dist/"}, // Should trigger expiration issue
					},
					Rules: []parser.Rule{
						{If: `$CI_COMMIT_BRANCH == "main"`},
						{If: `$CI_MERGE_REQUEST_ID`},
						{If: `$CI_PIPELINE_SOURCE == "push"`},
						{When: "always"}, // Should trigger verbose rules (>3 rules)
					},
				},
				// Duplicate before_script
				"test1": {
					Stage:        "test",
					BeforeScript: []string{"npm install", "setup env"},
					Script:       []string{"npm test"},
				},
				"test2": {
					Stage:        "test", 
					BeforeScript: []string{"npm install", "setup env"}, // Same as test1
					Script:       []string{"npm run coverage"},
				},
				// Job with complex script
				"complex_deploy": {
					Stage: "deploy",
					Script: []string{
						"echo starting deploy",
						"docker build -t app .",
						"docker tag app registry.com/app:latest",
						"docker push registry.com/app:latest", 
						"kubectl apply -f k8s/",
						"kubectl rollout status deployment/app",
						"curl https://api.example.com/notify", // Hardcoded URL
						"echo deploy complete",
						"cleanup temp files",
						"send notification",
						"update status",
						"log deployment", // >10 lines - should trigger complexity
					},
				},
				// Job with high retry
				"flaky_job": {
					Stage: "test",
					Retry: &parser.Retry{
						Max: 5, // Should trigger retry issue
					},
				},
				// Duplicate script content
				"duplicate_script1": {
					Stage:  "test",
					Script: []string{"echo duplicate", "npm run test"},
				},
				"duplicate_script2": {
					Stage:  "test",
					Script: []string{"echo duplicate", "npm run test"}, // Same as duplicate_script1
				},
			},
		}

		result := analyzer.Analyze(config)

		// Verify we found issues
		if result.TotalIssues < 10 {
			t.Errorf("Expected at least 10 issues in comprehensive config, got %d", result.TotalIssues)
		}

		// Check we have issues of each type
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

		// Check for specific critical issues that users would expect
		expectedIssues := map[string]bool{
			"secret":             false, // Security: variable names with secrets
			"cache":              false, // Performance: cache key missing
			"expiration":         false, // Performance: artifact expiration
			"tag":                false, // Security: image without tag
			"spaces":             false, // Maintainability: job name with spaces
			"complex":            false, // Maintainability: complex script
			"Duplicate":          false, // Maintainability: duplicate code/scripts
			"retry":              false, // Reliability: high retry count
			"rules":              false, // Maintainability: verbose rules
			"include":            false, // Maintainability: include optimization
		}

		for _, issue := range result.Issues {
			for keyword := range expectedIssues {
				if contains(issue.Message, keyword) {
					expectedIssues[keyword] = true
				}
			}
		}

		// Report any missing critical issue types
		for keyword, found := range expectedIssues {
			if !found {
				t.Errorf("Expected to find issue containing '%s' but didn't", keyword)
			}
		}

		// Verify proper categorization
		for _, issue := range result.Issues {
			switch issue.Type {
			case IssueTypePerformance, IssueTypeSecurity, IssueTypeMaintainability, IssueTypeReliability:
				// Good - valid types
			default:
				t.Errorf("Found issue with invalid type: %s", issue.Type)
			}
		}
	})

	t.Run("Check count comparison with expected", func(t *testing.T) {
		// Get list of all registered checks
		checks := analyzer.ListChecks()
		
		// We should have most of the important checks
		if len(checks) < 15 {
			t.Errorf("Expected at least 15 checks registered, got %d", len(checks))
		}

		// Verify we have checks in all categories
		typeCounts := make(map[string]int)
		for _, check := range checks {
			typeCounts[string(check.Type)]++
		}

		if typeCounts["performance"] < 4 {
			t.Errorf("Expected at least 4 performance checks, got %d", typeCounts["performance"])
		}
		if typeCounts["security"] < 2 {
			t.Errorf("Expected at least 2 security checks, got %d", typeCounts["security"])
		}
		if typeCounts["maintainability"] < 6 {
			t.Errorf("Expected at least 6 maintainability checks, got %d", typeCounts["maintainability"])
		}
		if typeCounts["reliability"] < 2 {
			t.Errorf("Expected at least 2 reliability checks, got %d", typeCounts["reliability"])
		}
	})
}

// Uses the contains function from integration_test.go