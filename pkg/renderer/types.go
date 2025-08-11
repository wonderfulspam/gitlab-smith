package renderer

import (
	"net/http"
	"time"
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
