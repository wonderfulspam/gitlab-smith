package analyzer

import (
	"github.com/emt/gitlab-smith/pkg/analyzer/types"
)

// Re-export types for backwards compatibility
type IssueType = types.IssueType
type Severity = types.Severity
type Issue = types.Issue
type AnalysisResult = types.AnalysisResult
type Summary = types.Summary
type CheckFunc = types.CheckFunc
type CheckConfig = types.CheckConfig

// Re-export constants
const (
	IssueTypePerformance     = types.IssueTypePerformance
	IssueTypeSecurity        = types.IssueTypeSecurity
	IssueTypeMaintainability = types.IssueTypeMaintainability
	IssueTypeReliability     = types.IssueTypeReliability
)

const (
	SeverityLow    = types.SeverityLow
	SeverityMedium = types.SeverityMedium
	SeverityHigh   = types.SeverityHigh
)

// Re-export functions for backwards compatibility
var CalculateSummary = types.CalculateSummary