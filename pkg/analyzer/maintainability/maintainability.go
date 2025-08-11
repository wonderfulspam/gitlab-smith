package maintainability

import (
	"github.com/wonderfulspam/gitlab-smith/pkg/analyzer/types"
)

// CheckRegistry interface to avoid import cycles
type CheckRegistry interface {
	Register(name string, issueType types.IssueType, checkFunc types.CheckFunc)
}

// RegisterChecks registers all maintainability-related checks
func RegisterChecks(registry CheckRegistry) {
	// Naming checks
	registry.Register("job_naming", types.IssueTypeMaintainability, CheckJobNaming)

	// Complexity checks
	registry.Register("script_complexity", types.IssueTypeMaintainability, CheckScriptComplexity)
	registry.Register("verbose_rules", types.IssueTypeMaintainability, CheckVerboseRules)

	// Duplication checks
	registry.Register("duplicated_code", types.IssueTypeMaintainability, CheckDuplicatedCode)
	registry.Register("duplicated_before_scripts", types.IssueTypeMaintainability, CheckDuplicatedBeforeScripts)
	registry.Register("duplicated_cache_config", types.IssueTypeMaintainability, CheckDuplicatedCacheConfig)
	registry.Register("duplicated_image_config", types.IssueTypeMaintainability, CheckDuplicatedImageConfig)
	registry.Register("duplicated_setup", types.IssueTypeMaintainability, CheckDuplicatedSetup)

	// Structure checks
	registry.Register("stages_definition", types.IssueTypeMaintainability, CheckStagesDefinition)
	registry.Register("include_optimization", types.IssueTypeMaintainability, CheckIncludeOptimization)
}
