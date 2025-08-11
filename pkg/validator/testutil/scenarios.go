package testutil

// RefactoringScenario represents a complete refactoring test case
type RefactoringScenario struct {
	Name         string
	Description  string
	BeforeDir    string
	AfterDir     string
	IncludesDir  string
	Expectations RefactoringExpectations
}

// RefactoringExpectations defines what success looks like for a refactoring
type RefactoringExpectations struct {
	ShouldSucceed          bool                     // Whether the refactor should be considered successful
	ExpectedIssueReduction int                      // Expected reduction in analyzer issues
	MaxAllowedNewIssues    int                      // Maximum new issues that are acceptable
	RequiredImprovements   []string                 // Required improvement categories
	ForbiddenChanges       []string                 // Changes that should not happen
	SemanticEquivalence    bool                     // Whether pipelines should be semantically equivalent
	PerformanceImprovement bool                     // Whether performance should improve
	ExpectedJobChanges     map[string]JobChangeType // Expected changes per job

	// Detailed expectations
	ExpectedIssueTypes    map[string]int // Expected count per issue type
	ExpectedIssuePatterns []string       // Expected issue patterns/messages
	MinimumJobsAnalyzed   int            // Minimum jobs that should be parsed
	ExpectedIncludes      int            // Expected includes (for include scenarios)
}

type JobChangeType string

const (
	JobAdded     JobChangeType = "added"
	JobRemoved   JobChangeType = "removed"
	JobUnchanged JobChangeType = "unchanged"
	JobImproved  JobChangeType = "improved"
	JobRenamed   JobChangeType = "renamed"
)

// ScenarioConfig represents scenario configuration that can be loaded from YAML
type ScenarioConfig struct {
	Name         string `yaml:"name"`
	Description  string `yaml:"description"`
	Expectations struct {
		ShouldSucceed          bool                     `yaml:"should_succeed"`
		ExpectedIssueReduction int                      `yaml:"expected_issue_reduction"`
		MaxAllowedNewIssues    int                      `yaml:"max_allowed_new_issues"`
		RequiredImprovements   []string                 `yaml:"required_improvements"`
		ForbiddenChanges       []string                 `yaml:"forbidden_changes"`
		SemanticEquivalence    bool                     `yaml:"semantic_equivalence"`
		PerformanceImprovement bool                     `yaml:"performance_improvement"`
		ExpectedJobChanges     map[string]JobChangeType `yaml:"expected_job_changes"`

		// Detailed expectations for specific improvement types
		ExpectedIssueTypes    map[string]int `yaml:"expected_issue_types"`    // e.g., "maintainability": 5
		ExpectedIssuePatterns []string       `yaml:"expected_issue_patterns"` // e.g., "template complexity", "matrix opportunities"
		MinimumJobsAnalyzed   int            `yaml:"minimum_jobs_analyzed"`   // Ensure parser is working
		ExpectedIncludes      int            `yaml:"expected_includes"`       // For include consolidation tests
	} `yaml:"expectations"`
}

// GoldStandardCase represents a flawless CI configuration that should produce zero issues
type GoldStandardCase struct {
	Name        string
	Description string
	ConfigFile  string
}
