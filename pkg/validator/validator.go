package validator

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

// SetDeployer sets an existing deployer instance (for reusing already deployed GitLab)
func (rv *RefactoringValidator) SetDeployer(deployer DeployerInterface) {
	rv.deployer = deployer
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

	// Compare pipeline executions - use GitLab API if available
	var pipelineComparison *renderer.PipelineComparison

	if rv.fullTestingEnabled && rv.gitlabClient != nil {
		// Use GitLab API for actual pipeline rendering
		fmt.Printf("Using GitLab API for pipeline rendering (full testing mode)\n")
		rendererInstance := rv.createRendererWithGitLabAPI()
		pipelineComparison, err = rv.comparePipelinesViaAPI(rendererInstance, beforeConfig, afterConfig)
	} else {
		// Use static simulation
		fmt.Printf("Using static simulation for pipeline rendering\n")
		rendererInstance := renderer.New(nil)
		pipelineComparison, err = rendererInstance.CompareConfigurations(beforeConfig, afterConfig)
	}

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

// createRendererWithGitLabAPI creates a renderer configured to use the GitLab API
func (rv *RefactoringValidator) createRendererWithGitLabAPI() *renderer.Renderer {
	if rv.gitlabClient == nil {
		return renderer.New(nil)
	}

	// Extract GitLab client details for renderer
	baseURL := rv.gitlabClient.(*GitLabClient).baseURL
	token := rv.gitlabClient.(*GitLabClient).token

	// Create GitLab client compatible with renderer
	gitlabClient := renderer.NewGitLabClient(baseURL, token, "test-project")
	return renderer.New(gitlabClient)
}

// comparePipelinesViaAPI uses the GitLab API to compare actual pipeline executions
func (rv *RefactoringValidator) comparePipelinesViaAPI(r *renderer.Renderer, beforeConfig, afterConfig *parser.GitLabConfig) (*renderer.PipelineComparison, error) {
	// Create temporary projects for before and after configurations
	beforeProject, err := rv.createTempProject("before-config")
	if err != nil {
		return nil, fmt.Errorf("failed to create before project: %w", err)
	}
	defer func() {
		go func() {
			time.Sleep(2 * time.Minute)
			rv.gitlabClient.DeleteProject(beforeProject.ID)
		}()
	}()

	afterProject, err := rv.createTempProject("after-config")
	if err != nil {
		return nil, fmt.Errorf("failed to create after project: %w", err)
	}
	defer func() {
		go func() {
			time.Sleep(2 * time.Minute)
			rv.gitlabClient.DeleteProject(afterProject.ID)
		}()
	}()

	// Upload configurations and trigger pipelines
	beforePipelineID, err := rv.uploadConfigAndTriggerPipeline(beforeProject.ID, beforeConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to trigger before pipeline: %w", err)
	}

	afterPipelineID, err := rv.uploadConfigAndTriggerPipeline(afterProject.ID, afterConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to trigger after pipeline: %w", err)
	}

	// Wait for both pipelines to complete
	beforePipeline, err := rv.gitlabClient.WaitForPipelineCompletion(beforeProject.ID, beforePipelineID.ID, 10*time.Minute)
	if err != nil {
		return nil, fmt.Errorf("before pipeline failed to complete: %w", err)
	}

	afterPipeline, err := rv.gitlabClient.WaitForPipelineCompletion(afterProject.ID, afterPipelineID.ID, 10*time.Minute)
	if err != nil {
		return nil, fmt.Errorf("after pipeline failed to complete: %w", err)
	}

	// Use renderer to compare the actual pipeline executions
	ctx := context.Background()
	comparison, err := r.ComparePipelines(ctx, beforePipeline.ID, afterPipeline.ID)
	if err != nil {
		return nil, fmt.Errorf("GitLab API pipeline comparison failed: %w", err)
	}

	return comparison, nil
}

// createTempProject creates a temporary GitLab project for testing
func (rv *RefactoringValidator) createTempProject(namePrefix string) (*Project, error) {
	projectName := fmt.Sprintf("gitlab-smith-%s-%d", namePrefix, time.Now().Unix())
	return rv.gitlabClient.CreateProject(projectName, projectName)
}

// uploadConfigAndTriggerPipeline uploads a config and triggers a pipeline
func (rv *RefactoringValidator) uploadConfigAndTriggerPipeline(projectID int, config *parser.GitLabConfig) (*Pipeline, error) {
	// Convert config back to YAML for upload
	yamlContent, err := rv.configToYAML(config)
	if err != nil {
		return nil, fmt.Errorf("failed to convert config to YAML: %w", err)
	}

	// Upload CI configuration
	err = rv.gitlabClient.CreateFile(projectID, ".gitlab-ci.yml", yamlContent, "Add CI configuration")
	if err != nil {
		return nil, fmt.Errorf("failed to upload CI config: %w", err)
	}

	// Trigger pipeline
	return rv.gitlabClient.TriggerPipeline(projectID, "main")
}

// configToYAML converts a GitLab config back to YAML (simplified implementation)
func (rv *RefactoringValidator) configToYAML(config *parser.GitLabConfig) (string, error) {
	// This is a simplified YAML generation - in a real implementation,
	// you might want to use a proper YAML marshaling library
	var yamlLines []string

	// Add stages
	if len(config.Stages) > 0 {
		yamlLines = append(yamlLines, "stages:")
		for _, stage := range config.Stages {
			yamlLines = append(yamlLines, fmt.Sprintf("  - %s", stage))
		}
		yamlLines = append(yamlLines, "")
	}

	// Add variables
	if len(config.Variables) > 0 {
		yamlLines = append(yamlLines, "variables:")
		for k, v := range config.Variables {
			yamlLines = append(yamlLines, fmt.Sprintf("  %s: %v", k, v))
		}
		yamlLines = append(yamlLines, "")
	}

	// Add jobs (simplified - just basic structure)
	for jobName, job := range config.Jobs {
		if job == nil || strings.HasPrefix(jobName, ".") {
			continue
		}

		yamlLines = append(yamlLines, fmt.Sprintf("%s:", jobName))
		if job.Stage != "" {
			yamlLines = append(yamlLines, fmt.Sprintf("  stage: %s", job.Stage))
		}
		if len(job.Script) > 0 {
			yamlLines = append(yamlLines, "  script:")
			for _, line := range job.Script {
				yamlLines = append(yamlLines, fmt.Sprintf("    - %s", line))
			}
		}
		yamlLines = append(yamlLines, "")
	}

	return strings.Join(yamlLines, "\n"), nil
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
