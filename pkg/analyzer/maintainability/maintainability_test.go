package maintainability

import (
	"testing"

	"github.com/wonderfulspam/gitlab-smith/pkg/analyzer/types"
)

// Mock registry for testing
type mockRegistry struct {
	checks map[string]types.CheckFunc
}

func newMockRegistry() *mockRegistry {
	return &mockRegistry{
		checks: make(map[string]types.CheckFunc),
	}
}

func (r *mockRegistry) Register(name string, issueType types.IssueType, checkFunc types.CheckFunc) {
	r.checks[name] = checkFunc
}

func TestRegisterChecks(t *testing.T) {
	t.Run("registers all checks", func(t *testing.T) {
		registry := newMockRegistry()

		RegisterChecks(registry)

		// Verify that checks are registered by checking count
		if len(registry.checks) == 0 {
			t.Error("Expected at least one maintainability check to be registered")
		}

		// Check for specific known check names
		expectedChecks := []string{
			"job_naming",
			"script_complexity",
			"verbose_rules",
			"duplicated_code",
			"duplicated_before_scripts",
			"duplicated_cache_config",
			"duplicated_image_config",
			"duplicated_setup",
			"stages_definition",
			"include_optimization",
		}

		for _, expectedName := range expectedChecks {
			if registry.checks[expectedName] == nil {
				t.Errorf("Expected check '%s' to be registered", expectedName)
			}
		}

		// Verify we have exactly the expected number of checks
		if len(registry.checks) != len(expectedChecks) {
			t.Errorf("Expected %d checks to be registered, got %d", len(expectedChecks), len(registry.checks))
		}
	})

	t.Run("all registered functions are not nil", func(t *testing.T) {
		registry := newMockRegistry()

		RegisterChecks(registry)

		for name, checkFunc := range registry.checks {
			if checkFunc == nil {
				t.Errorf("Check function for '%s' should not be nil", name)
			}
		}
	})
}
