package testutil

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

// DiscoverScenarios discovers refactoring scenarios from the filesystem
func DiscoverScenarios(scenariosPath string) ([]*RefactoringScenario, error) {
	var scenarios []*RefactoringScenario

	entries, err := ioutil.ReadDir(scenariosPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read scenarios directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() || !strings.HasPrefix(entry.Name(), "scenario-") {
			continue
		}

		scenarioName := entry.Name()
		scenarioDir := filepath.Join(scenariosPath, scenarioName)

		beforeDir := filepath.Join(scenarioDir, "before")
		afterDir := filepath.Join(scenarioDir, "after")
		includesDir := filepath.Join(scenarioDir, "includes")

		if !dirExists(beforeDir) || !dirExists(afterDir) {
			continue
		}

		scenario := &RefactoringScenario{
			Name:         scenarioName,
			Description:  GenerateDescription(scenarioName),
			BeforeDir:    beforeDir,
			AfterDir:     afterDir,
			IncludesDir:  includesDir,
			Expectations: GetDefaultExpectations(scenarioName),
		}

		configPath := filepath.Join(scenarioDir, "config.yaml")
		if FileExists(configPath) {
			config, err := LoadScenarioConfig(configPath)
			if err != nil {
				return nil, fmt.Errorf("failed to load config for %s: %w", scenarioName, err)
			}
			ApplyScenarioConfig(scenario, config)
		}

		scenarios = append(scenarios, scenario)
	}

	return scenarios, nil
}

// DiscoverRealisticScenarios discovers realistic application scenarios
func DiscoverRealisticScenarios(scenariosPath string) ([]*RefactoringScenario, error) {
	var scenarios []*RefactoringScenario

	entries, err := ioutil.ReadDir(scenariosPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read realistic scenarios directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		scenarioName := entry.Name()
		scenarioDir := filepath.Join(scenariosPath, scenarioName)

		beforeDir := filepath.Join(scenarioDir, "before")
		afterDir := filepath.Join(scenarioDir, "after")

		if !dirExists(beforeDir) || !dirExists(afterDir) {
			continue
		}

		scenario := &RefactoringScenario{
			Name:         scenarioName,
			Description:  GenerateRealisticDescription(scenarioName),
			BeforeDir:    beforeDir,
			AfterDir:     afterDir,
			IncludesDir:  filepath.Join(scenarioDir, "includes"),
			Expectations: GetRealisticExpectations(scenarioName),
		}

		configPath := filepath.Join(scenarioDir, "config.yaml")
		if FileExists(configPath) {
			config, err := LoadScenarioConfig(configPath)
			if err != nil {
				return nil, fmt.Errorf("failed to load config for realistic scenario %s: %w", scenarioName, err)
			}
			ApplyScenarioConfig(scenario, config)
		}

		scenarios = append(scenarios, scenario)
	}

	return scenarios, nil
}

// LoadScenarioConfig loads scenario configuration from YAML file
func LoadScenarioConfig(configPath string) (*ScenarioConfig, error) {
	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var config ScenarioConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

// ApplyScenarioConfig applies configuration to a scenario
func ApplyScenarioConfig(scenario *RefactoringScenario, config *ScenarioConfig) {
	if config.Name != "" {
		scenario.Name = config.Name
	}
	if config.Description != "" {
		scenario.Description = config.Description
	}

	scenario.Expectations = RefactoringExpectations{
		ShouldSucceed:          config.Expectations.ShouldSucceed,
		ExpectedIssueReduction: config.Expectations.ExpectedIssueReduction,
		MaxAllowedNewIssues:    config.Expectations.MaxAllowedNewIssues,
		RequiredImprovements:   config.Expectations.RequiredImprovements,
		ForbiddenChanges:       config.Expectations.ForbiddenChanges,
		SemanticEquivalence:    config.Expectations.SemanticEquivalence,
		PerformanceImprovement: config.Expectations.PerformanceImprovement,
		ExpectedJobChanges:     config.Expectations.ExpectedJobChanges,
		ExpectedIssueTypes:     config.Expectations.ExpectedIssueTypes,
		ExpectedIssuePatterns:  config.Expectations.ExpectedIssuePatterns,
		MinimumJobsAnalyzed:    config.Expectations.MinimumJobsAnalyzed,
		ExpectedIncludes:       config.Expectations.ExpectedIncludes,
	}
}

// GetDefaultExpectations returns default expectations for a scenario
func GetDefaultExpectations(scenarioName string) RefactoringExpectations {
	return RefactoringExpectations{
		ShouldSucceed:          true,
		ExpectedIssueReduction: 1,
		MaxAllowedNewIssues:    0,
		SemanticEquivalence:    true,
		PerformanceImprovement: false,
		ExpectedJobChanges:     make(map[string]JobChangeType),
	}
}

// GenerateDescription generates a description for a scenario
func GenerateDescription(scenarioName string) string {
	descriptions := map[string]string{
		"scenario-1": "Duplicate script blocks consolidation",
		"scenario-2": "Complex include consolidation",
		"scenario-3": "Variable and cache optimization",
		"scenario-4": "Job dependency simplification",
		"scenario-5": "Template extraction and reuse",
		"scenario-6": "Monolithic include breakdown for microservices",
		"scenario-7": "Multi-environment matrix expansion optimization",
		"scenario-8": "Nested template inheritance consolidation",
		"scenario-9": "Cross-repository include optimization",
	}

	if desc, exists := descriptions[scenarioName]; exists {
		return desc
	}
	return fmt.Sprintf("Refactoring scenario %s", scenarioName)
}

// GenerateRealisticDescription generates a description for a realistic scenario
func GenerateRealisticDescription(scenarioName string) string {
	descriptions := map[string]string{
		"flask-microservice": "Realistic Flask microservice CI/CD pipeline optimization",
		"react-frontend":     "Frontend application pipeline with build, test, and deployment stages",
		"go-api":             "Go API service with comprehensive testing and deployment",
		"python-backend":     "Python backend service with database migrations and testing",
	}

	if desc, exists := descriptions[scenarioName]; exists {
		return desc
	}
	return fmt.Sprintf("Realistic application scenario: %s", scenarioName)
}

// GetRealisticExpectations returns expectations for realistic scenarios
func GetRealisticExpectations(scenarioName string) RefactoringExpectations {
	return RefactoringExpectations{
		ShouldSucceed:          true,
		ExpectedIssueReduction: 3,
		MaxAllowedNewIssues:    2,
		SemanticEquivalence:    false,
		PerformanceImprovement: true,
		ExpectedJobChanges:     make(map[string]JobChangeType),
		MinimumJobsAnalyzed:    5,
	}
}

// dirExists checks if a directory exists
func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// FileExists checks if a file exists
func FileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
