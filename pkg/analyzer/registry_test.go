package analyzer

import (
	"testing"

	"github.com/wonderfulspam/gitlab-smith/pkg/analyzer/types"
	"github.com/wonderfulspam/gitlab-smith/pkg/parser"
)

func TestCheckRegistry(t *testing.T) {
	registry := NewCheckRegistry()

	// Test that registry is properly initialized
	if registry == nil {
		t.Fatal("NewCheckRegistry() returned nil")
	}
	if registry.checks == nil {
		t.Fatal("registry.checks map is nil")
	}

	// Test initially empty registry
	checks := registry.GetChecks()
	if len(checks) != 0 {
		t.Errorf("Expected 0 checks in new registry, got %d", len(checks))
	}
}

func TestCheckRegistryRegister(t *testing.T) {
	registry := NewCheckRegistry()

	// Mock check function
	mockCheckFunc := func(config *parser.GitLabConfig) []types.Issue {
		return []types.Issue{
			{
				Type:     types.IssueTypePerformance,
				Severity: types.SeverityMedium,
				Message:  "Test issue",
				Path:     "test_job",
			},
		}
	}

	// Register a check
	registry.Register("test_check", types.IssueTypePerformance, mockCheckFunc)

	checks := registry.GetChecks()
	if len(checks) != 1 {
		t.Errorf("Expected 1 check after registration, got %d", len(checks))
	}

	check := checks[0]
	if check.Name() != "test_check" {
		t.Errorf("Expected check name 'test_check', got '%s'", check.Name())
	}
	if check.Type() != types.IssueTypePerformance {
		t.Errorf("Expected check type Performance, got %v", check.Type())
	}
	if !check.Enabled() {
		t.Error("Expected check to be enabled by default")
	}
}

func TestCheckRegistryRegisterMultiple(t *testing.T) {
	registry := NewCheckRegistry()

	mockCheckFunc1 := func(config *parser.GitLabConfig) []types.Issue {
		return []types.Issue{}
	}
	mockCheckFunc2 := func(config *parser.GitLabConfig) []types.Issue {
		return []types.Issue{}
	}

	registry.Register("check1", types.IssueTypePerformance, mockCheckFunc1)
	registry.Register("check2", types.IssueTypeSecurity, mockCheckFunc2)

	checks := registry.GetChecks()
	if len(checks) != 2 {
		t.Errorf("Expected 2 checks, got %d", len(checks))
	}

	// Verify both checks exist (order not guaranteed)
	checkNames := make(map[string]bool)
	checkTypes := make(map[types.IssueType]bool)

	for _, check := range checks {
		checkNames[check.Name()] = true
		checkTypes[check.Type()] = true
	}

	if !checkNames["check1"] {
		t.Error("Expected 'check1' to be registered")
	}
	if !checkNames["check2"] {
		t.Error("Expected 'check2' to be registered")
	}
	if !checkTypes[types.IssueTypePerformance] {
		t.Error("Expected Performance check type")
	}
	if !checkTypes[types.IssueTypeSecurity] {
		t.Error("Expected Security check type")
	}
}

func TestCheckRegistryGetChecksByType(t *testing.T) {
	registry := NewCheckRegistry()

	mockCheckFunc := func(config *parser.GitLabConfig) []types.Issue {
		return []types.Issue{}
	}

	// Register checks of different types
	registry.Register("perf1", types.IssueTypePerformance, mockCheckFunc)
	registry.Register("perf2", types.IssueTypePerformance, mockCheckFunc)
	registry.Register("security1", types.IssueTypeSecurity, mockCheckFunc)
	registry.Register("maintainability1", types.IssueTypeMaintainability, mockCheckFunc)

	// Test getting performance checks
	perfChecks := registry.GetChecksByType(types.IssueTypePerformance)
	if len(perfChecks) != 2 {
		t.Errorf("Expected 2 performance checks, got %d", len(perfChecks))
	}

	// Test getting security checks
	secChecks := registry.GetChecksByType(types.IssueTypeSecurity)
	if len(secChecks) != 1 {
		t.Errorf("Expected 1 security check, got %d", len(secChecks))
	}

	// Test getting reliability checks (should be empty)
	relChecks := registry.GetChecksByType(types.IssueTypeReliability)
	if len(relChecks) != 0 {
		t.Errorf("Expected 0 reliability checks, got %d", len(relChecks))
	}
}

func TestBaseCheckerOperations(t *testing.T) {
	mockCheckFunc := func(config *parser.GitLabConfig) []types.Issue {
		return []types.Issue{
			{
				Type:     types.IssueTypePerformance,
				Severity: types.SeverityHigh,
				Message:  "Test performance issue",
				Path:     "test_job",
			},
		}
	}

	checker := NewBaseChecker("test_check", types.IssueTypePerformance, mockCheckFunc)

	// Test basic properties
	if checker.Name() != "test_check" {
		t.Errorf("Expected name 'test_check', got '%s'", checker.Name())
	}
	if checker.Type() != types.IssueTypePerformance {
		t.Errorf("Expected type Performance, got %v", checker.Type())
	}
	if !checker.Enabled() {
		t.Error("Expected checker to be enabled by default")
	}

	// Test check execution
	mockConfig := &parser.GitLabConfig{
		Jobs: map[string]*parser.JobConfig{
			"test_job": {
				Stage:  "test",
				Script: []string{"echo test"},
			},
		},
	}

	issues := checker.Check(mockConfig)
	if len(issues) != 1 {
		t.Errorf("Expected 1 issue from check, got %d", len(issues))
	}
	if len(issues) > 0 && issues[0].Message != "Test performance issue" {
		t.Errorf("Expected 'Test performance issue', got '%s'", issues[0].Message)
	}
}

func TestBaseCheckerDisabled(t *testing.T) {
	mockCheckFunc := func(config *parser.GitLabConfig) []types.Issue {
		return []types.Issue{
			{
				Type:     types.IssueTypePerformance,
				Severity: types.SeverityHigh,
				Message:  "This should not appear",
				Path:     "test_job",
			},
		}
	}

	checker := NewBaseChecker("disabled_check", types.IssueTypePerformance, mockCheckFunc)
	checker.SetEnabled(false)

	if checker.Enabled() {
		t.Error("Expected checker to be disabled")
	}

	mockConfig := &parser.GitLabConfig{
		Jobs: map[string]*parser.JobConfig{
			"test_job": {
				Stage:  "test",
				Script: []string{"echo test"},
			},
		},
	}

	issues := checker.Check(mockConfig)
	if len(issues) != 0 {
		t.Errorf("Expected 0 issues from disabled check, got %d", len(issues))
	}
}

func TestBaseCheckerSetDescription(t *testing.T) {
	mockCheckFunc := func(config *parser.GitLabConfig) []types.Issue {
		return []types.Issue{}
	}

	checker := NewBaseChecker("test_check", types.IssueTypePerformance, mockCheckFunc)

	// Test setting description
	description := "This is a test check for performance issues"
	checker.SetDescription(description)

	// Note: The description field is private, so we can't directly test it
	// but we can verify the method doesn't panic and the checker still works
	if checker.Name() != "test_check" {
		t.Error("Checker name should remain unchanged after setting description")
	}
	if checker.Type() != types.IssueTypePerformance {
		t.Error("Checker type should remain unchanged after setting description")
	}
}

func TestBaseCheckerEnableDisable(t *testing.T) {
	mockCheckFunc := func(config *parser.GitLabConfig) []types.Issue {
		return []types.Issue{}
	}

	checker := NewBaseChecker("toggle_check", types.IssueTypePerformance, mockCheckFunc)

	// Initially enabled
	if !checker.Enabled() {
		t.Error("Checker should be enabled by default")
	}

	// Disable
	checker.SetEnabled(false)
	if checker.Enabled() {
		t.Error("Checker should be disabled after SetEnabled(false)")
	}

	// Re-enable
	checker.SetEnabled(true)
	if !checker.Enabled() {
		t.Error("Checker should be enabled after SetEnabled(true)")
	}
}
