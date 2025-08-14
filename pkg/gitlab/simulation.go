package gitlab

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/wonderfulspam/gitlab-smith/pkg/parser"
)

// simulationClient simulates GitLab behavior without requiring a real instance
type simulationClient struct {
	pipelines map[string]*Pipeline // key: projectID-pipelineID
	jobs     map[string]*Job       // key: projectID-jobID
	nextID   int
}

// newSimulationClient creates a new simulation client
func newSimulationClient() (*simulationClient, error) {
	return &simulationClient{
		pipelines: make(map[string]*Pipeline),
		jobs:      make(map[string]*Job),
		nextID:    1000,
	}, nil
}

// ValidateConfig validates a GitLab CI configuration
func (c *simulationClient) ValidateConfig(ctx context.Context, yaml string, projectID int) (*ValidationResult, error) {
	// Use parser to validate
	config, err := parser.Parse([]byte(yaml))
	if err != nil {
		return &ValidationResult{
			Valid:  false,
			Errors: []string{err.Error()},
		}, nil
	}

	result := &ValidationResult{
		Valid:    true,
		Warnings: []string{},
	}

	// Check for common issues
	if len(config.Jobs) == 0 {
		result.Warnings = append(result.Warnings, "No jobs defined in configuration")
	}

	if len(config.Stages) == 0 && len(config.Jobs) > 0 {
		result.Warnings = append(result.Warnings, "No stages explicitly defined")
	}

	// Check for undefined stages
	stageMap := make(map[string]bool)
	for _, stage := range config.Stages {
		stageMap[stage] = true
	}

	for name, job := range config.Jobs {
		if job.Stage != "" && !stageMap[job.Stage] {
			result.Warnings = append(result.Warnings, fmt.Sprintf("Job '%s' uses undefined stage '%s'", name, job.Stage))
		}
	}

	return result, nil
}

// LintConfig performs GitLab CI lint validation
func (c *simulationClient) LintConfig(ctx context.Context, yaml string) (*ValidationResult, error) {
	return c.ValidateConfig(ctx, yaml, 0)
}

// CreatePipeline creates a simulated pipeline
func (c *simulationClient) CreatePipeline(ctx context.Context, projectID int, ref string, variables map[string]string) (*Pipeline, error) {
	c.nextID++
	pipelineID := c.nextID
	
	now := time.Now()
	pipeline := &Pipeline{
		ID:        pipelineID,
		Status:    "created",
		Ref:       ref,
		SHA:       generateSHA(),
		CreatedAt: now,
		UpdatedAt: now,
		Variables: variables,
		Jobs:      []*Job{},
	}

	key := fmt.Sprintf("%d-%d", projectID, pipelineID)
	c.pipelines[key] = pipeline

	// Simulate pipeline execution
	go c.simulatePipelineExecution(ctx, projectID, pipelineID)

	return pipeline, nil
}

// GetPipeline retrieves a pipeline
func (c *simulationClient) GetPipeline(ctx context.Context, projectID, pipelineID int) (*Pipeline, error) {
	key := fmt.Sprintf("%d-%d", projectID, pipelineID)
	pipeline, exists := c.pipelines[key]
	if !exists {
		return nil, fmt.Errorf("pipeline %d not found", pipelineID)
	}
	return pipeline, nil
}

// GetPipelineJobs retrieves jobs for a pipeline
func (c *simulationClient) GetPipelineJobs(ctx context.Context, projectID, pipelineID int) ([]*Job, error) {
	pipeline, err := c.GetPipeline(ctx, projectID, pipelineID)
	if err != nil {
		return nil, err
	}
	return pipeline.Jobs, nil
}

// CancelPipeline cancels a pipeline
func (c *simulationClient) CancelPipeline(ctx context.Context, projectID, pipelineID int) error {
	pipeline, err := c.GetPipeline(ctx, projectID, pipelineID)
	if err != nil {
		return err
	}
	
	pipeline.Status = "canceled"
	now := time.Now()
	pipeline.FinishedAt = &now
	pipeline.UpdatedAt = now
	
	// Cancel all jobs
	for _, job := range pipeline.Jobs {
		if job.Status == "pending" || job.Status == "running" {
			job.Status = "canceled"
			job.FinishedAt = &now
		}
	}
	
	return nil
}

// RetryPipeline retries a failed pipeline
func (c *simulationClient) RetryPipeline(ctx context.Context, projectID, pipelineID int) (*Pipeline, error) {
	oldPipeline, err := c.GetPipeline(ctx, projectID, pipelineID)
	if err != nil {
		return nil, err
	}
	
	// Create new pipeline with same configuration
	return c.CreatePipeline(ctx, projectID, oldPipeline.Ref, oldPipeline.Variables)
}

// GetJob retrieves a job
func (c *simulationClient) GetJob(ctx context.Context, projectID, jobID int) (*Job, error) {
	key := fmt.Sprintf("%d-%d", projectID, jobID)
	job, exists := c.jobs[key]
	if !exists {
		return nil, fmt.Errorf("job %d not found", jobID)
	}
	return job, nil
}

// GetJobLog retrieves job logs
func (c *simulationClient) GetJobLog(ctx context.Context, projectID, jobID int) (string, error) {
	job, err := c.GetJob(ctx, projectID, jobID)
	if err != nil {
		return "", err
	}
	
	if job.Log == "" {
		// Generate simulated log
		job.Log = c.generateJobLog(job)
	}
	
	return job.Log, nil
}

// GetJobArtifacts retrieves job artifacts
func (c *simulationClient) GetJobArtifacts(ctx context.Context, projectID, jobID int) ([]byte, error) {
	job, err := c.GetJob(ctx, projectID, jobID)
	if err != nil {
		return nil, err
	}
	
	if len(job.Artifacts) == 0 {
		return nil, fmt.Errorf("no artifacts found for job %d", jobID)
	}
	
	// Return simulated artifact data
	return []byte("simulated artifact data"), nil
}

// RetryJob retries a failed job
func (c *simulationClient) RetryJob(ctx context.Context, projectID, jobID int) (*Job, error) {
	job, err := c.GetJob(ctx, projectID, jobID)
	if err != nil {
		return nil, err
	}
	
	// Reset job status
	job.Status = "pending"
	job.StartedAt = nil
	job.FinishedAt = nil
	job.Duration = 0
	job.Log = ""
	
	// Simulate job execution
	go c.simulateJobExecution(ctx, projectID, jobID)
	
	return job, nil
}

// CancelJob cancels a running job
func (c *simulationClient) CancelJob(ctx context.Context, projectID, jobID int) (*Job, error) {
	job, err := c.GetJob(ctx, projectID, jobID)
	if err != nil {
		return nil, err
	}
	
	if job.Status == "pending" || job.Status == "running" {
		job.Status = "canceled"
		now := time.Now()
		job.FinishedAt = &now
	}
	
	return job, nil
}

// GetProject retrieves project information
func (c *simulationClient) GetProject(ctx context.Context, projectID int) (*Project, error) {
	// Return simulated project
	return &Project{
		ID:                projectID,
		Name:              fmt.Sprintf("project-%d", projectID),
		Path:              fmt.Sprintf("project-%d", projectID),
		DefaultBranch:     "main",
		WebURL:            fmt.Sprintf("http://localhost/project-%d", projectID),
		NamespaceFullPath: "gitlab-smith",
	}, nil
}

// WaitForPipeline waits for a pipeline to complete
func (c *simulationClient) WaitForPipeline(ctx context.Context, projectID, pipelineID int, timeout time.Duration) (*Pipeline, error) {
	deadline := time.Now().Add(timeout)
	
	for time.Now().Before(deadline) {
		pipeline, err := c.GetPipeline(ctx, projectID, pipelineID)
		if err != nil {
			return nil, err
		}
		
		if isTerminalStatus(pipeline.Status) {
			return pipeline, nil
		}
		
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(1 * time.Second):
			// Continue polling
		}
	}
	
	return nil, fmt.Errorf("timeout waiting for pipeline %d", pipelineID)
}

// WaitForJob waits for a job to complete
func (c *simulationClient) WaitForJob(ctx context.Context, projectID, jobID int, timeout time.Duration) (*Job, error) {
	deadline := time.Now().Add(timeout)
	
	for time.Now().Before(deadline) {
		job, err := c.GetJob(ctx, projectID, jobID)
		if err != nil {
			return nil, err
		}
		
		if isTerminalStatus(job.Status) {
			return job, nil
		}
		
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(1 * time.Second):
			// Continue polling
		}
	}
	
	return nil, fmt.Errorf("timeout waiting for job %d", jobID)
}

// HealthCheck checks if the simulated GitLab is healthy
func (c *simulationClient) HealthCheck(ctx context.Context) error {
	// Simulation is always healthy
	return nil
}

// simulatePipelineExecution simulates the execution of a pipeline
func (c *simulationClient) simulatePipelineExecution(ctx context.Context, projectID, pipelineID int) {
	key := fmt.Sprintf("%d-%d", projectID, pipelineID)
	pipeline := c.pipelines[key]
	
	// Update status to running
	pipeline.Status = "running"
	now := time.Now()
	pipeline.StartedAt = &now
	pipeline.UpdatedAt = now
	
	// Create simulated jobs
	stages := []string{"build", "test", "deploy"}
	for i, stage := range stages {
		for j := 0; j < rand.Intn(3)+1; j++ {
			c.nextID++
			jobID := c.nextID
			
			job := &Job{
				ID:        jobID,
				Name:      fmt.Sprintf("%s-job-%d", stage, j+1),
				Stage:     stage,
				Status:    "pending",
				CreatedAt: now,
				When:      "on_success",
			}
			
			jobKey := fmt.Sprintf("%d-%d", projectID, jobID)
			c.jobs[jobKey] = job
			pipeline.Jobs = append(pipeline.Jobs, job)
			
			// Simulate job execution with delay
			go func(jID int, delay time.Duration) {
				time.Sleep(delay)
				c.simulateJobExecution(ctx, projectID, jID)
			}(jobID, time.Duration(i)*2*time.Second)
		}
	}
	
	// Wait for all jobs to complete
	time.Sleep(10 * time.Second)
	
	// Update pipeline status based on job results
	allSuccess := true
	for _, job := range pipeline.Jobs {
		if job.Status != "success" {
			allSuccess = false
			break
		}
	}
	
	if allSuccess {
		pipeline.Status = "success"
	} else {
		pipeline.Status = "failed"
	}
	
	finishTime := time.Now()
	pipeline.FinishedAt = &finishTime
	pipeline.UpdatedAt = finishTime
	pipeline.Duration = int(finishTime.Sub(*pipeline.StartedAt).Seconds())
}

// simulateJobExecution simulates the execution of a job
func (c *simulationClient) simulateJobExecution(ctx context.Context, projectID, jobID int) {
	key := fmt.Sprintf("%d-%d", projectID, jobID)
	job := c.jobs[key]
	
	// Update to running
	job.Status = "running"
	now := time.Now()
	job.StartedAt = &now
	
	// Simulate execution time
	duration := time.Duration(rand.Intn(30)+10) * time.Second
	time.Sleep(duration / 10) // Speed up simulation
	
	// Randomly succeed or fail (80% success rate)
	if rand.Float32() < 0.8 {
		job.Status = "success"
	} else {
		job.Status = "failed"
	}
	
	finishTime := time.Now()
	job.FinishedAt = &finishTime
	job.Duration = finishTime.Sub(*job.StartedAt).Seconds()
}

// generateJobLog generates simulated job logs
func (c *simulationClient) generateJobLog(job *Job) string {
	var log strings.Builder
	
	log.WriteString(fmt.Sprintf("Running job: %s\n", job.Name))
	log.WriteString(fmt.Sprintf("Stage: %s\n", job.Stage))
	log.WriteString("=========================\n\n")
	
	// Simulate some command output
	commands := []string{
		"$ echo 'Starting job execution'",
		"Starting job execution",
		"$ npm install",
		"added 1234 packages in 15.2s",
		"$ npm test",
		"PASS src/app.test.js",
		"Test Suites: 1 passed, 1 total",
		"Tests: 5 passed, 5 total",
	}
	
	for _, cmd := range commands {
		log.WriteString(cmd + "\n")
	}
	
	if job.Status == "success" {
		log.WriteString("\nJob succeeded\n")
	} else if job.Status == "failed" {
		log.WriteString("\nERROR: Job failed\n")
		log.WriteString("Exit code: 1\n")
	}
	
	return log.String()
}

// generateSHA generates a random SHA for simulation
func generateSHA() string {
	const chars = "0123456789abcdef"
	b := make([]byte, 40)
	for i := range b {
		b[i] = chars[rand.Intn(len(chars))]
	}
	return string(b)
}

// isTerminalStatus checks if a status is terminal (finished)
func isTerminalStatus(status string) bool {
	switch status {
	case "success", "failed", "canceled", "skipped", "manual":
		return true
	default:
		return false
	}
}