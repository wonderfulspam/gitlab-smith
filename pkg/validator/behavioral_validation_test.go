package validator

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/wonderfulspam/gitlab-smith/pkg/gitlab"
)

func TestBehavioralValidation_BasicFlow(t *testing.T) {
	// Skip this test by default as it requires GitLab setup
	if testing.Short() {
		t.Skip("Skipping behavioral validation test in short mode")
	}

	// Create validator with simulation client
	validator := NewRefactoringValidator()

	// Test with simple refactoring scenario
	beforeDir := "../../test/simple-refactoring-cases/job-consolidation-before"
	afterDir := "../../test/simple-refactoring-cases/job-consolidation-after"

	// Skip if test directories don't exist (to avoid test failures in CI)
	if !testDirExists(beforeDir) || !testDirExists(afterDir) {
		t.Skip("Test directories not found, skipping behavioral validation test")
	}

	result, err := validator.CompareConfigurations(beforeDir, afterDir)
	if err != nil {
		t.Fatalf("CompareConfigurations failed: %v", err)
	}

	// Check that we have some result
	if result == nil {
		t.Fatal("Expected validation results, got nil")
	}

	// Verify basic comparison was performed
	if result.ActualChanges == nil {
		t.Error("Expected comparison results, got nil")
	}
}

func TestBehavioralValidation_SimulationMode(t *testing.T) {
	// Test the behavioral validation logic with simulation
	validator := NewRefactoringValidator()
	validator.EnableFullTesting()

	// Create temporary test directories with CI configs
	beforeDir, afterDir := createTestDirs(t)
	defer cleanupTestDirs(beforeDir, afterDir)

	result, err := validator.CompareConfigurations(beforeDir, afterDir)
	if err != nil {
		t.Fatalf("CompareConfigurations failed: %v", err)
	}

	// Verify results structure
	if result == nil {
		t.Fatal("Expected validation result, got nil")
	}

	// Should have pipeline comparison in simulation mode
	if result.PipelineComparison == nil {
		t.Error("Expected pipeline comparison, got nil")
	}
}

func TestExecutionComparison(t *testing.T) {
	validator := &RefactoringValidator{}

	// Define test execution results
	before := []string{"build", "test", "deploy"}
	beforeTimes := map[string]int64{"build": 120, "test": 180, "deploy": 90}

	after := []string{"build", "test-optimized", "deploy"}
	afterTimes := map[string]int64{"build": 120, "test-optimized": 150, "deploy": 90}

	comparison := &PipelineExecutionComparison{
		BeforeJobsExecuted:   before,
		AfterJobsExecuted:    after,
		ExecutionTimesBefore: beforeTimes,
		ExecutionTimesAfter:  afterTimes,
		JobsAdded:            []string{"test-optimized"},
		JobsRemoved:          []string{"test"},
		JobsModified:         []string{},
	}

	// Test behavioral equivalence determination
	isEquivalent := validator.determineBehavioralEquivalence(comparison)
	if isEquivalent {
		t.Error("Expected configurations with job changes to not be behaviorally equivalent")
	}
}

func TestBehavioralEquivalence(t *testing.T) {
	validator := &RefactoringValidator{}

	testCases := []struct {
		name       string
		comparison *PipelineExecutionComparison
		expected   bool
	}{
		{
			name: "identical executions",
			comparison: &PipelineExecutionComparison{
				BeforeJobsExecuted: []string{"build", "test"},
				AfterJobsExecuted:  []string{"build", "test"},
				JobsAdded:          []string{},
				JobsRemoved:        []string{},
				JobsModified:       []string{},
			},
			expected: true,
		},
		{
			name: "job added",
			comparison: &PipelineExecutionComparison{
				BeforeJobsExecuted: []string{"build", "test"},
				AfterJobsExecuted:  []string{"build", "test", "lint"},
				JobsAdded:          []string{"lint"},
				JobsRemoved:        []string{},
				JobsModified:       []string{},
			},
			expected: false,
		},
		{
			name: "job removed",
			comparison: &PipelineExecutionComparison{
				BeforeJobsExecuted: []string{"build", "test", "lint"},
				AfterJobsExecuted:  []string{"build", "test"},
				JobsAdded:          []string{},
				JobsRemoved:        []string{"lint"},
				JobsModified:       []string{},
			},
			expected: false,
		},
		{
			name: "performance improvement",
			comparison: &PipelineExecutionComparison{
				BeforeJobsExecuted: []string{"build", "test"},
				AfterJobsExecuted:  []string{"build", "test"},
				JobsAdded:          []string{},
				JobsRemoved:        []string{},
				JobsModified:       []string{"test"}, // Performance improvement
			},
			expected: true, // Allow performance improvements
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := validator.determineBehavioralEquivalence(tc.comparison)
			if result != tc.expected {
				t.Errorf("Expected %v, got %v", tc.expected, result)
			}
		})
	}
}

func TestValidatorWithGitLabClient(t *testing.T) {
	// Create a simulation client
	client, err := gitlab.NewClient(gitlab.BackendSimulation, nil)
	if err != nil {
		t.Fatalf("Failed to create simulation client: %v", err)
	}

	validator := NewRefactoringValidator()
	validator.SetGitLabClient(client)

	// Create test directories
	beforeDir, afterDir := createTestDirs(t)
	defer cleanupTestDirs(beforeDir, afterDir)

	result, err := validator.CompareConfigurations(beforeDir, afterDir)
	if err != nil {
		t.Fatalf("CompareConfigurations failed: %v", err)
	}

	// Should have results
	if result == nil {
		t.Fatal("Expected validation result, got nil")
	}

	// Should use GitLab client for comparison
	if result.PipelineComparison == nil {
		t.Error("Expected pipeline comparison, got nil")
	}
}

// Mock implementations for testing

type MockGitLabClient struct{}

func (m *MockGitLabClient) ValidateConfig(ctx interface{}, yaml string, projectID int) (*gitlab.ValidationResult, error) {
	return &gitlab.ValidationResult{Valid: true}, nil
}

func (m *MockGitLabClient) LintConfig(ctx interface{}, yaml string) (*gitlab.ValidationResult, error) {
	return &gitlab.ValidationResult{Valid: true}, nil
}

func (m *MockGitLabClient) CreatePipeline(ctx interface{}, projectID int, ref string, variables map[string]string) (*gitlab.Pipeline, error) {
	return &gitlab.Pipeline{ID: 1, Status: "running", Ref: ref}, nil
}

func (m *MockGitLabClient) GetPipeline(ctx interface{}, projectID, pipelineID int) (*gitlab.Pipeline, error) {
	return &gitlab.Pipeline{ID: pipelineID, Status: "success", Ref: "main"}, nil
}

func (m *MockGitLabClient) GetPipelineJobs(ctx interface{}, projectID, pipelineID int) ([]*gitlab.Job, error) {
	return []*gitlab.Job{
		{ID: 1, Name: "build", Status: "success", Duration: 120},
		{ID: 2, Name: "test", Status: "success", Duration: 180},
		{ID: 3, Name: "deploy", Status: "success", Duration: 90},
	}, nil
}

func (m *MockGitLabClient) CancelPipeline(ctx interface{}, projectID, pipelineID int) error {
	return nil
}

func (m *MockGitLabClient) RetryPipeline(ctx interface{}, projectID, pipelineID int) (*gitlab.Pipeline, error) {
	return &gitlab.Pipeline{ID: pipelineID + 1, Status: "running", Ref: "main"}, nil
}

func (m *MockGitLabClient) GetJob(ctx interface{}, projectID, jobID int) (*gitlab.Job, error) {
	return &gitlab.Job{ID: jobID, Name: "test-job", Status: "success", Duration: 60}, nil
}

func (m *MockGitLabClient) GetJobLog(ctx interface{}, projectID, jobID int) (string, error) {
	return "mock job log", nil
}

func (m *MockGitLabClient) GetJobArtifacts(ctx interface{}, projectID, jobID int) ([]byte, error) {
	return []byte("mock artifacts"), nil
}

func (m *MockGitLabClient) RetryJob(ctx interface{}, projectID, jobID int) (*gitlab.Job, error) {
	return &gitlab.Job{ID: jobID, Name: "test-job", Status: "running"}, nil
}

func (m *MockGitLabClient) CancelJob(ctx interface{}, projectID, jobID int) (*gitlab.Job, error) {
	return &gitlab.Job{ID: jobID, Name: "test-job", Status: "canceled"}, nil
}

func (m *MockGitLabClient) GetProject(ctx interface{}, projectID int) (*gitlab.Project, error) {
	return &gitlab.Project{ID: projectID, Name: "test-project"}, nil
}

func (m *MockGitLabClient) WaitForPipeline(ctx interface{}, projectID, pipelineID int, timeout time.Duration) (*gitlab.Pipeline, error) {
	return &gitlab.Pipeline{ID: pipelineID, Status: "success", Ref: "main"}, nil
}

func (m *MockGitLabClient) WaitForJob(ctx interface{}, projectID, jobID int, timeout time.Duration) (*gitlab.Job, error) {
	return &gitlab.Job{ID: jobID, Name: "test-job", Status: "success"}, nil
}

func (m *MockGitLabClient) HealthCheck(ctx interface{}) error {
	return nil
}

// testDirExists checks if directory exists
func testDirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// createTestDirs creates temporary directories with minimal CI configurations for testing
func createTestDirs(t *testing.T) (string, string) {
	beforeDir := t.TempDir()
	afterDir := t.TempDir()

	// Create a simple CI config for before
	beforeConfig := `
stages:
  - build
  - test

build:
  stage: build
  script:
    - echo "Building"

test:
  stage: test
  script:
    - echo "Testing"
`

	// Create a similar CI config for after
	afterConfig := `
stages:
  - build
  - test

build:
  stage: build
  script:
    - echo "Building optimized"

test:
  stage: test
  script:
    - echo "Testing optimized"
`

	// Write CI configs to directories
	os.WriteFile(filepath.Join(beforeDir, ".gitlab-ci.yml"), []byte(beforeConfig), 0644)
	os.WriteFile(filepath.Join(afterDir, ".gitlab-ci.yml"), []byte(afterConfig), 0644)

	return beforeDir, afterDir
}

// cleanupTestDirs removes temporary directories (though Go's t.TempDir() auto-cleans)
func cleanupTestDirs(beforeDir, afterDir string) {
	// t.TempDir() automatically cleans up, but we could add manual cleanup if needed
}