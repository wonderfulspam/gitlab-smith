package types

import (
	"testing"

	"github.com/wonderfulspam/gitlab-smith/pkg/parser"
)

func TestIssueConstants(t *testing.T) {
	if IssueTypePerformance != IssueType("performance") {
		t.Errorf("Expected IssueTypePerformance to be 'performance', got %s", IssueTypePerformance)
	}
	if IssueTypeSecurity != IssueType("security") {
		t.Errorf("Expected IssueTypeSecurity to be 'security', got %s", IssueTypeSecurity)
	}
	if IssueTypeMaintainability != IssueType("maintainability") {
		t.Errorf("Expected IssueTypeMaintainability to be 'maintainability', got %s", IssueTypeMaintainability)
	}
	if IssueTypeReliability != IssueType("reliability") {
		t.Errorf("Expected IssueTypeReliability to be 'reliability', got %s", IssueTypeReliability)
	}
}

func TestSeverityConstants(t *testing.T) {
	if SeverityLow != Severity("low") {
		t.Errorf("Expected SeverityLow to be 'low', got %s", SeverityLow)
	}
	if SeverityMedium != Severity("medium") {
		t.Errorf("Expected SeverityMedium to be 'medium', got %s", SeverityMedium)
	}
	if SeverityHigh != Severity("high") {
		t.Errorf("Expected SeverityHigh to be 'high', got %s", SeverityHigh)
	}
}

func TestAnalysisResult_FilterBySeverity(t *testing.T) {
	result := &AnalysisResult{
		Issues: []Issue{
			{Type: IssueTypePerformance, Severity: SeverityHigh, Message: "High perf issue"},
			{Type: IssueTypeSecurity, Severity: SeverityLow, Message: "Low security issue"},
			{Type: IssueTypeReliability, Severity: SeverityHigh, Message: "High reliability issue"},
			{Type: IssueTypeMaintainability, Severity: SeverityMedium, Message: "Medium maint issue"},
		},
	}

	highIssues := result.FilterBySeverity(SeverityHigh)
	if len(highIssues) != 2 {
		t.Errorf("Expected 2 high severity issues, got %d", len(highIssues))
	}
	if len(highIssues) > 0 && highIssues[0].Message != "High perf issue" {
		t.Errorf("Expected first high issue to be 'High perf issue', got %s", highIssues[0].Message)
	}
	if len(highIssues) > 1 && highIssues[1].Message != "High reliability issue" {
		t.Errorf("Expected second high issue to be 'High reliability issue', got %s", highIssues[1].Message)
	}

	lowIssues := result.FilterBySeverity(SeverityLow)
	if len(lowIssues) != 1 {
		t.Errorf("Expected 1 low severity issue, got %d", len(lowIssues))
	}
	if len(lowIssues) > 0 && lowIssues[0].Message != "Low security issue" {
		t.Errorf("Expected low issue to be 'Low security issue', got %s", lowIssues[0].Message)
	}

	mediumIssues := result.FilterBySeverity(SeverityMedium)
	if len(mediumIssues) != 1 {
		t.Errorf("Expected 1 medium severity issue, got %d", len(mediumIssues))
	}
	if len(mediumIssues) > 0 && mediumIssues[0].Message != "Medium maint issue" {
		t.Errorf("Expected medium issue to be 'Medium maint issue', got %s", mediumIssues[0].Message)
	}
}

func TestAnalysisResult_FilterByType(t *testing.T) {
	result := &AnalysisResult{
		Issues: []Issue{
			{Type: IssueTypePerformance, Severity: SeverityHigh, Message: "Perf issue 1"},
			{Type: IssueTypeSecurity, Severity: SeverityLow, Message: "Security issue 1"},
			{Type: IssueTypePerformance, Severity: SeverityMedium, Message: "Perf issue 2"},
			{Type: IssueTypeMaintainability, Severity: SeverityMedium, Message: "Maint issue 1"},
		},
	}

	perfIssues := result.FilterByType(IssueTypePerformance)
	if len(perfIssues) != 2 {
		t.Errorf("Expected 2 performance issues, got %d", len(perfIssues))
	}
	if len(perfIssues) > 0 && perfIssues[0].Message != "Perf issue 1" {
		t.Errorf("Expected first perf issue to be 'Perf issue 1', got %s", perfIssues[0].Message)
	}
	if len(perfIssues) > 1 && perfIssues[1].Message != "Perf issue 2" {
		t.Errorf("Expected second perf issue to be 'Perf issue 2', got %s", perfIssues[1].Message)
	}

	secIssues := result.FilterByType(IssueTypeSecurity)
	if len(secIssues) != 1 {
		t.Errorf("Expected 1 security issue, got %d", len(secIssues))
	}
	if len(secIssues) > 0 && secIssues[0].Message != "Security issue 1" {
		t.Errorf("Expected security issue to be 'Security issue 1', got %s", secIssues[0].Message)
	}

	maintIssues := result.FilterByType(IssueTypeMaintainability)
	if len(maintIssues) != 1 {
		t.Errorf("Expected 1 maintainability issue, got %d", len(maintIssues))
	}
	if len(maintIssues) > 0 && maintIssues[0].Message != "Maint issue 1" {
		t.Errorf("Expected maintainability issue to be 'Maint issue 1', got %s", maintIssues[0].Message)
	}

	reliabilityIssues := result.FilterByType(IssueTypeReliability)
	if len(reliabilityIssues) != 0 {
		t.Errorf("Expected 0 reliability issues, got %d", len(reliabilityIssues))
	}
}

func TestCalculateSummary(t *testing.T) {
	issues := []Issue{
		{Type: IssueTypePerformance, Severity: SeverityHigh},
		{Type: IssueTypePerformance, Severity: SeverityMedium},
		{Type: IssueTypeSecurity, Severity: SeverityLow},
		{Type: IssueTypeReliability, Severity: SeverityHigh},
		{Type: IssueTypeMaintainability, Severity: SeverityMedium},
		{Type: IssueTypeMaintainability, Severity: SeverityLow},
	}

	summary := CalculateSummary(issues)

	if summary.Performance != 2 {
		t.Errorf("Expected 2 performance issues, got %d", summary.Performance)
	}
	if summary.Security != 1 {
		t.Errorf("Expected 1 security issue, got %d", summary.Security)
	}
	if summary.Reliability != 1 {
		t.Errorf("Expected 1 reliability issue, got %d", summary.Reliability)
	}
	if summary.Maintainability != 2 {
		t.Errorf("Expected 2 maintainability issues, got %d", summary.Maintainability)
	}
}

func TestCalculateSummary_EmptyIssues(t *testing.T) {
	summary := CalculateSummary([]Issue{})

	if summary.Performance != 0 {
		t.Errorf("Expected 0 performance issues, got %d", summary.Performance)
	}
	if summary.Security != 0 {
		t.Errorf("Expected 0 security issues, got %d", summary.Security)
	}
	if summary.Reliability != 0 {
		t.Errorf("Expected 0 reliability issues, got %d", summary.Reliability)
	}
	if summary.Maintainability != 0 {
		t.Errorf("Expected 0 maintainability issues, got %d", summary.Maintainability)
	}
}

func TestCheckFunc(t *testing.T) {
	// Test that CheckFunc type works as expected
	var checkFunc CheckFunc = func(config *parser.GitLabConfig) []Issue {
		return []Issue{
			{Type: IssueTypePerformance, Severity: SeverityHigh, Message: "Test issue"},
		}
	}

	config := &parser.GitLabConfig{}
	issues := checkFunc(config)

	if len(issues) != 1 {
		t.Errorf("Expected 1 issue, got %d", len(issues))
	}
	if len(issues) > 0 && issues[0].Type != IssueTypePerformance {
		t.Errorf("Expected issue type to be performance, got %s", issues[0].Type)
	}
	if len(issues) > 0 && issues[0].Severity != SeverityHigh {
		t.Errorf("Expected issue severity to be high, got %s", issues[0].Severity)
	}
	if len(issues) > 0 && issues[0].Message != "Test issue" {
		t.Errorf("Expected issue message to be 'Test issue', got %s", issues[0].Message)
	}
}
