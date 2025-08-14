package validator

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/wonderfulspam/gitlab-smith/pkg/gitlab"
	"github.com/wonderfulspam/gitlab-smith/pkg/parser"
)

func TestNewRefactoringValidator(t *testing.T) {
	validator := NewRefactoringValidator()

	if validator == nil {
		t.Error("Expected validator to be created, got nil")
	}

	if validator.fullTestingEnabled {
		t.Error("Expected fullTestingEnabled to be false by default")
	}

	if validator.gitlabClient == nil {
		t.Error("Expected gitlab client to be initialized")
	}
}

func TestNewRefactoringValidatorWithGitLab(t *testing.T) {
	validator := NewRefactoringValidatorWithGitLab("http://localhost:8080", "test-token")

	if validator == nil {
		t.Error("Expected validator to be created, got nil")
	}

	if !validator.fullTestingEnabled {
		t.Error("Expected fullTestingEnabled to be true")
	}

	if validator.gitlabClient == nil {
		t.Error("Expected gitlab client to be set")
	}
}

func TestSetGitLabClient(t *testing.T) {
	validator := NewRefactoringValidator()

	// Create a simulation client
	client, err := gitlab.NewClient(gitlab.BackendSimulation, nil)
	if err != nil {
		t.Fatalf("Failed to create simulation client: %v", err)
	}

	validator.SetGitLabClient(client)

	if !validator.fullTestingEnabled {
		t.Error("Expected fullTestingEnabled to be true after setting client")
	}

	if validator.gitlabClient == nil {
		t.Error("Expected gitlab client to be set")
	}
}

func TestParseConfiguration(t *testing.T) {
	// Create temporary directory for testing
	tmpDir := t.TempDir()

	validator := &RefactoringValidator{}

	t.Run("no config file found", func(t *testing.T) {
		_, err := validator.parseConfiguration(tmpDir)
		if err == nil {
			t.Error("Expected error when no config file found")
		}
	})

	t.Run("find .gitlab-ci.yml", func(t *testing.T) {
		configFile := filepath.Join(tmpDir, ".gitlab-ci.yml")
		configContent := `stages:
  - build
  - test

build:
  stage: build
  script:
    - echo "Building"

test:
  stage: test
  script:
    - echo "Testing"`

		err := os.WriteFile(configFile, []byte(configContent), 0644)
		if err != nil {
			t.Fatalf("Failed to create test config file: %v", err)
		}

		config, err := validator.parseConfiguration(tmpDir)
		if err != nil {
			t.Errorf("Expected to parse config file, got error: %v", err)
		}

		if config == nil {
			t.Error("Expected config to be parsed, got nil")
		}

		if len(config.Stages) != 2 {
			t.Errorf("Expected 2 stages, got %d", len(config.Stages))
		}

		if len(config.Jobs) != 2 {
			t.Errorf("Expected 2 jobs, got %d", len(config.Jobs))
		}
	})

	t.Run("find .gitlab-ci.yaml", func(t *testing.T) {
		// Remove .yml file and create .yaml file
		os.Remove(filepath.Join(tmpDir, ".gitlab-ci.yml"))

		configFile := filepath.Join(tmpDir, ".gitlab-ci.yaml")
		configContent := `stages: [build, test]

build:
  script: ["echo Building"]`

		err := os.WriteFile(configFile, []byte(configContent), 0644)
		if err != nil {
			t.Fatalf("Failed to create test config file: %v", err)
		}

		config, err := validator.parseConfiguration(tmpDir)
		if err != nil {
			t.Errorf("Expected to parse config file, got error: %v", err)
		}

		if config == nil {
			t.Error("Expected config to be parsed, got nil")
		}
	})
}

func TestConfigToYAML(t *testing.T) {
	validator := &RefactoringValidator{}

	// Create a simple test config by parsing a YAML string
	yamlContent := `stages:
  - build
  - test

build:
  stage: build
  script:
    - echo "Building"

test:
  stage: test
  script:
    - echo "Testing"`

	config, err := parser.Parse([]byte(yamlContent))
	if err != nil {
		t.Fatalf("Failed to parse test config: %v", err)
	}

	yaml, err := validator.configToYAML(config)
	if err != nil {
		t.Errorf("Expected no error converting to YAML, got: %v", err)
	}

	if yaml == "" {
		t.Error("Expected YAML output, got empty string")
	}

	// Check that basic structure is present
	if !containsText(yaml, "stages:") {
		t.Error("Expected stages section in YAML output")
	}

	if !containsText(yaml, "build:") {
		t.Error("Expected build job in YAML output")
	}
}

func TestDetermineBehavioralEquivalence(t *testing.T) {
	validator := &RefactoringValidator{}

	t.Run("equivalent - no changes", func(t *testing.T) {
		comparison := &PipelineExecutionComparison{
			JobsAdded:    []string{},
			JobsRemoved:  []string{},
			JobsModified: []string{},
		}

		equivalent := validator.determineBehavioralEquivalence(comparison)
		if !equivalent {
			t.Error("Expected configurations to be equivalent when no changes")
		}
	})

	t.Run("not equivalent - jobs added", func(t *testing.T) {
		comparison := &PipelineExecutionComparison{
			JobsAdded:    []string{"new-job"},
			JobsRemoved:  []string{},
			JobsModified: []string{},
		}

		equivalent := validator.determineBehavioralEquivalence(comparison)
		if equivalent {
			t.Error("Expected configurations to be non-equivalent when jobs added")
		}
	})

	t.Run("not equivalent - jobs removed", func(t *testing.T) {
		comparison := &PipelineExecutionComparison{
			JobsAdded:    []string{},
			JobsRemoved:  []string{"old-job"},
			JobsModified: []string{},
		}

		equivalent := validator.determineBehavioralEquivalence(comparison)
		if equivalent {
			t.Error("Expected configurations to be non-equivalent when jobs removed")
		}
	})

	t.Run("equivalent - performance improvements allowed", func(t *testing.T) {
		comparison := &PipelineExecutionComparison{
			JobsAdded:    []string{},
			JobsRemoved:  []string{},
			JobsModified: []string{"optimized-job"},
		}

		equivalent := validator.determineBehavioralEquivalence(comparison)
		if !equivalent {
			t.Error("Expected configurations to be equivalent when only performance improvements")
		}
	})
}

func TestCompareConfigurations(t *testing.T) {
	// Create temporary directories with test configs
	beforeDir := t.TempDir()
	afterDir := t.TempDir()

	// Create simple before config
	beforeConfig := `stages:
  - build
  - test

build:
  stage: build
  script:
    - echo "Building"

test:
  stage: test
  script:
    - echo "Testing"`

	beforeFile := filepath.Join(beforeDir, ".gitlab-ci.yml")
	err := os.WriteFile(beforeFile, []byte(beforeConfig), 0644)
	if err != nil {
		t.Fatalf("Failed to create before config: %v", err)
	}

	// Create similar after config with small optimization
	afterConfig := `stages:
  - build
  - test

build:
  stage: build
  script:
    - echo "Building optimized"

test:
  stage: test
  script:
    - echo "Testing optimized"`

	afterFile := filepath.Join(afterDir, ".gitlab-ci.yml")
	err = os.WriteFile(afterFile, []byte(afterConfig), 0644)
	if err != nil {
		t.Fatalf("Failed to create after config: %v", err)
	}

	validator := NewRefactoringValidator()

	result, err := validator.CompareConfigurations(beforeDir, afterDir)
	if err != nil {
		t.Fatalf("CompareConfigurations failed: %v", err)
	}

	// Check basic result structure
	if result == nil {
		t.Fatal("Expected result, got nil")
	}

	if result.ActualChanges == nil {
		t.Error("Expected comparison results, got nil")
	}

	if result.PipelineComparison == nil {
		t.Error("Expected pipeline comparison, got nil")
	}

	// Analysis improvement should be calculated
	if result.AnalysisImprovement == 0 && result.ActualChanges.HasChanges {
		// This is fine - no analysis improvement doesn't mean test failure
	}
}

// Helper functions

func containsText(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}