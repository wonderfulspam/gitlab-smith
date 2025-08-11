package validator

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/wonderfulspam/gitlab-smith/pkg/deployer"
)

// Enhanced mock implementations for more detailed testing

type TestDeployer struct {
	isRunning bool
	deployErr error
	statusErr error
}

func (m *TestDeployer) GetStatus() (*deployer.DeploymentStatus, error) {
	if m.statusErr != nil {
		return nil, m.statusErr
	}
	return &deployer.DeploymentStatus{IsRunning: m.isRunning}, nil
}

func (m *TestDeployer) Deploy() error {
	return m.deployErr
}

func (m *TestDeployer) Destroy() error {
	return nil
}

type TestGitLabClient struct {
	projects       map[string]*Project
	nextProjectID  int
	nextPipelineID int
	createFileErr  error
	pipelineStatus string
	jobs           []Job
}

func NewTestGitLabClient() *TestGitLabClient {
	return &TestGitLabClient{
		projects:       make(map[string]*Project),
		nextProjectID:  1,
		pipelineStatus: "success",
		jobs:           []Job{},
	}
}

func (m *TestGitLabClient) CreateProject(name, path string) (*Project, error) {
	project := &Project{
		ID:   m.nextProjectID,
		Name: name,
		Path: path,
	}
	m.projects[path] = project
	m.nextProjectID++
	return project, nil
}

func (m *TestGitLabClient) GetProject(path string) (*Project, error) {
	if project, exists := m.projects[path]; exists {
		return project, nil
	}
	return nil, os.ErrNotExist
}

func (m *TestGitLabClient) DeleteProject(projectID int) error {
	return nil
}

func (m *TestGitLabClient) CreateFile(projectID int, filePath, content, commitMessage string) error {
	return m.createFileErr
}

func (m *TestGitLabClient) TriggerPipeline(projectID int, ref string) (*Pipeline, error) {
	m.nextPipelineID++
	return &Pipeline{
		ID:     m.nextPipelineID,
		Status: "running",
		Ref:    ref,
	}, nil
}

func (m *TestGitLabClient) GetPipeline(projectID, pipelineID int) (*Pipeline, error) {
	return &Pipeline{
		ID:     pipelineID,
		Status: m.pipelineStatus,
		Ref:    "main",
	}, nil
}

func (m *TestGitLabClient) GetPipelineJobs(projectID, pipelineID int) ([]Job, error) {
	return m.jobs, nil
}

func (m *TestGitLabClient) WaitForPipelineCompletion(projectID, pipelineID int, timeout time.Duration) (*Pipeline, error) {
	return &Pipeline{
		ID:     pipelineID,
		Status: m.pipelineStatus,
		Ref:    "main",
	}, nil
}

// Test functions

func TestNewRefactoringValidator(t *testing.T) {
	validator := NewRefactoringValidator()

	if validator == nil {
		t.Error("Expected validator to be created, got nil")
	}

	if validator.fullTestingEnabled {
		t.Error("Expected fullTestingEnabled to be false by default")
	}
}

func TestNewRefactoringValidatorWithFullTesting(t *testing.T) {
	config := &deployer.DeploymentConfig{
		ExternalHostname: "localhost",
		HTTPPort:         "8080",
	}

	validator := NewRefactoringValidatorWithFullTesting(config)

	if validator == nil {
		t.Error("Expected validator to be created, got nil")
	}

	if !validator.fullTestingEnabled {
		t.Error("Expected fullTestingEnabled to be true")
	}

	if validator.deployer == nil {
		t.Error("Expected deployer to be set")
	}
}

func TestEnableFullTesting(t *testing.T) {
	validator := NewRefactoringValidator()

	config := &deployer.DeploymentConfig{
		ExternalHostname: "localhost",
		HTTPPort:         "8080",
	}

	validator.EnableFullTesting(config)

	if !validator.fullTestingEnabled {
		t.Error("Expected fullTestingEnabled to be true after enabling")
	}

	if validator.deployer == nil {
		t.Error("Expected deployer to be set after enabling full testing")
	}

	if validator.gitlabClient == nil {
		t.Error("Expected GitLab client to be set after enabling full testing")
	}
}

func TestFindCIConfigFile(t *testing.T) {
	// Create temporary directory for testing
	tmpDir := t.TempDir()

	validator := &RefactoringValidator{}

	t.Run("no config file found", func(t *testing.T) {
		_, err := validator.findCIConfigFile(tmpDir)
		if err == nil {
			t.Error("Expected error when no config file found")
		}
	})

	t.Run("find .gitlab-ci.yml", func(t *testing.T) {
		configFile := filepath.Join(tmpDir, ".gitlab-ci.yml")
		err := os.WriteFile(configFile, []byte("stages: [build, test]"), 0644)
		if err != nil {
			t.Fatalf("Failed to create test config file: %v", err)
		}

		foundPath, err := validator.findCIConfigFile(tmpDir)
		if err != nil {
			t.Errorf("Expected to find config file, got error: %v", err)
		}

		if foundPath != configFile {
			t.Errorf("Expected path %s, got %s", configFile, foundPath)
		}
	})

	t.Run("find .gitlab-ci.yaml", func(t *testing.T) {
		// Remove .yml file and create .yaml file
		os.Remove(filepath.Join(tmpDir, ".gitlab-ci.yml"))

		configFile := filepath.Join(tmpDir, ".gitlab-ci.yaml")
		err := os.WriteFile(configFile, []byte("stages: [build, test]"), 0644)
		if err != nil {
			t.Fatalf("Failed to create test config file: %v", err)
		}

		foundPath, err := validator.findCIConfigFile(tmpDir)
		if err != nil {
			t.Errorf("Expected to find config file, got error: %v", err)
		}

		if foundPath != configFile {
			t.Errorf("Expected path %s, got %s", configFile, foundPath)
		}
	})
}

func TestCompareExecutions(t *testing.T) {
	validator := &RefactoringValidator{}

	before := &ConfigurationTestResult{
		ExecutionPassed: true,
		JobsExecuted:    []string{"build", "test", "deploy"},
		ExecutionTimes: map[string]int64{
			"build":  100,
			"test":   200,
			"deploy": 150,
		},
	}

	after := &ConfigurationTestResult{
		ExecutionPassed: true,
		JobsExecuted:    []string{"build", "test", "package"},
		ExecutionTimes: map[string]int64{
			"build":   90,
			"test":    225, // 25/200 = 12.5% > 10%
			"package": 80,
		},
	}

	comparison := validator.compareExecutions(before, after)

	if len(comparison.JobsAdded) != 1 || comparison.JobsAdded[0] != "package" {
		t.Errorf("Expected 1 added job 'package', got %v", comparison.JobsAdded)
	}

	if len(comparison.JobsRemoved) != 1 || comparison.JobsRemoved[0] != "deploy" {
		t.Errorf("Expected 1 removed job 'deploy', got %v", comparison.JobsRemoved)
	}

	// Test job should be modified due to execution time difference > 10%
	// 225 vs 200 is 12.5% difference, which should trigger modification detection
	// Let's check if the test job was detected as modified
	foundTestModified := false
	for _, job := range comparison.JobsModified {
		if job == "test" {
			foundTestModified = true
			break
		}
	}
	if !foundTestModified {
		t.Errorf("Expected 'test' job to be modified (225 vs 200 is 12.5%% difference), got modified jobs: %v", comparison.JobsModified)
	}

	if len(comparison.BeforeJobsExecuted) != 3 {
		t.Errorf("Expected 3 before jobs, got %d", len(comparison.BeforeJobsExecuted))
	}

	if len(comparison.AfterJobsExecuted) != 3 {
		t.Errorf("Expected 3 after jobs, got %d", len(comparison.AfterJobsExecuted))
	}
}

func TestDetermineBehavioralEquivalence(t *testing.T) {
	validator := &RefactoringValidator{}

	t.Run("equivalent - no changes", func(t *testing.T) {
		comparison := &PipelineExecutionComparison{
			JobsAdded:    []string{},
			JobsRemoved:  []string{},
			JobsModified: []string{},
		}

		equivalent := validator.determineBehavioralEquivalence(comparison)
		if !equivalent {
			t.Error("Expected configurations to be equivalent when no changes")
		}
	})

	t.Run("not equivalent - jobs added", func(t *testing.T) {
		comparison := &PipelineExecutionComparison{
			JobsAdded:    []string{"new-job"},
			JobsRemoved:  []string{},
			JobsModified: []string{},
		}

		equivalent := validator.determineBehavioralEquivalence(comparison)
		if equivalent {
			t.Error("Expected configurations to be non-equivalent when jobs added")
		}
	})

	t.Run("not equivalent - jobs removed", func(t *testing.T) {
		comparison := &PipelineExecutionComparison{
			JobsAdded:    []string{},
			JobsRemoved:  []string{"old-job"},
			JobsModified: []string{},
		}

		equivalent := validator.determineBehavioralEquivalence(comparison)
		if equivalent {
			t.Error("Expected configurations to be non-equivalent when jobs removed")
		}
	})

	t.Run("not equivalent - jobs modified", func(t *testing.T) {
		comparison := &PipelineExecutionComparison{
			JobsAdded:    []string{},
			JobsRemoved:  []string{},
			JobsModified: []string{"changed-job"},
		}

		equivalent := validator.determineBehavioralEquivalence(comparison)
		if equivalent {
			t.Error("Expected configurations to be non-equivalent when jobs modified")
		}
	})
}

func TestUploadConfigFiles(t *testing.T) {
	// Create temporary directory with test files
	tmpDir := t.TempDir()

	// Create test files
	testFiles := map[string]string{
		".gitlab-ci.yml":    "stages: [build]",
		"ci/build.yml":      "build: script: echo build",
		"scripts/deploy.sh": "#!/bin/bash\necho deploy",
		".hidden":           "should be ignored",
	}

	for filePath, content := range testFiles {
		fullPath := filepath.Join(tmpDir, filePath)
		dir := filepath.Dir(fullPath)

		// Create directory if it doesn't exist
		if dir != tmpDir {
			err := os.MkdirAll(dir, 0755)
			if err != nil {
				t.Fatalf("Failed to create directory %s: %v", dir, err)
			}
		}

		err := os.WriteFile(fullPath, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file %s: %v", fullPath, err)
		}
	}

	validator := &RefactoringValidator{}
	mockClient := NewTestGitLabClient()
	validator.gitlabClient = mockClient

	err := validator.uploadConfigFiles(1, tmpDir)
	if err != nil {
		t.Errorf("Expected no error uploading config files, got: %v", err)
	}
}

func TestUploadConfigFilesWithError(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a test file
	testFile := filepath.Join(tmpDir, "test.yml")
	err := os.WriteFile(testFile, []byte("test: content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	validator := &RefactoringValidator{}
	mockClient := NewTestGitLabClient()
	mockClient.createFileErr = errors.New("upload error")
	validator.gitlabClient = mockClient

	err = validator.uploadConfigFiles(1, tmpDir)
	if err == nil {
		t.Error("Expected error when upload fails, got nil")
	}
}

func TestPerformBehavioralValidation(t *testing.T) {
	// Create temporary directories for before/after configs
	beforeDir := t.TempDir()
	afterDir := t.TempDir()

	// Create minimal config files
	beforeConfig := filepath.Join(beforeDir, ".gitlab-ci.yml")
	afterConfig := filepath.Join(afterDir, ".gitlab-ci.yml")

	configContent := `stages:
  - build
  - test

build-job:
  stage: build
  script:
    - echo "Building..."

test-job:
  stage: test
  script:
    - echo "Testing..."`

	err := os.WriteFile(beforeConfig, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create before config: %v", err)
	}

	err = os.WriteFile(afterConfig, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create after config: %v", err)
	}

	validator := &RefactoringValidator{}

	// Test with deployer error
	mockDeployer := &TestDeployer{statusErr: errors.New("status error")}
	validator.deployer = mockDeployer

	_, err = validator.performBehavioralValidation(beforeDir, afterDir)
	if err == nil {
		t.Error("Expected error when deployer status fails")
	}

	// Test with successful deployment status check
	mockDeployer = &TestDeployer{isRunning: true}
	validator.deployer = mockDeployer
	validator.gitlabClient = NewTestGitLabClient()

	result, err := validator.performBehavioralValidation(beforeDir, afterDir)
	if err != nil {
		t.Errorf("Expected no error with successful setup, got: %v", err)
	}

	if result == nil {
		t.Error("Expected result to be returned")
	}

	if !result.BeforeExecutionPassed {
		t.Error("Expected before execution to pass with mock setup")
	}

	if !result.AfterExecutionPassed {
		t.Error("Expected after execution to pass with mock setup")
	}

	if !result.BehaviorEquivalent {
		t.Error("Expected behaviors to be equivalent with identical configs")
	}
}
