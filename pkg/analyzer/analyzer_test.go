package analyzer

import (
	"strings"
	"testing"

	"github.com/emt/gitlab-smith/pkg/parser"
)

func TestAnalyze_EmptyConfig(t *testing.T) {
	config := &parser.GitLabConfig{
		Jobs: make(map[string]*parser.JobConfig),
	}

	result := Analyze(config)

	if result.TotalIssues != len(result.Issues) {
		t.Errorf("TotalIssues (%d) doesn't match actual issues count (%d)", result.TotalIssues, len(result.Issues))
	}

	// Empty config should have at least the missing stages issue
	if result.TotalIssues == 0 {
		t.Error("Expected at least one issue for empty config")
	}
}

func TestNewAnalyzer(t *testing.T) {
	analyzer := New()
	
	if analyzer == nil {
		t.Fatal("Expected analyzer to be created")
	}

	if analyzer.registry == nil {
		t.Error("Expected registry to be initialized")
	}

	if analyzer.config == nil {
		t.Error("Expected config to be initialized")
	}

	// Check that checks are registered
	checks := analyzer.registry.GetChecks()
	if len(checks) == 0 {
		t.Error("Expected checks to be registered")
	}

	// Verify we have checks of different types
	hasPerformance := false
	hasSecurity := false
	hasMaintainability := false
	hasReliability := false

	for _, check := range checks {
		switch check.Type() {
		case IssueTypePerformance:
			hasPerformance = true
		case IssueTypeSecurity:
			hasSecurity = true
		case IssueTypeMaintainability:
			hasMaintainability = true
		case IssueTypeReliability:
			hasReliability = true
		}
	}

	if !hasPerformance {
		t.Error("Expected performance checks to be registered")
	}
	if !hasSecurity {
		t.Error("Expected security checks to be registered")
	}
	if !hasMaintainability {
		t.Error("Expected maintainability checks to be registered")
	}
	if !hasReliability {
		t.Error("Expected reliability checks to be registered")
	}
}

func TestNewWithConfig(t *testing.T) {
	config := DefaultConfig()
	config.DisableCheck("cache_usage")
	
	analyzer := NewWithConfig(config)
	
	// Test that the specific check is disabled
	result := analyzer.Analyze(&parser.GitLabConfig{
		Jobs: map[string]*parser.JobConfig{
			"build": {Stage: "build"}, // This would normally trigger cache_usage issue
		},
	})

	// Should not contain cache_usage issues
	for _, issue := range result.Issues {
		if strings.Contains(issue.Message, "cache") {
			t.Error("Expected cache_usage check to be disabled")
		}
	}
}

func TestAnalyzerEnableDisableCheck(t *testing.T) {
	analyzer := New()
	
	// Disable a check
	analyzer.DisableCheck("job_naming")
	
	// Test with a config that would trigger job_naming issues
	config := &parser.GitLabConfig{
		Jobs: map[string]*parser.JobConfig{
			"build project": { // Job name with spaces should trigger issue
				Stage: "build",
			},
		},
	}

	result := analyzer.Analyze(config)
	
	// Should not contain job_naming issues
	for _, issue := range result.Issues {
		if strings.Contains(issue.Message, "spaces") {
			t.Error("Expected job_naming check to be disabled")
		}
	}

	// Re-enable the check
	analyzer.EnableCheck("job_naming")
	
	result = analyzer.Analyze(config)
	
	// Should now contain job_naming issues
	found := false
	for _, issue := range result.Issues {
		if strings.Contains(issue.Message, "spaces") {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected job_naming issue after re-enabling check")
	}
}

func TestAnalyzeWithFilter(t *testing.T) {
	analyzer := New()
	
	config := &parser.GitLabConfig{
		Variables: map[string]interface{}{
			"API_PASSWORD": "secret123", // Should trigger security issue
		},
		Jobs: map[string]*parser.JobConfig{
			"build project": { // Should trigger maintainability issue (job naming)
				Stage: "build",
			},
			"test": {
				Stage: "test",
				Cache: &parser.Cache{
					Paths: []string{"node_modules/"}, // Should trigger performance issue (missing key)
				},
			},
		},
	}

	// Test filtering by security only
	result := analyzer.AnalyzeWithFilter(config, IssueTypeSecurity)
	
	securityIssuesFound := 0
	for _, issue := range result.Issues {
		if issue.Type == IssueTypeSecurity {
			securityIssuesFound++
		} else {
			t.Errorf("Found non-security issue when filtering by security: %s", issue.Type)
		}
	}
	
	if securityIssuesFound == 0 {
		t.Error("Expected to find security issues")
	}

	// Test filtering by multiple types
	result = analyzer.AnalyzeWithFilter(config, IssueTypeSecurity, IssueTypeMaintainability)
	
	validTypes := map[IssueType]bool{
		IssueTypeSecurity:        true,
		IssueTypeMaintainability: true,
	}
	
	for _, issue := range result.Issues {
		if !validTypes[issue.Type] {
			t.Errorf("Found unexpected issue type when filtering: %s", issue.Type)
		}
	}
}

func TestListChecks(t *testing.T) {
	analyzer := New()
	
	checks := analyzer.ListChecks()
	
	if len(checks) == 0 {
		t.Error("Expected checks to be listed")
	}

	// Verify check structure
	for _, check := range checks {
		if check.Name == "" {
			t.Error("Expected check to have a name")
		}
		if check.Type == "" {
			t.Error("Expected check to have a type")
		}
		if check.Description == "" {
			t.Error("Expected check to have a description")
		}
	}

	// Verify we have the expected check names
	checkNames := make(map[string]bool)
	for _, check := range checks {
		checkNames[check.Name] = true
	}

	expectedChecks := []string{
		"cache_usage", "job_naming", "image_tags", "retry_configuration",
		"script_complexity", "duplicated_code", "environment_variables",
	}

	for _, expected := range expectedChecks {
		if !checkNames[expected] {
			t.Errorf("Expected check '%s' to be listed", expected)
		}
	}
}

func TestCalculateSummary(t *testing.T) {
	issues := []Issue{
		{Type: IssueTypePerformance},
		{Type: IssueTypePerformance},
		{Type: IssueTypeSecurity},
		{Type: IssueTypeMaintainability},
		{Type: IssueTypeMaintainability},
		{Type: IssueTypeMaintainability},
		{Type: IssueTypeReliability},
	}

	summary := CalculateSummary(issues)

	if summary.Performance != 2 {
		t.Errorf("Expected 2 performance issues, got %d", summary.Performance)
	}

	if summary.Security != 1 {
		t.Errorf("Expected 1 security issue, got %d", summary.Security)
	}

	if summary.Maintainability != 3 {
		t.Errorf("Expected 3 maintainability issues, got %d", summary.Maintainability)
	}

	if summary.Reliability != 1 {
		t.Errorf("Expected 1 reliability issue, got %d", summary.Reliability)
	}
}

func TestFilterBySeverity(t *testing.T) {
	result := &AnalysisResult{
		Issues: []Issue{
			{Severity: SeverityHigh, Type: IssueTypeSecurity},
			{Severity: SeverityMedium, Type: IssueTypePerformance},
			{Severity: SeverityHigh, Type: IssueTypeReliability},
			{Severity: SeverityLow, Type: IssueTypeMaintainability},
		},
	}

	highSeverityIssues := result.FilterBySeverity(SeverityHigh)

	if len(highSeverityIssues) != 2 {
		t.Errorf("Expected 2 high severity issues, got %d", len(highSeverityIssues))
	}

	for _, issue := range highSeverityIssues {
		if issue.Severity != SeverityHigh {
			t.Errorf("Expected high severity, got %s", issue.Severity)
		}
	}
}

func TestFilterByType(t *testing.T) {
	result := &AnalysisResult{
		Issues: []Issue{
			{Type: IssueTypeSecurity, Severity: SeverityHigh},
			{Type: IssueTypePerformance, Severity: SeverityMedium},
			{Type: IssueTypeSecurity, Severity: SeverityLow},
			{Type: IssueTypeMaintainability, Severity: SeverityMedium},
		},
	}

	securityIssues := result.FilterByType(IssueTypeSecurity)

	if len(securityIssues) != 2 {
		t.Errorf("Expected 2 security issues, got %d", len(securityIssues))
	}

	for _, issue := range securityIssues {
		if issue.Type != IssueTypeSecurity {
			t.Errorf("Expected security issue, got %s", issue.Type)
		}
	}
}

func TestAnalyze_ComprehensiveConfig(t *testing.T) {
	config := &parser.GitLabConfig{
		Stages: []string{"build", "test", "deploy"},
		Variables: map[string]interface{}{
			"NODE_VERSION": "16",
			"API_SECRET":   "secret123", // Should trigger security issue
		},
		Jobs: map[string]*parser.JobConfig{
			"build project": { // Should trigger naming issue
				Stage: "build",
				Image: "node", // Should trigger image tag issue
				Script: []string{
					"npm install",
					"npm run build",
					"curl https://api.example.com/notify", // Should trigger hardcoded URL issue
				},
				Cache: &parser.Cache{
					Paths: []string{"node_modules/"}, // Should trigger cache key issue
				},
			},
			"test": {
				Stage:  "test",
				Script: make([]string, 15), // Should trigger complexity issue
				Retry: &parser.Retry{
					Max: 5, // Should trigger retry issue
				},
			},
		},
	}

	result := Analyze(config)

	if result.TotalIssues == 0 {
		t.Error("Expected issues to be found in comprehensive config")
	}

	if result.Summary.Performance == 0 {
		t.Error("Expected performance issues")
	}

	if result.Summary.Security == 0 {
		t.Error("Expected security issues")
	}

	if result.Summary.Maintainability == 0 {
		t.Error("Expected maintainability issues")
	}

	if result.Summary.Reliability == 0 {
		t.Error("Expected reliability issues")
	}

	// Verify total matches sum
	expectedTotal := result.Summary.Performance + result.Summary.Security + result.Summary.Maintainability + result.Summary.Reliability
	if result.TotalIssues != expectedTotal {
		t.Errorf("TotalIssues (%d) doesn't match summary totals (%d)", result.TotalIssues, expectedTotal)
	}
}

func TestRegistryOperations(t *testing.T) {
	registry := NewCheckRegistry()
	
	// Test registering a check
	registry.Register("test_check", IssueTypePerformance, func(config *parser.GitLabConfig) []Issue {
		return []Issue{
			{
				Type:     IssueTypePerformance,
				Severity: SeverityMedium,
				Message:  "Test issue",
			},
		}
	})
	
	checks := registry.GetChecks()
	found := false
	for _, check := range checks {
		if check.Name() == "test_check" {
			found = true
			break
		}
	}
	
	if !found {
		t.Error("Expected test check to be registered")
	}
	
	// Test getting checks by type
	performanceChecks := registry.GetChecksByType(IssueTypePerformance)
	if len(performanceChecks) == 0 {
		t.Error("Expected to find performance checks")
	}
}

func TestBaseChecker(t *testing.T) {
	checkFunc := func(config *parser.GitLabConfig) []Issue {
		return []Issue{
			{
				Type:     IssueTypeSecurity,
				Severity: SeverityHigh,
				Message:  "Test security issue",
			},
		}
	}
	
	checker := NewBaseChecker("security_test", IssueTypeSecurity, checkFunc)
	
	if checker.Name() != "security_test" {
		t.Errorf("Expected name 'security_test', got '%s'", checker.Name())
	}
	
	if checker.Type() != IssueTypeSecurity {
		t.Errorf("Expected type %s, got %s", IssueTypeSecurity, checker.Type())
	}
	
	if !checker.Enabled() {
		t.Error("Expected checker to be enabled by default")
	}
	
	// Test disabling
	checker.SetEnabled(false)
	if checker.Enabled() {
		t.Error("Expected checker to be disabled")
	}
	
	// Test that disabled checker returns no issues
	config := &parser.GitLabConfig{}
	issues := checker.Check(config)
	if len(issues) != 0 {
		t.Error("Expected disabled checker to return no issues")
	}
	
	// Test re-enabling
	checker.SetEnabled(true)
	issues = checker.Check(config)
	if len(issues) != 1 {
		t.Errorf("Expected 1 issue, got %d", len(issues))
	}
	
	if issues[0].Type != IssueTypeSecurity {
		t.Errorf("Expected security issue, got %s", issues[0].Type)
	}
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()
	
	if config == nil {
		t.Fatal("Expected default config to be created")
	}
	
	if len(config.Checks) == 0 {
		t.Error("Expected default config to have checks")
	}
	
	// Verify some expected checks exist
	expectedChecks := []string{
		"cache_usage", "job_naming", "image_tags", "retry_configuration",
	}
	
	for _, expected := range expectedChecks {
		if checkConfig, exists := config.Checks[expected]; !exists {
			t.Errorf("Expected check '%s' to be in default config", expected)
		} else {
			if !checkConfig.Enabled {
				t.Errorf("Expected check '%s' to be enabled by default", expected)
			}
			if checkConfig.Name != expected {
				t.Errorf("Expected check name to be '%s', got '%s'", expected, checkConfig.Name)
			}
		}
	}
}

func TestConfigOperations(t *testing.T) {
	config := DefaultConfig()
	
	// Test IsCheckEnabled
	if !config.IsCheckEnabled("cache_usage") {
		t.Error("Expected cache_usage to be enabled")
	}
	
	if config.IsCheckEnabled("nonexistent_check") {
		t.Error("Expected nonexistent check to be disabled")
	}
	
	// Test DisableCheck
	config.DisableCheck("cache_usage")
	if config.IsCheckEnabled("cache_usage") {
		t.Error("Expected cache_usage to be disabled")
	}
	
	// Test EnableCheck
	config.EnableCheck("cache_usage")
	if !config.IsCheckEnabled("cache_usage") {
		t.Error("Expected cache_usage to be re-enabled")
	}
	
	// Test GetEnabledChecks
	enabledChecks := config.GetEnabledChecks()
	if len(enabledChecks) == 0 {
		t.Error("Expected enabled checks")
	}
	
	// Test GetChecksByType
	performanceChecks := config.GetChecksByType(IssueTypePerformance)
	if len(performanceChecks) == 0 {
		t.Error("Expected performance checks")
	}
	
	securityChecks := config.GetChecksByType(IssueTypeSecurity)
	if len(securityChecks) == 0 {
		t.Error("Expected security checks")
	}
}