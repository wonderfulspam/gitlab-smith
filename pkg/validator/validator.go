package validator

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/wonderfulspam/gitlab-smith/pkg/analyzer"
	"github.com/wonderfulspam/gitlab-smith/pkg/deployer"
	"github.com/wonderfulspam/gitlab-smith/pkg/differ"
	"github.com/wonderfulspam/gitlab-smith/pkg/parser"
	"github.com/wonderfulspam/gitlab-smith/pkg/renderer"
)

// DeployerInterface defines the interface for deployers to enable mocking
type DeployerInterface interface {
	GetStatus() (*deployer.DeploymentStatus, error)
	Deploy() error
	Destroy() error
}

// GitLabClientInterface defines the interface for GitLab clients to enable mocking
type GitLabClientInterface interface {
	CreateProject(name, path string) (*Project, error)
	GetProject(path string) (*Project, error)
	DeleteProject(projectID int) error
	CreateFile(projectID int, filePath, content, commitMessage string) error
	TriggerPipeline(projectID int, ref string) (*Pipeline, error)
	GetPipeline(projectID, pipelineID int) (*Pipeline, error)
	GetPipelineJobs(projectID, pipelineID int) ([]Job, error)
	WaitForPipelineCompletion(projectID, pipelineID int, timeout time.Duration) (*Pipeline, error)
}

// RefactoringResult contains the validation results
type RefactoringResult struct {
	ActualChanges        *differ.DiffResult
	AnalysisImprovement  int
	PipelineComparison   *renderer.PipelineComparison
	BehavioralValidation *BehavioralValidationResult
}

// BehavioralValidationResult contains results from full behavioral testing
type BehavioralValidationResult struct {
	BeforeExecutionPassed bool
	AfterExecutionPassed  bool
	BehaviorEquivalent    bool
	ExecutionComparison   *PipelineExecutionComparison
	ValidationErrors      []string
}

// PipelineExecutionComparison contains comparison of pipeline executions
type PipelineExecutionComparison struct {
	BeforeJobsExecuted   []string
	AfterJobsExecuted    []string
	JobsAdded            []string
	JobsRemoved          []string
	JobsModified         []string
	ExecutionTimesBefore map[string]int64
	ExecutionTimesAfter  map[string]int64
}

// RefactoringValidator performs GitLab CI refactoring analysis
type RefactoringValidator struct {
	deployer           DeployerInterface
	fullTestingEnabled bool
	gitlabClient       GitLabClientInterface
}

// NewRefactoringValidator creates a new refactoring validator
func NewRefactoringValidator() *RefactoringValidator {
	return &RefactoringValidator{
		fullTestingEnabled: false,
	}
}

// NewRefactoringValidatorWithFullTesting creates a validator with full behavioral testing capabilities
func NewRefactoringValidatorWithFullTesting(deployerConfig *deployer.DeploymentConfig) *RefactoringValidator {
	return &RefactoringValidator{
		deployer:           deployer.New(deployerConfig),
		fullTestingEnabled: true,
	}
}

// EnableFullTesting enables full behavioral testing mode
func (rv *RefactoringValidator) EnableFullTesting(deployerConfig *deployer.DeploymentConfig) {
	rv.deployer = deployer.New(deployerConfig)
	rv.fullTestingEnabled = true

	// Initialize GitLab client for local instance
	baseURL := fmt.Sprintf("http://%s:%s", deployerConfig.ExternalHostname, deployerConfig.HTTPPort)
	rv.gitlabClient = NewGitLabClient(baseURL, "test-token")
}

// CompareConfigurations compares before and after GitLab CI configurations
func (rv *RefactoringValidator) CompareConfigurations(beforeDir, afterDir string) (*RefactoringResult, error) {
	// Parse before and after configurations
	beforeConfig, err := rv.parseConfiguration(beforeDir)
	if err != nil {
		return nil, fmt.Errorf("failed to parse before config: %w", err)
	}

	afterConfig, err := rv.parseConfiguration(afterDir)
	if err != nil {
		return nil, fmt.Errorf("failed to parse after config: %w", err)
	}

	result := &RefactoringResult{}

	// Perform semantic diff
	result.ActualChanges = differ.Compare(beforeConfig, afterConfig)

	// Analyze both configurations
	beforeAnalysis := analyzer.Analyze(beforeConfig)
	afterAnalysis := analyzer.Analyze(afterConfig)
	result.AnalysisImprovement = beforeAnalysis.TotalIssues - afterAnalysis.TotalIssues

	// Compare pipeline executions
	renderer := renderer.New(nil)
	pipelineComparison, err := renderer.CompareConfigurations(beforeConfig, afterConfig)
	if err != nil {
		return result, fmt.Errorf("pipeline comparison failed: %w", err)
	}
	result.PipelineComparison = pipelineComparison

	// Perform full behavioral testing if enabled
	if rv.fullTestingEnabled && rv.deployer != nil {
		behavioralResult, err := rv.performBehavioralValidation(beforeDir, afterDir)
		if err != nil {
			return result, fmt.Errorf("behavioral validation failed: %w", err)
		}
		result.BehavioralValidation = behavioralResult
	}

	return result, nil
}

// parseConfiguration parses a GitLab CI configuration from a directory
func (rv *RefactoringValidator) parseConfiguration(configDir string) (*parser.GitLabConfig, error) {
	// Look for main CI file
	mainFiles := []string{".gitlab-ci.yml", ".gitlab-ci.yaml", "gitlab-ci.yml", "gitlab-ci.yaml"}

	var mainFile string
	for _, filename := range mainFiles {
		path := filepath.Join(configDir, filename)
		if _, err := os.Stat(path); err == nil {
			mainFile = path
			break
		}
	}

	if mainFile == "" {
		return nil, fmt.Errorf("no GitLab CI main file found in %s", configDir)
	}

	// Parse the configuration with includes
	config, err := parser.ParseFile(mainFile)
	if err != nil {
		return nil, fmt.Errorf("failed to parse configuration: %w", err)
	}

	return config, nil
}

// performBehavioralValidation runs full behavioral testing using local GitLab
func (rv *RefactoringValidator) performBehavioralValidation(beforeDir, afterDir string) (*BehavioralValidationResult, error) {
	result := &BehavioralValidationResult{
		ValidationErrors: []string{},
	}

	// Deploy GitLab if not already running
	status, err := rv.deployer.GetStatus()
	if err != nil {
		return nil, fmt.Errorf("failed to check deployer status: %w", err)
	}

	if !status.IsRunning {
		fmt.Printf("Deploying GitLab instance for behavioral testing...\n")
		if err := rv.deployer.Deploy(); err != nil {
			return nil, fmt.Errorf("failed to deploy GitLab: %w", err)
		}
	}

	// Test before configuration
	beforeResult, err := rv.testConfiguration(beforeDir, "before")
	if err != nil {
		result.ValidationErrors = append(result.ValidationErrors, fmt.Sprintf("Before config test failed: %v", err))
		result.BeforeExecutionPassed = false
	} else {
		result.BeforeExecutionPassed = beforeResult.ExecutionPassed
	}

	// Test after configuration
	afterResult, err := rv.testConfiguration(afterDir, "after")
	if err != nil {
		result.ValidationErrors = append(result.ValidationErrors, fmt.Sprintf("After config test failed: %v", err))
		result.AfterExecutionPassed = false
	} else {
		result.AfterExecutionPassed = afterResult.ExecutionPassed
	}

	// Compare executions if both passed
	if result.BeforeExecutionPassed && result.AfterExecutionPassed {
		result.ExecutionComparison = rv.compareExecutions(beforeResult, afterResult)
		result.BehaviorEquivalent = rv.determineBehavioralEquivalence(result.ExecutionComparison)
	} else {
		result.BehaviorEquivalent = false
	}

	return result, nil
}

// ConfigurationTestResult contains results from testing a single configuration
type ConfigurationTestResult struct {
	ExecutionPassed bool
	JobsExecuted    []string
	ExecutionTimes  map[string]int64
	PipelineID      string
}

// testConfiguration tests a single GitLab CI configuration against the local GitLab instance
func (rv *RefactoringValidator) testConfiguration(configDir, testName string) (*ConfigurationTestResult, error) {
	// Create or get test project
	projectName := fmt.Sprintf("gitlab-smith-test-%s", testName)
	project, err := rv.gitlabClient.CreateProject(projectName, projectName)
	if err != nil {
		// Try to get existing project if creation failed
		existingProject, getErr := rv.gitlabClient.GetProject(projectName)
		if getErr != nil {
			return nil, fmt.Errorf("failed to create or get project: %w", err)
		}
		project = existingProject
	}

	// Read and upload CI configuration
	ciConfigPath, err := rv.findCIConfigFile(configDir)
	if err != nil {
		return nil, fmt.Errorf("failed to find CI config: %w", err)
	}

	ciContent, err := os.ReadFile(ciConfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read CI config: %w", err)
	}

	// Upload CI configuration to project
	err = rv.gitlabClient.CreateFile(project.ID, ".gitlab-ci.yml", string(ciContent), fmt.Sprintf("Add CI config for %s test", testName))
	if err != nil {
		return nil, fmt.Errorf("failed to upload CI config: %w", err)
	}

	// Upload any additional files from the config directory
	err = rv.uploadConfigFiles(project.ID, configDir)
	if err != nil {
		return nil, fmt.Errorf("failed to upload config files: %w", err)
	}

	// Trigger pipeline
	pipeline, err := rv.gitlabClient.TriggerPipeline(project.ID, "main")
	if err != nil {
		return nil, fmt.Errorf("failed to trigger pipeline: %w", err)
	}

	// Wait for pipeline completion
	completedPipeline, err := rv.gitlabClient.WaitForPipelineCompletion(project.ID, pipeline.ID, 10*time.Minute)
	if err != nil {
		return nil, fmt.Errorf("pipeline did not complete: %w", err)
	}

	// Get pipeline jobs
	jobs, err := rv.gitlabClient.GetPipelineJobs(project.ID, pipeline.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get pipeline jobs: %w", err)
	}

	// Process results
	result := &ConfigurationTestResult{
		ExecutionPassed: completedPipeline.Status == "success",
		JobsExecuted:    []string{},
		ExecutionTimes:  make(map[string]int64),
		PipelineID:      fmt.Sprintf("%d", pipeline.ID),
	}

	for _, job := range jobs {
		result.JobsExecuted = append(result.JobsExecuted, job.Name)
		result.ExecutionTimes[job.Name] = int64(job.Duration)
	}

	// Cleanup test project
	go func() {
		time.Sleep(5 * time.Minute) // Keep project for debugging for 5 minutes
		rv.gitlabClient.DeleteProject(project.ID)
	}()

	return result, nil
}

// findCIConfigFile finds the main CI configuration file in a directory
func (rv *RefactoringValidator) findCIConfigFile(configDir string) (string, error) {
	mainFiles := []string{".gitlab-ci.yml", ".gitlab-ci.yaml", "gitlab-ci.yml", "gitlab-ci.yaml"}

	for _, filename := range mainFiles {
		path := filepath.Join(configDir, filename)
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	return "", fmt.Errorf("no GitLab CI configuration file found in %s", configDir)
}

// uploadConfigFiles uploads all configuration files from a directory to the GitLab project
func (rv *RefactoringValidator) uploadConfigFiles(projectID int, configDir string) error {
	return filepath.Walk(configDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and hidden files (except .gitlab-ci.yml which is already uploaded)
		if info.IsDir() || (filepath.Base(path)[0] == '.' && filepath.Base(path) != ".gitlab-ci.yml") {
			return nil
		}

		// Skip the main CI file as it's already uploaded
		if filepath.Base(path) == ".gitlab-ci.yml" {
			return nil
		}

		// Calculate relative path
		relPath, err := filepath.Rel(configDir, path)
		if err != nil {
			return err
		}

		// Read file content
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		// Upload to GitLab
		return rv.gitlabClient.CreateFile(projectID, relPath, string(content), fmt.Sprintf("Add config file: %s", relPath))
	})
}

// compareExecutions compares the execution results of before and after configurations
func (rv *RefactoringValidator) compareExecutions(before, after *ConfigurationTestResult) *PipelineExecutionComparison {
	comparison := &PipelineExecutionComparison{
		BeforeJobsExecuted:   before.JobsExecuted,
		AfterJobsExecuted:    after.JobsExecuted,
		ExecutionTimesBefore: before.ExecutionTimes,
		ExecutionTimesAfter:  after.ExecutionTimes,
	}

	// Find jobs that were added or removed
	beforeJobsSet := make(map[string]bool)
	afterJobsSet := make(map[string]bool)

	for _, job := range before.JobsExecuted {
		beforeJobsSet[job] = true
	}

	for _, job := range after.JobsExecuted {
		afterJobsSet[job] = true
	}

	for job := range afterJobsSet {
		if !beforeJobsSet[job] {
			comparison.JobsAdded = append(comparison.JobsAdded, job)
		}
	}

	for job := range beforeJobsSet {
		if !afterJobsSet[job] {
			comparison.JobsRemoved = append(comparison.JobsRemoved, job)
		}
	}

	// Find modified jobs (different execution times)
	for job := range beforeJobsSet {
		if afterJobsSet[job] {
			beforeTime := before.ExecutionTimes[job]
			afterTime := after.ExecutionTimes[job]

			// Consider significant time differences as modifications
			timeDifference := beforeTime - afterTime
			if timeDifference < 0 {
				timeDifference = -timeDifference
			}

			// If difference is more than 10% of original time, consider it modified
			if float64(timeDifference)/float64(beforeTime) > 0.1 {
				comparison.JobsModified = append(comparison.JobsModified, job)
			}
		}
	}

	return comparison
}

// determineBehavioralEquivalence determines if the before and after configurations are behaviorally equivalent
func (rv *RefactoringValidator) determineBehavioralEquivalence(comparison *PipelineExecutionComparison) bool {
	// Configurations are considered equivalent if:
	// 1. Same jobs were executed (no additions or removals)
	// 2. Job modifications are within acceptable thresholds

	if len(comparison.JobsAdded) > 0 || len(comparison.JobsRemoved) > 0 {
		return false
	}

	// Minor performance differences are acceptable for equivalence
	// This is a simplified check - real implementation might be more sophisticated
	return len(comparison.JobsModified) == 0
}
