package analyzer

import (
	"testing"

	"github.com/wonderfulspam/gitlab-smith/pkg/parser"
	"github.com/wonderfulspam/gitlab-smith/pkg/validator/testutil"
)

// TestGoldStandardCases tests the analyzer against gold standard CI/CD configurations
// These are high-quality configurations that should produce ZERO issues
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

			// Gold standard cases should be flawless - zero issues expected
			t.Logf("Gold Standard Case: %s - %s", goldCase.Name, goldCase.Description)
			t.Logf("Jobs: %d, Stages: %d", len(config.Jobs), len(config.Stages))

			if analysisResult.TotalIssues == 0 {
				t.Logf("✅ Perfect! Zero issues found as expected for gold standard case")
			} else {
				t.Errorf("❌ Gold standard case should have ZERO issues, but found %d:", analysisResult.TotalIssues)
				t.Errorf("  Performance: %d, Security: %d, Maintainability: %d, Reliability: %d",
					analysisResult.Summary.Performance,
					analysisResult.Summary.Security,
					analysisResult.Summary.Maintainability,
					analysisResult.Summary.Reliability)

				// Log all issues for debugging
				for _, issue := range analysisResult.Issues {
					t.Errorf("  [%s/%s] %s", issue.Type, issue.Severity, issue.Message)
					if issue.JobName != "" {
						t.Errorf("    Job: %s", issue.JobName)
					}
					if issue.Suggestion != "" {
						t.Errorf("    💡 %s", issue.Suggestion)
					}
				}
			}
		})
	}
}
