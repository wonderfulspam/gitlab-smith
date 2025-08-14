package gitlab

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// apiClient implements the Client interface using real GitLab API
type apiClient struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

// NewAPIClientImpl creates a new API client for real GitLab instance
func NewAPIClientImpl(config *Config) (*apiClient, error) {
	if config.BaseURL == "" {
		return nil, fmt.Errorf("GitLab URL is required")
	}
	if config.Token == "" {
		return nil, fmt.Errorf("GitLab token is required")
	}

	timeout := config.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	return &apiClient{
		baseURL: config.BaseURL,
		token:   config.Token,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}, nil
}

// doRequest performs an HTTP request with GitLab authentication
func (c *apiClient) doRequest(ctx context.Context, method, path string, body interface{}) (*http.Response, error) {
	url := fmt.Sprintf("%s/api/v4%s", c.baseURL, path)

	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		bodyReader = bytes.NewBuffer(jsonBody)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, err
	}

	req.Header.Set("PRIVATE-TOKEN", c.token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	return c.httpClient.Do(req)
}

// ValidateConfig validates a GitLab CI configuration
func (c *apiClient) ValidateConfig(ctx context.Context, yaml string, projectID int) (*ValidationResult, error) {
	// Use the CI lint API
	body := map[string]interface{}{
		"content": yaml,
	}

	resp, err := c.doRequest(ctx, "POST", "/ci/lint", body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var lintResult struct {
		Valid    bool     `json:"valid"`
		Errors   []string `json:"errors"`
		Warnings []string `json:"warnings"`
		MergedYaml string `json:"merged_yaml"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&lintResult); err != nil {
		return nil, err
	}

	return &ValidationResult{
		Valid:    lintResult.Valid,
		Errors:   lintResult.Errors,
		Warnings: lintResult.Warnings,
		Merged:   lintResult.MergedYaml,
	}, nil
}

// LintConfig performs GitLab CI lint validation
func (c *apiClient) LintConfig(ctx context.Context, yaml string) (*ValidationResult, error) {
	return c.ValidateConfig(ctx, yaml, 0)
}

// CreatePipeline creates a new pipeline
func (c *apiClient) CreatePipeline(ctx context.Context, projectID int, ref string, variables map[string]string) (*Pipeline, error) {
	body := map[string]interface{}{
		"ref": ref,
	}

	if len(variables) > 0 {
		vars := make([]map[string]string, 0, len(variables))
		for k, v := range variables {
			vars = append(vars, map[string]string{
				"key":   k,
				"value": v,
			})
		}
		body["variables"] = vars
	}

	resp, err := c.doRequest(ctx, "POST", fmt.Sprintf("/projects/%d/pipeline", projectID), body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to create pipeline: %s", string(bodyBytes))
	}

	var pipeline Pipeline
	if err := json.NewDecoder(resp.Body).Decode(&pipeline); err != nil {
		return nil, err
	}

	// Optionally cancel the pipeline immediately to prevent execution
	// This is useful for just rendering without running
	// c.CancelPipeline(ctx, projectID, pipeline.ID)

	return &pipeline, nil
}

// GetPipeline retrieves a pipeline
func (c *apiClient) GetPipeline(ctx context.Context, projectID, pipelineID int) (*Pipeline, error) {
	resp, err := c.doRequest(ctx, "GET", fmt.Sprintf("/projects/%d/pipelines/%d", projectID, pipelineID), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var pipeline Pipeline
	if err := json.NewDecoder(resp.Body).Decode(&pipeline); err != nil {
		return nil, err
	}

	return &pipeline, nil
}

// GetPipelineJobs retrieves jobs for a pipeline
func (c *apiClient) GetPipelineJobs(ctx context.Context, projectID, pipelineID int) ([]*Job, error) {
	resp, err := c.doRequest(ctx, "GET", fmt.Sprintf("/projects/%d/pipelines/%d/jobs", projectID, pipelineID), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var jobs []*Job
	if err := json.NewDecoder(resp.Body).Decode(&jobs); err != nil {
		return nil, err
	}

	return jobs, nil
}

// CancelPipeline cancels a pipeline
func (c *apiClient) CancelPipeline(ctx context.Context, projectID, pipelineID int) error {
	resp, err := c.doRequest(ctx, "POST", fmt.Sprintf("/projects/%d/pipelines/%d/cancel", projectID, pipelineID), nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to cancel pipeline: %s", string(bodyBytes))
	}

	return nil
}

// RetryPipeline retries a pipeline
func (c *apiClient) RetryPipeline(ctx context.Context, projectID, pipelineID int) (*Pipeline, error) {
	resp, err := c.doRequest(ctx, "POST", fmt.Sprintf("/projects/%d/pipelines/%d/retry", projectID, pipelineID), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var pipeline Pipeline
	if err := json.NewDecoder(resp.Body).Decode(&pipeline); err != nil {
		return nil, err
	}

	return &pipeline, nil
}

// GetJob retrieves a job
func (c *apiClient) GetJob(ctx context.Context, projectID, jobID int) (*Job, error) {
	resp, err := c.doRequest(ctx, "GET", fmt.Sprintf("/projects/%d/jobs/%d", projectID, jobID), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var job Job
	if err := json.NewDecoder(resp.Body).Decode(&job); err != nil {
		return nil, err
	}

	return &job, nil
}

// GetJobLog retrieves job logs
func (c *apiClient) GetJobLog(ctx context.Context, projectID, jobID int) (string, error) {
	resp, err := c.doRequest(ctx, "GET", fmt.Sprintf("/projects/%d/jobs/%d/trace", projectID, jobID), nil)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	logBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(logBytes), nil
}

// GetJobArtifacts retrieves job artifacts
func (c *apiClient) GetJobArtifacts(ctx context.Context, projectID, jobID int) ([]byte, error) {
	resp, err := c.doRequest(ctx, "GET", fmt.Sprintf("/projects/%d/jobs/%d/artifacts", projectID, jobID), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return io.ReadAll(resp.Body)
}

// RetryJob retries a job
func (c *apiClient) RetryJob(ctx context.Context, projectID, jobID int) (*Job, error) {
	resp, err := c.doRequest(ctx, "POST", fmt.Sprintf("/projects/%d/jobs/%d/retry", projectID, jobID), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var job Job
	if err := json.NewDecoder(resp.Body).Decode(&job); err != nil {
		return nil, err
	}

	return &job, nil
}

// CancelJob cancels a job
func (c *apiClient) CancelJob(ctx context.Context, projectID, jobID int) (*Job, error) {
	resp, err := c.doRequest(ctx, "POST", fmt.Sprintf("/projects/%d/jobs/%d/cancel", projectID, jobID), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var job Job
	if err := json.NewDecoder(resp.Body).Decode(&job); err != nil {
		return nil, err
	}

	return &job, nil
}

// GetProject retrieves project information
func (c *apiClient) GetProject(ctx context.Context, projectID int) (*Project, error) {
	resp, err := c.doRequest(ctx, "GET", fmt.Sprintf("/projects/%d", projectID), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var project Project
	if err := json.NewDecoder(resp.Body).Decode(&project); err != nil {
		return nil, err
	}

	return &project, nil
}

// WaitForPipeline waits for a pipeline to complete
func (c *apiClient) WaitForPipeline(ctx context.Context, projectID, pipelineID int, timeout time.Duration) (*Pipeline, error) {
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
		case <-time.After(2 * time.Second):
			// Continue polling
		}
	}

	return nil, fmt.Errorf("timeout waiting for pipeline %d", pipelineID)
}

// WaitForJob waits for a job to complete
func (c *apiClient) WaitForJob(ctx context.Context, projectID, jobID int, timeout time.Duration) (*Job, error) {
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
		case <-time.After(2 * time.Second):
			// Continue polling
		}
	}

	return nil, fmt.Errorf("timeout waiting for job %d", jobID)
}

// HealthCheck checks if GitLab is accessible
func (c *apiClient) HealthCheck(ctx context.Context) error {
	resp, err := c.doRequest(ctx, "GET", "/version", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GitLab health check failed with status %d", resp.StatusCode)
	}

	return nil
}