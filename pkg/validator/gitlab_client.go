package validator

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// GitLabClient provides API access to a GitLab instance
type GitLabClient struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

// NewGitLabClient creates a new GitLab API client
func NewGitLabClient(baseURL, token string) *GitLabClient {
	return &GitLabClient{
		baseURL: baseURL,
		token:   token,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Project represents a GitLab project
type Project struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Path string `json:"path"`
}

// Pipeline represents a GitLab pipeline
type Pipeline struct {
	ID     int    `json:"id"`
	Status string `json:"status"`
	Ref    string `json:"ref"`
	WebURL string `json:"web_url"`
}

// Job represents a GitLab CI job
type Job struct {
	ID         int     `json:"id"`
	Name       string  `json:"name"`
	Status     string  `json:"status"`
	Stage      string  `json:"stage"`
	Duration   float64 `json:"duration"`
	FinishedAt string  `json:"finished_at"`
	StartedAt  string  `json:"started_at"`
	WebURL     string  `json:"web_url"`
	PipelineID int     `json:"pipeline_id"`
}

// CreateProject creates a new project in GitLab
func (c *GitLabClient) CreateProject(name, path string) (*Project, error) {
	payload := map[string]interface{}{
		"name":                   name,
		"path":                   path,
		"visibility":             "private",
		"initialize_with_readme": false,
	}

	project := &Project{}
	err := c.makeRequest("POST", "/api/v4/projects", payload, project)
	if err != nil {
		return nil, fmt.Errorf("failed to create project: %w", err)
	}

	return project, nil
}

// GetProject retrieves a project by path
func (c *GitLabClient) GetProject(path string) (*Project, error) {
	project := &Project{}
	err := c.makeRequest("GET", fmt.Sprintf("/api/v4/projects/%s", path), nil, project)
	if err != nil {
		return nil, fmt.Errorf("failed to get project: %w", err)
	}

	return project, nil
}

// DeleteProject deletes a project
func (c *GitLabClient) DeleteProject(projectID int) error {
	err := c.makeRequest("DELETE", fmt.Sprintf("/api/v4/projects/%d", projectID), nil, nil)
	if err != nil {
		return fmt.Errorf("failed to delete project: %w", err)
	}

	return nil
}

// CreateFile creates or updates a file in the project repository
func (c *GitLabClient) CreateFile(projectID int, filePath, content, commitMessage string) error {
	payload := map[string]interface{}{
		"file_path":      filePath,
		"branch":         "main",
		"content":        content,
		"commit_message": commitMessage,
	}

	err := c.makeRequest("POST", fmt.Sprintf("/api/v4/projects/%d/repository/files/%s", projectID, filePath), payload, nil)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}

	return nil
}

// TriggerPipeline triggers a new pipeline for the project
func (c *GitLabClient) TriggerPipeline(projectID int, ref string) (*Pipeline, error) {
	payload := map[string]interface{}{
		"ref": ref,
	}

	pipeline := &Pipeline{}
	err := c.makeRequest("POST", fmt.Sprintf("/api/v4/projects/%d/pipeline", projectID), payload, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to trigger pipeline: %w", err)
	}

	return pipeline, nil
}

// GetPipeline retrieves a pipeline by ID
func (c *GitLabClient) GetPipeline(projectID, pipelineID int) (*Pipeline, error) {
	pipeline := &Pipeline{}
	err := c.makeRequest("GET", fmt.Sprintf("/api/v4/projects/%d/pipelines/%d", projectID, pipelineID), nil, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to get pipeline: %w", err)
	}

	return pipeline, nil
}

// GetPipelineJobs retrieves all jobs for a pipeline
func (c *GitLabClient) GetPipelineJobs(projectID, pipelineID int) ([]Job, error) {
	var jobs []Job
	err := c.makeRequest("GET", fmt.Sprintf("/api/v4/projects/%d/pipelines/%d/jobs", projectID, pipelineID), nil, &jobs)
	if err != nil {
		return nil, fmt.Errorf("failed to get pipeline jobs: %w", err)
	}

	return jobs, nil
}

// WaitForPipelineCompletion waits for a pipeline to complete
func (c *GitLabClient) WaitForPipelineCompletion(projectID, pipelineID int, timeout time.Duration) (*Pipeline, error) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	timeoutTimer := time.NewTimer(timeout)
	defer timeoutTimer.Stop()

	for {
		select {
		case <-timeoutTimer.C:
			return nil, fmt.Errorf("pipeline did not complete within timeout")
		case <-ticker.C:
			pipeline, err := c.GetPipeline(projectID, pipelineID)
			if err != nil {
				return nil, err
			}

			switch pipeline.Status {
			case "success", "failed", "canceled", "skipped":
				return pipeline, nil
			case "running", "pending":
				// Continue waiting
				continue
			default:
				return pipeline, fmt.Errorf("unexpected pipeline status: %s", pipeline.Status)
			}
		}
	}
}

// makeRequest makes an HTTP request to the GitLab API
func (c *GitLabClient) makeRequest(method, endpoint string, payload interface{}, result interface{}) error {
	var body io.Reader

	if payload != nil {
		jsonPayload, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("failed to marshal payload: %w", err)
		}
		body = bytes.NewBuffer(jsonPayload)
	}

	req, err := http.NewRequest(method, c.baseURL+endpoint, body)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	if result != nil && resp.StatusCode != http.StatusNoContent {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}
	}

	return nil
}
