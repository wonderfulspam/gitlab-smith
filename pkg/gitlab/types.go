package gitlab

import (
	"time"
)

// Pipeline represents a GitLab CI pipeline
type Pipeline struct {
	ID        int                    `json:"id"`
	Status    string                 `json:"status"`
	Ref       string                 `json:"ref"`
	SHA       string                 `json:"sha"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
	StartedAt *time.Time             `json:"started_at"`
	FinishedAt *time.Time            `json:"finished_at"`
	Duration  int                    `json:"duration"`
	Jobs      []*Job                 `json:"jobs,omitempty"`
	Variables map[string]string      `json:"variables,omitempty"`
}

// Job represents a GitLab CI job
type Job struct {
	ID         int                    `json:"id"`
	Name       string                 `json:"name"`
	Stage      string                 `json:"stage"`
	Status     string                 `json:"status"`
	CreatedAt  time.Time              `json:"created_at"`
	StartedAt  *time.Time             `json:"started_at"`
	FinishedAt *time.Time            `json:"finished_at"`
	Duration   float64                `json:"duration"`
	Runner     *Runner                `json:"runner,omitempty"`
	Artifacts  []Artifact             `json:"artifacts,omitempty"`
	Log        string                 `json:"log,omitempty"`
	When       string                 `json:"when,omitempty"`
	Dependencies []string             `json:"dependencies,omitempty"`
	Needs      []string               `json:"needs,omitempty"`
}

// Runner represents a GitLab runner
type Runner struct {
	ID          int    `json:"id"`
	Description string `json:"description"`
	Active      bool   `json:"active"`
	Tags        []string `json:"tags"`
}

// Artifact represents a job artifact
type Artifact struct {
	Filename string `json:"filename"`
	Size     int64  `json:"size"`
	Type     string `json:"file_type"`
}

// ValidationResult represents the result of config validation
type ValidationResult struct {
	Valid    bool     `json:"valid"`
	Errors   []string `json:"errors"`
	Warnings []string `json:"warnings"`
	Merged   string   `json:"merged_yaml,omitempty"`
}

// Project represents a GitLab project
type Project struct {
	ID                int    `json:"id"`
	Name              string `json:"name"`
	Path              string `json:"path"`
	DefaultBranch     string `json:"default_branch"`
	WebURL            string `json:"web_url"`
	NamespaceFullPath string `json:"namespace_full_path"`
}