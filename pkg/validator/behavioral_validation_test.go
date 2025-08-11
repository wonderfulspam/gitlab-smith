package validator

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/wonderfulspam/gitlab-smith/pkg/deployer"
)

func TestBehavioralValidation_BasicFlow(t *testing.T) {
	// Skip this test by default as it requires Docker and significant setup
	if testing.Short() {
		t.Skip("Skipping behavioral validation test in short mode")
	}

	// Create validator with full testing enabled
	config := deployer.DefaultConfig()
	config.ContainerName = "gitlab-smith-test-behavioral"
	validator := NewRefactoringValidatorWithFullTesting(config)

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

	// Check that behavioral validation was performed
	if result.BehavioralValidation == nil {
		t.Fatal("Expected behavioral validation results, got nil")
	}

	// Verify the behavioral validation structure
	bv := result.BehavioralValidation
	if bv.ExecutionComparison == nil {
		t.Error("Expected execution comparison, got nil")
	}

	// Clean up deployer if it was started
	if validator.deployer != nil {
		validator.deployer.Destroy()
	}
}

func TestBehavioralValidation_MockedExecution(t *testing.T) {
	// Test the behavioral validation logic with mocked execution
	validator := &RefactoringValidator{
		fullTestingEnabled: true,
	}

	// Create mock deployer that implements the interface properly
	mockDeployer := NewMockDeployer()
	validator.deployer = mockDeployer
	validator.gitlabClient = &MockGitLabClient{}

	// Create temporary test directories with CI configs
	beforeDir, afterDir := createTestDirs(t)
	defer cleanupTestDirs(beforeDir, afterDir)

	result, err := validator.performBehavioralValidation(beforeDir, afterDir)
	if err != nil {
		t.Fatalf("performBehavioralValidation failed: %v", err)
	}

	// Verify results structure
	if result == nil {
		t.Fatal("Expected behavioral validation result, got nil")
	}

	if result.ExecutionComparison == nil {
		t.Error("Expected execution comparison, got nil")
	}

	// Test behavioral equivalence determination
	if !result.BehaviorEquivalent {
		t.Error("Expected mock configurations to be behaviorally equivalent")
	}
}

func TestExecutionComparison(t *testing.T) {
	validator := &RefactoringValidator{}

	before := &ConfigurationTestResult{
		ExecutionPassed: true,
		JobsExecuted:    []string{"build", "test", "deploy"},
		ExecutionTimes:  map[string]int64{"build": 120, "test": 180, "deploy": 90},
	}

	after := &ConfigurationTestResult{
		ExecutionPassed: true,
		JobsExecuted:    []string{"build", "test-optimized", "deploy"},
		ExecutionTimes:  map[string]int64{"build": 120, "test-optimized": 150, "deploy": 90},
	}

	comparison := validator.compareExecutions(before, after)

	// Check that job changes are detected correctly
	if len(comparison.JobsRemoved) != 1 || comparison.JobsRemoved[0] != "test" {
		t.Errorf("Expected 'test' job to be removed, got: %v", comparison.JobsRemoved)
	}

	if len(comparison.JobsAdded) != 1 || comparison.JobsAdded[0] != "test-optimized" {
		t.Errorf("Expected 'test-optimized' job to be added, got: %v", comparison.JobsAdded)
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
			name: "job modified",
			comparison: &PipelineExecutionComparison{
				BeforeJobsExecuted: []string{"build", "test"},
				AfterJobsExecuted:  []string{"build", "test"},
				JobsAdded:          []string{},
				JobsRemoved:        []string{},
				JobsModified:       []string{"test"},
			},
			expected: false,
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

// Mock implementations for testing

// MockDeployer embeds the real deployer but overrides specific methods for testing
type MockDeployer struct {
	*deployer.Deployer
	isRunning bool
}

func NewMockDeployer() *MockDeployer {
	return &MockDeployer{
		Deployer:  deployer.New(nil),
		isRunning: true,
	}
}

func (m *MockDeployer) GetStatus() (*deployer.DeploymentStatus, error) {
	return &deployer.DeploymentStatus{
		IsRunning: m.isRunning,
		URL:       "http://localhost:8080",
	}, nil
}

func (m *MockDeployer) Deploy() error  { m.isRunning = true; return nil }
func (m *MockDeployer) Destroy() error { m.isRunning = false; return nil }

type MockGitLabClient struct{}

func (m *MockGitLabClient) CreateProject(name, path string) (*Project, error) {
	return &Project{ID: 1, Name: name, Path: path}, nil
}

func (m *MockGitLabClient) GetProject(path string) (*Project, error) {
	return &Project{ID: 1, Name: path, Path: path}, nil
}

func (m *MockGitLabClient) DeleteProject(projectID int) error { return nil }

func (m *MockGitLabClient) CreateFile(projectID int, filePath, content, commitMessage string) error {
	return nil
}

func (m *MockGitLabClient) TriggerPipeline(projectID int, ref string) (*Pipeline, error) {
	return &Pipeline{ID: 1, Status: "running", Ref: ref}, nil
}

func (m *MockGitLabClient) GetPipeline(projectID, pipelineID int) (*Pipeline, error) {
	return &Pipeline{ID: pipelineID, Status: "success", Ref: "main"}, nil
}

func (m *MockGitLabClient) GetPipelineJobs(projectID, pipelineID int) ([]Job, error) {
	return []Job{
		{ID: 1, Name: "build", Status: "success", Duration: 120},
		{ID: 2, Name: "test", Status: "success", Duration: 180},
		{ID: 3, Name: "deploy", Status: "success", Duration: 90},
	}, nil
}

func (m *MockGitLabClient) WaitForPipelineCompletion(projectID, pipelineID int, timeout time.Duration) (*Pipeline, error) {
	return &Pipeline{ID: pipelineID, Status: "success", Ref: "main"}, nil
}

// testDirExists checks if directory exists (renamed to avoid conflict with existing function)
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
