package gitlab

import (
	"context"
	"fmt"
	"time"
)

// Client defines the interface for interacting with GitLab
type Client interface {
	// Configuration validation
	ValidateConfig(ctx context.Context, yaml string, projectID int) (*ValidationResult, error)
	LintConfig(ctx context.Context, yaml string) (*ValidationResult, error)
	
	// Pipeline operations
	CreatePipeline(ctx context.Context, projectID int, ref string, variables map[string]string) (*Pipeline, error)
	GetPipeline(ctx context.Context, projectID, pipelineID int) (*Pipeline, error)
	GetPipelineJobs(ctx context.Context, projectID, pipelineID int) ([]*Job, error)
	CancelPipeline(ctx context.Context, projectID, pipelineID int) error
	RetryPipeline(ctx context.Context, projectID, pipelineID int) (*Pipeline, error)
	
	// Job operations
	GetJob(ctx context.Context, projectID, jobID int) (*Job, error)
	GetJobLog(ctx context.Context, projectID, jobID int) (string, error)
	GetJobArtifacts(ctx context.Context, projectID, jobID int) ([]byte, error)
	RetryJob(ctx context.Context, projectID, jobID int) (*Job, error)
	CancelJob(ctx context.Context, projectID, jobID int) (*Job, error)
	
	// Project operations
	GetProject(ctx context.Context, projectID int) (*Project, error)
	
	// Monitoring
	WaitForPipeline(ctx context.Context, projectID, pipelineID int, timeout time.Duration) (*Pipeline, error)
	WaitForJob(ctx context.Context, projectID, jobID int, timeout time.Duration) (*Job, error)
	
	// Health check
	HealthCheck(ctx context.Context) error
}

// Config holds the configuration for a GitLab client
type Config struct {
	BaseURL string
	Token   string
	Timeout time.Duration
}

// BackendType represents the type of GitLab backend
type BackendType string

const (
	// BackendAPI uses the real GitLab API
	BackendAPI BackendType = "api"
	// BackendSimulation simulates GitLab behavior locally
	BackendSimulation BackendType = "simulation"
	// BackendMock is for testing
	BackendMock BackendType = "mock"
)

// NewClient creates a new GitLab client based on the backend type
func NewClient(backendType BackendType, config *Config) (Client, error) {
	switch backendType {
	case BackendAPI:
		return NewAPIClient(config)
	case BackendSimulation:
		return NewSimulationClient()
	case BackendMock:
		return NewMockClient()
	default:
		return NewSimulationClient() // Default to simulation
	}
}

// NewAPIClient creates a client that talks to a real GitLab instance
func NewAPIClient(config *Config) (Client, error) {
	return NewAPIClientImpl(config)
}

// NewSimulationClient creates a client that simulates GitLab behavior
func NewSimulationClient() (Client, error) {
	return newSimulationClient()
}

// NewMockClient creates a mock client for testing
func NewMockClient() (Client, error) {
	// Will be implemented in backends/mock.go
	return nil, fmt.Errorf("mock client not yet implemented")
}