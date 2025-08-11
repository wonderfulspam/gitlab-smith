package types

import (
	"github.com/wonderfulspam/gitlab-smith/pkg/parser"
)

type IssueType string

const (
	IssueTypePerformance     IssueType = "performance"
	IssueTypeSecurity        IssueType = "security"
	IssueTypeMaintainability IssueType = "maintainability"
	IssueTypeReliability     IssueType = "reliability"
)

type Severity string

const (
	SeverityLow    Severity = "low"
	SeverityMedium Severity = "medium"
	SeverityHigh   Severity = "high"
)

type Issue struct {
	Type       IssueType `json:"type"`
	Severity   Severity  `json:"severity"`
	Path       string    `json:"path"`
	Message    string    `json:"message"`
	Suggestion string    `json:"suggestion,omitempty"`
	JobName    string    `json:"job_name,omitempty"`
}

type AnalysisResult struct {
	Issues      []Issue `json:"issues"`
	TotalIssues int     `json:"total_issues"`
	Summary     Summary `json:"summary"`
}

type Summary struct {
	Performance     int `json:"performance"`
	Security        int `json:"security"`
	Maintainability int `json:"maintainability"`
	Reliability     int `json:"reliability"`
}

// CheckFunc is a function type for check functions
type CheckFunc func(config *parser.GitLabConfig) []Issue

// CheckConfig holds configuration for individual checks
type CheckConfig struct {
	Name        string    `yaml:"name" json:"name"`
	Type        IssueType `yaml:"type" json:"type"`
	Enabled     bool      `yaml:"enabled" json:"enabled"`
	Severity    Severity  `yaml:"severity,omitempty" json:"severity,omitempty"`
	Description string    `yaml:"description,omitempty" json:"description,omitempty"`
}

func (r *AnalysisResult) FilterBySeverity(severity Severity) []Issue {
	var filtered []Issue
	for _, issue := range r.Issues {
		if issue.Severity == severity {
			filtered = append(filtered, issue)
		}
	}
	return filtered
}

func (r *AnalysisResult) FilterByType(issueType IssueType) []Issue {
	var filtered []Issue
	for _, issue := range r.Issues {
		if issue.Type == issueType {
			filtered = append(filtered, issue)
		}
	}
	return filtered
}

func CalculateSummary(issues []Issue) Summary {
	summary := Summary{}

	for _, issue := range issues {
		switch issue.Type {
		case IssueTypePerformance:
			summary.Performance++
		case IssueTypeSecurity:
			summary.Security++
		case IssueTypeMaintainability:
			summary.Maintainability++
		case IssueTypeReliability:
			summary.Reliability++
		}
	}

	return summary
}
