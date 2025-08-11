package analyzer

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wonderfulspam/gitlab-smith/pkg/analyzer/types"
	"gopkg.in/yaml.v2"
)

func TestDefaultConfigStructure(t *testing.T) {
	config := DefaultConfig()

	if config == nil {
		t.Fatal("DefaultConfig() returned nil")
	}
	if config.Checks == nil {
		t.Fatal("DefaultConfig().Checks is nil")
	}

	// Verify some expected checks exist
	expectedChecks := []string{
		"cache_usage",
		"image_tags",
		"job_naming",
		"missing_stages",
	}

	for _, checkName := range expectedChecks {
		if _, exists := config.Checks[checkName]; !exists {
			t.Errorf("Expected check '%s' not found in default config", checkName)
		}
	}

	// Verify all checks are enabled by default
	for checkName, check := range config.Checks {
		if !check.Enabled {
			t.Errorf("Check '%s' should be enabled by default", checkName)
		}
		if check.Name != checkName {
			t.Errorf("Check name mismatch: map key '%s' vs check.Name '%s'", checkName, check.Name)
		}
		if check.Description == "" {
			t.Errorf("Check '%s' has empty description", checkName)
		}
	}

	// Verify we have checks of all types
	typeCount := make(map[types.IssueType]int)
	for _, check := range config.Checks {
		typeCount[check.Type]++
	}

	if typeCount[types.IssueTypePerformance] == 0 {
		t.Error("No performance checks found in default config")
	}
	if typeCount[types.IssueTypeSecurity] == 0 {
		t.Error("No security checks found in default config")
	}
	if typeCount[types.IssueTypeMaintainability] == 0 {
		t.Error("No maintainability checks found in default config")
	}
	if typeCount[types.IssueTypeReliability] == 0 {
		t.Error("No reliability checks found in default config")
	}
}

func TestLoadConfigYAML(t *testing.T) {
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "test-config.yaml")

	// Create test YAML config
	yamlConfig := `
checks:
  test_check:
    name: test_check
    type: performance
    enabled: true
    description: Test check for YAML loading
  disabled_check:
    name: disabled_check
    type: security
    enabled: false
    description: Disabled test check
`

	err := os.WriteFile(configFile, []byte(yamlConfig), 0644)
	if err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	config, err := LoadConfig(configFile)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	// Verify loaded checks
	if len(config.Checks) < 2 {
		t.Error("Expected at least 2 checks (including defaults)")
	}

	testCheck, exists := config.Checks["test_check"]
	if !exists {
		t.Error("test_check not found in loaded config")
	} else {
		if testCheck.Name != "test_check" {
			t.Errorf("Expected name 'test_check', got '%s'", testCheck.Name)
		}
		if testCheck.Type != types.IssueTypePerformance {
			t.Errorf("Expected type Performance, got %v", testCheck.Type)
		}
		if !testCheck.Enabled {
			t.Error("test_check should be enabled")
		}
	}

	disabledCheck, exists := config.Checks["disabled_check"]
	if !exists {
		t.Error("disabled_check not found in loaded config")
	} else {
		if disabledCheck.Enabled {
			t.Error("disabled_check should be disabled")
		}
	}

	// Verify defaults are merged
	if _, exists := config.Checks["cache_usage"]; !exists {
		t.Error("Default check 'cache_usage' should be present")
	}
}

func TestLoadConfigJSON(t *testing.T) {
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "test-config.json")

	// Create test JSON config
	testConfig := Config{
		Checks: map[string]types.CheckConfig{
			"json_test_check": {
				Name:        "json_test_check",
				Type:        types.IssueTypeMaintainability,
				Enabled:     true,
				Description: "Test check for JSON loading",
			},
		},
	}

	data, err := json.MarshalIndent(testConfig, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal test config: %v", err)
	}

	err = os.WriteFile(configFile, data, 0644)
	if err != nil {
		t.Fatalf("Failed to write test config file: %v", err)
	}

	config, err := LoadConfig(configFile)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	// Verify loaded check
	jsonCheck, exists := config.Checks["json_test_check"]
	if !exists {
		t.Error("json_test_check not found in loaded config")
	} else {
		if jsonCheck.Type != types.IssueTypeMaintainability {
			t.Errorf("Expected type Maintainability, got %v", jsonCheck.Type)
		}
	}

	// Verify defaults are merged
	if _, exists := config.Checks["cache_usage"]; !exists {
		t.Error("Default check 'cache_usage' should be present")
	}
}

func TestLoadConfigNonExistentFile(t *testing.T) {
	_, err := LoadConfig("/non/existent/config.yaml")
	if err == nil {
		t.Error("Expected error when loading non-existent file")
	}
	if !strings.Contains(err.Error(), "failed to read config file") {
		t.Errorf("Expected 'failed to read config file' error, got: %v", err)
	}
}

func TestLoadConfigInvalidFormat(t *testing.T) {
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "invalid-config.yaml")

	// Create invalid config
	invalidConfig := `invalid: yaml: content: [unclosed`
	err := os.WriteFile(configFile, []byte(invalidConfig), 0644)
	if err != nil {
		t.Fatalf("Failed to create invalid config file: %v", err)
	}

	_, err = LoadConfig(configFile)
	if err == nil {
		t.Error("Expected error when loading invalid config")
	}
	if !strings.Contains(err.Error(), "failed to parse config file") {
		t.Errorf("Expected parsing error, got: %v", err)
	}
}

func TestSaveConfigYAML(t *testing.T) {
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "save-test.yaml")

	config := &Config{
		Checks: map[string]types.CheckConfig{
			"save_test_check": {
				Name:        "save_test_check",
				Type:        types.IssueTypePerformance,
				Enabled:     true,
				Description: "Test check for saving",
			},
		},
	}

	err := SaveConfig(config, configFile)
	if err != nil {
		t.Fatalf("SaveConfig failed: %v", err)
	}

	// Verify file was created and is valid YAML
	data, err := os.ReadFile(configFile)
	if err != nil {
		t.Fatalf("Failed to read saved config file: %v", err)
	}

	var loadedConfig Config
	err = yaml.Unmarshal(data, &loadedConfig)
	if err != nil {
		t.Fatalf("Failed to unmarshal saved YAML: %v", err)
	}

	if _, exists := loadedConfig.Checks["save_test_check"]; !exists {
		t.Error("Saved check not found in loaded config")
	}
}

func TestSaveConfigJSON(t *testing.T) {
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "save-test.json")

	config := &Config{
		Checks: map[string]types.CheckConfig{
			"save_test_check": {
				Name:        "save_test_check",
				Type:        types.IssueTypeSecurity,
				Enabled:     false,
				Description: "Test check for JSON saving",
			},
		},
	}

	err := SaveConfig(config, configFile)
	if err != nil {
		t.Fatalf("SaveConfig failed: %v", err)
	}

	// Verify file was created and is valid JSON
	data, err := os.ReadFile(configFile)
	if err != nil {
		t.Fatalf("Failed to read saved config file: %v", err)
	}

	var loadedConfig Config
	err = json.Unmarshal(data, &loadedConfig)
	if err != nil {
		t.Fatalf("Failed to unmarshal saved JSON: %v", err)
	}

	savedCheck, exists := loadedConfig.Checks["save_test_check"]
	if !exists {
		t.Error("Saved check not found in loaded config")
	} else {
		if savedCheck.Enabled {
			t.Error("Saved check should be disabled")
		}
		if savedCheck.Type != types.IssueTypeSecurity {
			t.Errorf("Expected Security type, got %v", savedCheck.Type)
		}
	}
}

func TestLoadOrCreateConfig(t *testing.T) {
	tempDir := t.TempDir()

	t.Run("create new config", func(t *testing.T) {
		configFile := filepath.Join(tempDir, "new-config.yaml")

		config, err := LoadOrCreateConfig(configFile)
		if err != nil {
			t.Fatalf("LoadOrCreateConfig failed: %v", err)
		}

		// Should return default config
		if config == nil {
			t.Fatal("Expected config, got nil")
		}
		if len(config.Checks) == 0 {
			t.Error("Expected default checks")
		}

		// Verify file was created
		if _, err := os.Stat(configFile); os.IsNotExist(err) {
			t.Error("Config file should have been created")
		}
	})

	t.Run("load existing config", func(t *testing.T) {
		configFile := filepath.Join(tempDir, "existing-config.yaml")

		// Create existing config
		existingConfig := `
checks:
  existing_check:
    name: existing_check
    type: performance
    enabled: true
    description: Existing check
`
		err := os.WriteFile(configFile, []byte(existingConfig), 0644)
		if err != nil {
			t.Fatalf("Failed to create existing config: %v", err)
		}

		config, err := LoadOrCreateConfig(configFile)
		if err != nil {
			t.Fatalf("LoadOrCreateConfig failed: %v", err)
		}

		// Should load existing config
		if _, exists := config.Checks["existing_check"]; !exists {
			t.Error("Existing check not found")
		}
	})
}

func TestConfigIsCheckEnabled(t *testing.T) {
	config := &Config{
		Checks: map[string]types.CheckConfig{
			"enabled_check": {
				Name:    "enabled_check",
				Enabled: true,
			},
			"disabled_check": {
				Name:    "disabled_check",
				Enabled: false,
			},
		},
	}

	if !config.IsCheckEnabled("enabled_check") {
		t.Error("enabled_check should be enabled")
	}
	if config.IsCheckEnabled("disabled_check") {
		t.Error("disabled_check should be disabled")
	}
	if config.IsCheckEnabled("nonexistent_check") {
		t.Error("nonexistent_check should return false")
	}
}

func TestConfigEnableDisableCheck(t *testing.T) {
	config := &Config{
		Checks: map[string]types.CheckConfig{
			"test_check": {
				Name:        "test_check",
				Type:        types.IssueTypePerformance,
				Enabled:     false,
				Description: "Test check",
			},
		},
	}

	// Enable check
	config.EnableCheck("test_check")
	if !config.IsCheckEnabled("test_check") {
		t.Error("test_check should be enabled after EnableCheck")
	}

	// Disable check
	config.DisableCheck("test_check")
	if config.IsCheckEnabled("test_check") {
		t.Error("test_check should be disabled after DisableCheck")
	}

	// Test with nonexistent check (should not panic)
	config.EnableCheck("nonexistent_check")
	config.DisableCheck("nonexistent_check")
}

func TestConfigGetEnabledChecks(t *testing.T) {
	config := &Config{
		Checks: map[string]types.CheckConfig{
			"enabled1": {Name: "enabled1", Enabled: true},
			"enabled2": {Name: "enabled2", Enabled: true},
			"disabled": {Name: "disabled", Enabled: false},
		},
	}

	enabled := config.GetEnabledChecks()
	if len(enabled) != 2 {
		t.Errorf("Expected 2 enabled checks, got %d", len(enabled))
	}

	// Convert to map for easier checking (order not guaranteed)
	enabledMap := make(map[string]bool)
	for _, checkName := range enabled {
		enabledMap[checkName] = true
	}

	if !enabledMap["enabled1"] {
		t.Error("enabled1 should be in enabled checks list")
	}
	if !enabledMap["enabled2"] {
		t.Error("enabled2 should be in enabled checks list")
	}
	if enabledMap["disabled"] {
		t.Error("disabled should not be in enabled checks list")
	}
}

func TestConfigGetChecksByType(t *testing.T) {
	config := &Config{
		Checks: map[string]types.CheckConfig{
			"perf1":     {Name: "perf1", Type: types.IssueTypePerformance, Enabled: true},
			"perf2":     {Name: "perf2", Type: types.IssueTypePerformance, Enabled: true},
			"security1": {Name: "security1", Type: types.IssueTypeSecurity, Enabled: true},
			"disabled":  {Name: "disabled", Type: types.IssueTypePerformance, Enabled: false},
		},
	}

	perfChecks := config.GetChecksByType(types.IssueTypePerformance)
	if len(perfChecks) != 2 {
		t.Errorf("Expected 2 performance checks, got %d", len(perfChecks))
	}

	secChecks := config.GetChecksByType(types.IssueTypeSecurity)
	if len(secChecks) != 1 {
		t.Errorf("Expected 1 security check, got %d", len(secChecks))
	}

	reliabilityChecks := config.GetChecksByType(types.IssueTypeReliability)
	if len(reliabilityChecks) != 0 {
		t.Errorf("Expected 0 reliability checks, got %d", len(reliabilityChecks))
	}
}
