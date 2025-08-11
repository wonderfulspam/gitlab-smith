package renderer

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/wonderfulspam/gitlab-smith/pkg/parser"
)

// NewGitLabClient creates a new GitLab API client
func NewGitLabClient(baseURL, token, projectID string) *GitLabClient {
	return &GitLabClient{
		BaseURL:   strings.TrimSuffix(baseURL, "/"),
		Token:     token,
		ProjectID: projectID,
		Client:    &http.Client{Timeout: 30 * time.Second},
	}
}

// Renderer handles pipeline execution rendering and comparison
type Renderer struct {
	client *GitLabClient
	visual *VisualRenderer
}

// New creates a new Renderer instance
func New(client *GitLabClient) *Renderer {
	return &Renderer{
		client: client,
		visual: NewVisualRenderer(),
	}
}

// RenderPipeline fetches and renders a pipeline execution
func (r *Renderer) RenderPipeline(ctx context.Context, pipelineID int) (*PipelineExecution, error) {
	pipeline, err := r.fetchPipeline(ctx, pipelineID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch pipeline %d: %w", pipelineID, err)
	}

	// Fetch jobs for the pipeline
	jobs, err := r.fetchPipelineJobs(ctx, pipelineID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch jobs for pipeline %d: %w", pipelineID, err)
	}

	pipeline.Jobs = jobs
	return pipeline, nil
}

func (r *Renderer) fetchPipeline(ctx context.Context, pipelineID int) (*PipelineExecution, error) {
	url := fmt.Sprintf("%s/api/v4/projects/%s/pipelines/%d", r.client.BaseURL, r.client.ProjectID, pipelineID)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("PRIVATE-TOKEN", r.client.Token)
	resp, err := r.client.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status: %d", resp.StatusCode)
	}

	var pipeline PipelineExecution
	if err := json.NewDecoder(resp.Body).Decode(&pipeline); err != nil {
		return nil, err
	}

	return &pipeline, nil
}

func (r *Renderer) fetchPipelineJobs(ctx context.Context, pipelineID int) ([]JobExecution, error) {
	url := fmt.Sprintf("%s/api/v4/projects/%s/pipelines/%d/jobs", r.client.BaseURL, r.client.ProjectID, pipelineID)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("PRIVATE-TOKEN", r.client.Token)
	resp, err := r.client.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status: %d", resp.StatusCode)
	}

	var jobs []JobExecution
	if err := json.NewDecoder(resp.Body).Decode(&jobs); err != nil {
		return nil, err
	}

	return jobs, nil
}

// RenderVisualPipeline generates a visual representation of a pipeline configuration
func (r *Renderer) RenderVisualPipeline(config *parser.GitLabConfig, format string) (string, error) {
	switch format {
	case "dot":
		return r.visual.RenderPipelineGraph(config, FormatDOT)
	case "mermaid":
		return r.visual.RenderPipelineGraph(config, FormatMermaid)
	default:
		return "", fmt.Errorf("unsupported visual format: %s (supported: dot, mermaid)", format)
	}
}

// RenderVisualComparison generates a visual comparison between two pipeline configurations
func (r *Renderer) RenderVisualComparison(oldConfig, newConfig *parser.GitLabConfig, comparison *PipelineComparison, format string) (string, error) {
	switch format {
	case "dot":
		return r.visual.RenderComparisonGraph(oldConfig, newConfig, comparison, FormatDOT)
	case "mermaid":
		return r.visual.RenderComparisonGraph(oldConfig, newConfig, comparison, FormatMermaid)
	default:
		return "", fmt.Errorf("unsupported visual format: %s (supported: dot, mermaid)", format)
	}
}
