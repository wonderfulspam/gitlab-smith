package analyzer

import (
	"github.com/wonderfulspam/gitlab-smith/pkg/analyzer/maintainability"
	"github.com/wonderfulspam/gitlab-smith/pkg/analyzer/performance"
	"github.com/wonderfulspam/gitlab-smith/pkg/analyzer/reliability"
	"github.com/wonderfulspam/gitlab-smith/pkg/analyzer/security"
	"github.com/wonderfulspam/gitlab-smith/pkg/analyzer/types"
	"github.com/wonderfulspam/gitlab-smith/pkg/parser"
)

// Analyzer manages the analysis process with configurable checks
type Analyzer struct {
	registry *CheckRegistry
	config   *Config
}

// New creates a new analyzer with default configuration
func New() *Analyzer {
	registry := NewCheckRegistry()
	config := DefaultConfig()

	// Register all checks
	performance.RegisterChecks(registry)
	security.RegisterChecks(registry)
	maintainability.RegisterChecks(registry)
	reliability.RegisterChecks(registry)

	return &Analyzer{
		registry: registry,
		config:   config,
	}
}

// NewWithConfig creates a new analyzer with custom configuration
func NewWithConfig(config *Config) *Analyzer {
	analyzer := New()
	analyzer.config = config

	// Update registry with config settings
	analyzer.applyConfig()

	return analyzer
}

// NewFromConfigFile creates a new analyzer loading config from file
func NewFromConfigFile(configFile string) (*Analyzer, error) {
	config, err := LoadOrCreateConfig(configFile)
	if err != nil {
		return nil, err
	}

	return NewWithConfig(config), nil
}

// applyConfig applies configuration settings to the registry
func (a *Analyzer) applyConfig() {
	for _, checker := range a.registry.GetChecks() {
		if checkConfig, exists := a.config.Checks[checker.Name()]; exists {
			if baseChecker, ok := checker.(*BaseChecker); ok {
				baseChecker.SetEnabled(checkConfig.Enabled)
				baseChecker.SetConfig(a.config) // Pass config reference
				if checkConfig.Description != "" {
					baseChecker.SetDescription(checkConfig.Description)
				}
			}
		}
	}
}

// Analyze performs analysis using configured checks
func (a *Analyzer) Analyze(config *parser.GitLabConfig) *types.AnalysisResult {
	result := &types.AnalysisResult{
		Issues: []types.Issue{},
	}

	// Run all enabled checks
	for _, checker := range a.registry.GetChecks() {
		if checker.Enabled() {
			issues := checker.Check(config)
			result.Issues = append(result.Issues, issues...)
		}
	}

	result.TotalIssues = len(result.Issues)
	result.Summary = types.CalculateSummary(result.Issues)

	return result
}

// AnalyzeWithFilter performs analysis with type filtering
func (a *Analyzer) AnalyzeWithFilter(config *parser.GitLabConfig, issueTypes ...types.IssueType) *types.AnalysisResult {
	result := &types.AnalysisResult{
		Issues: []types.Issue{},
	}

	// Create a map for quick lookup
	typeFilter := make(map[types.IssueType]bool)
	for _, t := range issueTypes {
		typeFilter[t] = true
	}

	// Run filtered checks
	for _, checker := range a.registry.GetChecks() {
		if checker.Enabled() && (len(typeFilter) == 0 || typeFilter[checker.Type()]) {
			issues := checker.Check(config)
			result.Issues = append(result.Issues, issues...)
		}
	}

	result.TotalIssues = len(result.Issues)
	result.Summary = types.CalculateSummary(result.Issues)

	return result
}

// EnableCheck enables a specific check
func (a *Analyzer) EnableCheck(checkName string) {
	a.config.EnableCheck(checkName)
	a.applyConfig()
}

// DisableCheck disables a specific check
func (a *Analyzer) DisableCheck(checkName string) {
	a.config.DisableCheck(checkName)
	a.applyConfig()
}

// GetConfig returns the current configuration
func (a *Analyzer) GetConfig() *Config {
	return a.config
}

// GetRegistry returns the check registry
func (a *Analyzer) GetRegistry() *CheckRegistry {
	return a.registry
}

// ListChecks returns information about all available checks
func (a *Analyzer) ListChecks() []types.CheckConfig {
	var checks []types.CheckConfig
	for _, checker := range a.registry.GetChecks() {
		if config, exists := a.config.Checks[checker.Name()]; exists {
			checks = append(checks, config)
		}
	}
	return checks
}

// Convenience function for backward compatibility
func Analyze(config *parser.GitLabConfig) *types.AnalysisResult {
	analyzer := New()
	return analyzer.Analyze(config)
}
