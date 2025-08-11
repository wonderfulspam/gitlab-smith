package renderer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/emt/gitlab-smith/pkg/parser"
)

// PipelineExecution represents a GitLab pipeline execution
type PipelineExecution struct {
	ID             int               `json:"id"`
	Status         string            `json:"status"`
	Ref            string            `json:"ref"`
	SHA            string            `json:"sha"`
	Jobs           []JobExecution    `json:"jobs"`
	Variables      map[string]string `json:"variables"`
	CreatedAt      time.Time         `json:"created_at"`
	UpdatedAt      time.Time         `json:"updated_at"`
	Duration       int               `json:"duration"`
	QueuedDuration int               `json:"queued_duration"`
}

// JobExecution represents a single job execution within a pipeline
type JobExecution struct {
	ID             int            `json:"id"`
	Name           string         `json:"name"`
	Stage          string         `json:"stage"`
	Status         string         `json:"status"`
	StartedAt      *time.Time     `json:"started_at"`
	FinishedAt     *time.Time     `json:"finished_at"`
	Duration       float64        `json:"duration"`
	QueuedDuration float64        `json:"queued_duration"`
	Runner         *RunnerInfo    `json:"runner"`
	Artifacts      []ArtifactInfo `json:"artifacts"`
	Dependencies   []string       `json:"dependencies"`
	Needs          []string       `json:"needs"`
}

// RunnerInfo represents information about the runner that executed a job
type RunnerInfo struct {
	ID          int      `json:"id"`
	Description string   `json:"description"`
	Tags        []string `json:"tags"`
}

// ArtifactInfo represents artifact information for a job
type ArtifactInfo struct {
	FileName string `json:"file_name"`
	FileType string `json:"file_type"`
	Size     int64  `json:"size"`
}

// PipelineComparison represents a comparison between two pipeline executions
type PipelineComparison struct {
	OldExecution    *PipelineExecution `json:"old_execution"`
	NewExecution    *PipelineExecution `json:"new_execution"`
	JobComparisons  []JobComparison    `json:"job_comparisons"`
	Summary         ComparisonSummary  `json:"summary"`
	PerformanceGain PerformanceMetrics `json:"performance_gain"`
}

// JobComparison represents a comparison between two job executions
type JobComparison struct {
	JobName         string        `json:"job_name"`
	OldJob          *JobExecution `json:"old_job,omitempty"`
	NewJob          *JobExecution `json:"new_job,omitempty"`
	Status          CompareStatus `json:"status"`
	DurationChange  float64       `json:"duration_change"`
	QueueTimeChange float64       `json:"queue_time_change"`
	Changes         []string      `json:"changes"`
}

// CompareStatus indicates the comparison status between jobs
type CompareStatus string

const (
	StatusIdentical    CompareStatus = "identical"
	StatusImproved     CompareStatus = "improved"
	StatusDegraded     CompareStatus = "degraded"
	StatusAdded        CompareStatus = "added"
	StatusRemoved      CompareStatus = "removed"
	StatusRestructured CompareStatus = "restructured"
)

// ComparisonSummary provides high-level comparison metrics
type ComparisonSummary struct {
	TotalJobs          int     `json:"total_jobs"`
	AddedJobs          int     `json:"added_jobs"`
	RemovedJobs        int     `json:"removed_jobs"`
	ImprovedJobs       int     `json:"improved_jobs"`
	DegradedJobs       int     `json:"degraded_jobs"`
	IdenticalJobs      int     `json:"identical_jobs"`
	OverallImprovement bool    `json:"overall_improvement"`
	TotalTimeChange    float64 `json:"total_time_change"`
}

// PerformanceMetrics tracks performance-related changes
type PerformanceMetrics struct {
	TotalPipelineDuration  float64 `json:"total_pipeline_duration"`
	AverageJobDuration     float64 `json:"average_job_duration"`
	ParallelismImprovement int     `json:"parallelism_improvement"`
	CacheHitImprovements   int     `json:"cache_hit_improvements"`
	StartupTimeReduction   float64 `json:"startup_time_reduction"`
}

// GitLabClient represents a GitLab API client for interacting with pipelines
type GitLabClient struct {
	BaseURL   string
	Token     string
	ProjectID string
	Client    *http.Client
}

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

// ComparePipelines compares two pipeline executions and provides detailed analysis
func (r *Renderer) ComparePipelines(ctx context.Context, oldPipelineID, newPipelineID int) (*PipelineComparison, error) {
	oldPipeline, err := r.RenderPipeline(ctx, oldPipelineID)
	if err != nil {
		return nil, fmt.Errorf("failed to render old pipeline: %w", err)
	}

	newPipeline, err := r.RenderPipeline(ctx, newPipelineID)
	if err != nil {
		return nil, fmt.Errorf("failed to render new pipeline: %w", err)
	}

	return r.compareExecutions(oldPipeline, newPipeline), nil
}

// CompareConfigurations simulates pipeline execution based on configurations
func (r *Renderer) CompareConfigurations(oldConfig, newConfig *parser.GitLabConfig) (*PipelineComparison, error) {
	oldSimulation := r.simulatePipelineExecution(oldConfig)
	newSimulation := r.simulatePipelineExecution(newConfig)

	return r.compareExecutions(oldSimulation, newSimulation), nil
}

// simulatePipelineExecution creates a simulated pipeline execution from a config
func (r *Renderer) simulatePipelineExecution(config *parser.GitLabConfig) *PipelineExecution {
	pipeline := &PipelineExecution{
		ID:        0, // Simulated
		Status:    "simulated",
		Ref:       "main",
		SHA:       "simulated",
		Jobs:      make([]JobExecution, 0),
		Variables: convertVariables(config.Variables),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Convert parsed jobs to job executions
	for jobName, job := range config.Jobs {
		if job == nil {
			continue
		}

		// Skip template jobs (starting with .) as they don't run independently
		if strings.HasPrefix(jobName, ".") {
			continue
		}

		jobExec := JobExecution{
			ID:             0, // Simulated
			Name:           jobName,
			Stage:          job.Stage,
			Status:         "simulated",
			Dependencies:   job.Dependencies,
			Needs:          extractJobNames(job.Needs),
			Duration:       estimateJobDurationWithContext(job, config.Jobs),
			QueuedDuration: 0,
		}

		pipeline.Jobs = append(pipeline.Jobs, jobExec)
	}

	// Sort jobs by stage order
	sort.Slice(pipeline.Jobs, func(i, j int) bool {
		return getStageOrder(pipeline.Jobs[i].Stage, config.Stages) <
			getStageOrder(pipeline.Jobs[j].Stage, config.Stages)
	})

	return pipeline
}

// compareExecutions performs detailed comparison between two pipeline executions
func (r *Renderer) compareExecutions(oldPipeline, newPipeline *PipelineExecution) *PipelineComparison {
	comparison := &PipelineComparison{
		OldExecution:   oldPipeline,
		NewExecution:   newPipeline,
		JobComparisons: make([]JobComparison, 0),
	}

	// Create job lookup maps
	oldJobs := make(map[string]*JobExecution)
	newJobs := make(map[string]*JobExecution)

	for i := range oldPipeline.Jobs {
		oldJobs[oldPipeline.Jobs[i].Name] = &oldPipeline.Jobs[i]
	}

	for i := range newPipeline.Jobs {
		newJobs[newPipeline.Jobs[i].Name] = &newPipeline.Jobs[i]
	}

	// Get all unique job names
	allJobNames := make(map[string]bool)
	for name := range oldJobs {
		allJobNames[name] = true
	}
	for name := range newJobs {
		allJobNames[name] = true
	}

	// Compare each job
	var totalTimeChange float64
	summary := ComparisonSummary{}

	for jobName := range allJobNames {
		oldJob := oldJobs[jobName]
		newJob := newJobs[jobName]

		jobComparison := r.compareJobs(jobName, oldJob, newJob)
		comparison.JobComparisons = append(comparison.JobComparisons, jobComparison)

		// Update summary statistics
		switch jobComparison.Status {
		case StatusAdded:
			summary.AddedJobs++
		case StatusRemoved:
			summary.RemovedJobs++
		case StatusImproved:
			summary.ImprovedJobs++
		case StatusDegraded:
			summary.DegradedJobs++
		case StatusIdentical:
			summary.IdenticalJobs++
		}

		totalTimeChange += jobComparison.DurationChange
		summary.TotalJobs++
	}

	summary.TotalTimeChange = totalTimeChange
	summary.OverallImprovement = totalTimeChange < 0 // Negative means faster

	comparison.Summary = summary
	comparison.PerformanceGain = r.calculatePerformanceMetrics(oldPipeline, newPipeline)

	return comparison
}

// compareJobs compares two individual job executions
func (r *Renderer) compareJobs(jobName string, oldJob, newJob *JobExecution) JobComparison {
	comparison := JobComparison{
		JobName: jobName,
		OldJob:  oldJob,
		NewJob:  newJob,
		Changes: make([]string, 0),
	}

	if oldJob == nil && newJob != nil {
		comparison.Status = StatusAdded
		comparison.DurationChange = newJob.Duration
		comparison.Changes = append(comparison.Changes, "Job added to pipeline")
	} else if oldJob != nil && newJob == nil {
		comparison.Status = StatusRemoved
		comparison.DurationChange = -oldJob.Duration
		comparison.Changes = append(comparison.Changes, "Job removed from pipeline")
	} else {
		// Both jobs exist, compare them
		comparison.DurationChange = newJob.Duration - oldJob.Duration
		comparison.QueueTimeChange = newJob.QueuedDuration - oldJob.QueuedDuration

		if oldJob.Stage != newJob.Stage {
			comparison.Changes = append(comparison.Changes, fmt.Sprintf("Stage changed from %s to %s", oldJob.Stage, newJob.Stage))
		}

		if !equalStringSlices(oldJob.Dependencies, newJob.Dependencies) {
			comparison.Changes = append(comparison.Changes, "Dependencies changed")
		}

		if !equalStringSlices(oldJob.Needs, newJob.Needs) {
			comparison.Changes = append(comparison.Changes, "Needs relationships changed")
		}

		// Determine overall status
		if len(comparison.Changes) == 0 && comparison.DurationChange == 0 {
			comparison.Status = StatusIdentical
		} else if comparison.DurationChange < -5 { // More than 5 seconds improvement
			comparison.Status = StatusImproved
		} else if comparison.DurationChange > 5 { // More than 5 seconds degradation
			comparison.Status = StatusDegraded
		} else {
			comparison.Status = StatusRestructured
		}
	}

	return comparison
}

// calculatePerformanceMetrics calculates performance metrics between pipelines
func (r *Renderer) calculatePerformanceMetrics(oldPipeline, newPipeline *PipelineExecution) PerformanceMetrics {
	oldTotalDuration := float64(oldPipeline.Duration)
	newTotalDuration := float64(newPipeline.Duration)

	oldAvgDuration := r.calculateAverageJobDuration(oldPipeline.Jobs)
	newAvgDuration := r.calculateAverageJobDuration(newPipeline.Jobs)

	return PerformanceMetrics{
		TotalPipelineDuration:  newTotalDuration - oldTotalDuration,
		AverageJobDuration:     newAvgDuration - oldAvgDuration,
		ParallelismImprovement: r.calculateParallelismImprovement(oldPipeline, newPipeline),
		StartupTimeReduction:   r.calculateStartupTimeReduction(oldPipeline, newPipeline),
	}
}

// Helper functions

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

func convertVariables(vars map[string]interface{}) map[string]string {
	if vars == nil {
		return make(map[string]string)
	}

	result := make(map[string]string, len(vars))
	for k, v := range vars {
		if str, ok := v.(string); ok {
			result[k] = str
		} else {
			result[k] = fmt.Sprintf("%v", v)
		}
	}
	return result
}

func extractJobNames(needs interface{}) []string {
	if needs == nil {
		return []string{}
	}

	// Handle different need formats
	switch v := needs.(type) {
	case []string:
		return v
	case []interface{}:
		names := make([]string, 0, len(v))
		for _, item := range v {
			if str, ok := item.(string); ok {
				names = append(names, str)
			} else if job, ok := item.(map[string]interface{}); ok {
				if jobName, exists := job["job"]; exists {
					if str, ok := jobName.(string); ok {
						names = append(names, str)
					}
				}
			}
		}
		return names
	case string:
		return []string{v}
	default:
		return []string{}
	}
}

func estimateJobDuration(job *parser.JobConfig) float64 {
	// Simple heuristic: base duration + script length factor
	baseDuration := 30.0                           // 30 seconds base
	scriptFactor := float64(len(job.Script)) * 2.0 // 2 seconds per script line

	// Add before_script overhead (typically setup commands)
	beforeScriptFactor := float64(len(job.BeforeScript)) * 2.0 // 2 seconds per before_script line

	if len(job.Services) > 0 {
		baseDuration += 15.0 // Additional time for services
	}

	return baseDuration + scriptFactor + beforeScriptFactor
}

// estimateJobDurationWithContext considers template inheritance for more accurate estimation
func estimateJobDurationWithContext(job *parser.JobConfig, allJobs map[string]*parser.JobConfig) float64 {
	baseDuration := 30.0                           // 30 seconds base
	scriptFactor := float64(len(job.Script)) * 2.0 // 2 seconds per script line

	// Calculate before_script - either direct or from template
	beforeScriptLines := len(job.BeforeScript)

	// If job uses extends, get before_script from template
	extendsTemplates := extractExtendsTemplates(job.Extends)
	if len(extendsTemplates) > 0 {
		for _, templateName := range extendsTemplates {
			if template, exists := allJobs[templateName]; exists && template != nil {
				beforeScriptLines += len(template.BeforeScript)
			}
		}
	}

	beforeScriptFactor := float64(beforeScriptLines) * 2.0

	if len(job.Services) > 0 {
		baseDuration += 15.0 // Additional time for services
	}

	// Optimization bonus: if using templates, reduce overhead slightly due to better caching/reuse
	optimizationBonus := 0.0
	if len(extendsTemplates) > 0 {
		optimizationBonus = 3.0 // Small improvement from template reuse
	}

	duration := baseDuration + scriptFactor + beforeScriptFactor - optimizationBonus
	if duration < 10.0 {
		duration = 10.0 // Minimum duration
	}

	return duration
}

func extractExtendsTemplates(extends interface{}) []string {
	if extends == nil {
		return []string{}
	}

	// Handle different extend formats
	switch v := extends.(type) {
	case string:
		return []string{v}
	case []string:
		return v
	case []interface{}:
		templates := make([]string, 0, len(v))
		for _, item := range v {
			if str, ok := item.(string); ok {
				templates = append(templates, str)
			}
		}
		return templates
	default:
		return []string{}
	}
}

func getStageOrder(stageName string, stages []string) int {
	for i, stage := range stages {
		if stage == stageName {
			return i
		}
	}
	return 999 // Unknown stage goes last
}

func (r *Renderer) calculateAverageJobDuration(jobs []JobExecution) float64 {
	if len(jobs) == 0 {
		return 0
	}

	total := 0.0
	for _, job := range jobs {
		total += job.Duration
	}

	return total / float64(len(jobs))
}

func (r *Renderer) calculateParallelismImprovement(oldPipeline, newPipeline *PipelineExecution) int {
	// Simple heuristic: count maximum concurrent jobs per stage
	oldParallel := r.countMaxConcurrentJobs(oldPipeline.Jobs)
	newParallel := r.countMaxConcurrentJobs(newPipeline.Jobs)
	return newParallel - oldParallel
}

func (r *Renderer) calculateStartupTimeReduction(oldPipeline, newPipeline *PipelineExecution) float64 {
	oldAvgQueue := 0.0
	newAvgQueue := 0.0

	if len(oldPipeline.Jobs) > 0 {
		for _, job := range oldPipeline.Jobs {
			oldAvgQueue += job.QueuedDuration
		}
		oldAvgQueue /= float64(len(oldPipeline.Jobs))
	}

	if len(newPipeline.Jobs) > 0 {
		for _, job := range newPipeline.Jobs {
			newAvgQueue += job.QueuedDuration
		}
		newAvgQueue /= float64(len(newPipeline.Jobs))
	}

	return oldAvgQueue - newAvgQueue
}

func (r *Renderer) countMaxConcurrentJobs(jobs []JobExecution) int {
	stageJobs := make(map[string]int)
	for _, job := range jobs {
		stageJobs[job.Stage]++
	}

	max := 0
	for _, count := range stageJobs {
		if count > max {
			max = count
		}
	}

	return max
}

func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	aCopy := make([]string, len(a))
	bCopy := make([]string, len(b))
	copy(aCopy, a)
	copy(bCopy, b)

	sort.Strings(aCopy)
	sort.Strings(bCopy)

	for i := range aCopy {
		if aCopy[i] != bCopy[i] {
			return false
		}
	}

	return true
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

// FormatComparison formats a pipeline comparison for display
func (r *Renderer) FormatComparison(comparison *PipelineComparison, format string) (string, error) {
	switch format {
	case "json":
		data, err := json.MarshalIndent(comparison, "", "  ")
		if err != nil {
			return "", err
		}
		return string(data), nil

	case "table", "":
		return r.formatComparisonTable(comparison), nil

	case "dot", "mermaid":
		// Visual formats require configuration data, which isn't available here
		// These should be handled by RenderVisualComparison instead
		return "", fmt.Errorf("visual format %s requires using RenderVisualComparison with configuration data", format)

	default:
		return "", fmt.Errorf("unsupported format: %s (supported: json, table, dot, mermaid)", format)
	}
}

// formatComparisonTable formats comparison as a table
func (r *Renderer) formatComparisonTable(comparison *PipelineComparison) string {
	var buf bytes.Buffer

	buf.WriteString("Pipeline Execution Comparison\n")
	buf.WriteString("============================\n\n")

	// Summary section
	summary := comparison.Summary
	buf.WriteString("Summary:\n")
	buf.WriteString("--------\n")
	buf.WriteString(fmt.Sprintf("  Total Jobs: %d\n", summary.TotalJobs))
	buf.WriteString(fmt.Sprintf("  Added Jobs: %d\n", summary.AddedJobs))
	buf.WriteString(fmt.Sprintf("  Removed Jobs: %d\n", summary.RemovedJobs))
	buf.WriteString(fmt.Sprintf("  Improved Jobs: %d\n", summary.ImprovedJobs))
	buf.WriteString(fmt.Sprintf("  Degraded Jobs: %d\n", summary.DegradedJobs))
	buf.WriteString(fmt.Sprintf("  Identical Jobs: %d\n", summary.IdenticalJobs))
	buf.WriteString(fmt.Sprintf("  Total Time Change: %.2fs\n", summary.TotalTimeChange))

	if summary.OverallImprovement {
		buf.WriteString("  Overall: ✓ Performance improved\n")
	} else {
		buf.WriteString("  Overall: ⚠ Performance degraded\n")
	}

	// Performance metrics
	perf := comparison.PerformanceGain
	buf.WriteString("\nPerformance Metrics:\n")
	buf.WriteString("-------------------\n")
	buf.WriteString(fmt.Sprintf("  Pipeline Duration Change: %.2fs\n", perf.TotalPipelineDuration))
	buf.WriteString(fmt.Sprintf("  Average Job Duration Change: %.2fs\n", perf.AverageJobDuration))
	buf.WriteString(fmt.Sprintf("  Parallelism Improvement: %d jobs\n", perf.ParallelismImprovement))
	buf.WriteString(fmt.Sprintf("  Startup Time Reduction: %.2fs\n", perf.StartupTimeReduction))

	// Job-by-job comparison
	buf.WriteString("\nJob Comparisons:\n")
	buf.WriteString("---------------\n")

	for _, jobComp := range comparison.JobComparisons {
		status := r.formatJobStatus(jobComp.Status)
		buf.WriteString(fmt.Sprintf("  [%s] %s: ", status, jobComp.JobName))

		switch jobComp.Status {
		case StatusAdded:
			buf.WriteString("Job added\n")
		case StatusRemoved:
			buf.WriteString("Job removed\n")
		case StatusIdentical:
			buf.WriteString("No changes\n")
		default:
			buf.WriteString(fmt.Sprintf("Duration change: %.2fs", jobComp.DurationChange))
			if len(jobComp.Changes) > 0 {
				buf.WriteString(fmt.Sprintf(" (%s)", strings.Join(jobComp.Changes, ", ")))
			}
			buf.WriteString("\n")
		}
	}

	return buf.String()
}

func (r *Renderer) formatJobStatus(status CompareStatus) string {
	switch status {
	case StatusIdentical:
		return "="
	case StatusImproved:
		return "✓"
	case StatusDegraded:
		return "⚠"
	case StatusAdded:
		return "+"
	case StatusRemoved:
		return "-"
	case StatusRestructured:
		return "~"
	default:
		return "?"
	}
}
