package differ

import (
	"testing"

	"github.com/wonderfulspam/gitlab-smith/pkg/parser"
)

func TestCompare_NoChanges(t *testing.T) {
	config := &parser.GitLabConfig{
		Stages: []string{"build", "test", "deploy"},
		Jobs: map[string]*parser.JobConfig{
			"test": {
				Stage:  "test",
				Script: []string{"npm test"},
				Image:  "node:16",
			},
		},
		Variables: map[string]interface{}{
			"NODE_VERSION": "16",
		},
	}

	result := Compare(config, config)

	if result.HasChanges {
		t.Error("Expected no changes, but HasChanges is true")
	}

	if len(result.Semantic) != 0 {
		t.Errorf("Expected 0 semantic changes, got %d", len(result.Semantic))
	}

	if len(result.Dependencies) != 0 {
		t.Errorf("Expected 0 dependency changes, got %d", len(result.Dependencies))
	}

	if len(result.Performance) != 0 {
		t.Errorf("Expected 0 performance changes, got %d", len(result.Performance))
	}

	if result.Summary != "No semantic differences found" {
		t.Errorf("Expected 'No semantic differences found', got '%s'", result.Summary)
	}
}

func TestCompare_StagesChanged(t *testing.T) {
	oldConfig := &parser.GitLabConfig{
		Stages: []string{"build", "test"},
		Jobs:   make(map[string]*parser.JobConfig),
	}

	newConfig := &parser.GitLabConfig{
		Stages: []string{"build", "test", "deploy"},
		Jobs:   make(map[string]*parser.JobConfig),
	}

	result := Compare(oldConfig, newConfig)

	if !result.HasChanges {
		t.Error("Expected changes, but HasChanges is false")
	}

	if len(result.Semantic) != 1 {
		t.Errorf("Expected 1 semantic change, got %d", len(result.Semantic))
	}

	if result.Semantic[0].Type != DiffTypeModified {
		t.Errorf("Expected DiffTypeModified, got %s", result.Semantic[0].Type)
	}

	if result.Semantic[0].Path != "stages" {
		t.Errorf("Expected path 'stages', got '%s'", result.Semantic[0].Path)
	}
}

func TestCompare_JobAdded(t *testing.T) {
	oldConfig := &parser.GitLabConfig{
		Jobs: map[string]*parser.JobConfig{
			"build": {
				Stage:  "build",
				Script: []string{"make build"},
			},
		},
	}

	newConfig := &parser.GitLabConfig{
		Jobs: map[string]*parser.JobConfig{
			"build": {
				Stage:  "build",
				Script: []string{"make build"},
			},
			"test": {
				Stage:  "test",
				Script: []string{"make test"},
			},
		},
	}

	result := Compare(oldConfig, newConfig)

	if !result.HasChanges {
		t.Error("Expected changes, but HasChanges is false")
	}

	if len(result.Semantic) != 1 {
		t.Errorf("Expected 1 semantic change, got %d", len(result.Semantic))
	}

	diff := result.Semantic[0]
	if diff.Type != DiffTypeAdded {
		t.Errorf("Expected DiffTypeAdded, got %s", diff.Type)
	}

	if diff.Path != "jobs.test" {
		t.Errorf("Expected path 'jobs.test', got '%s'", diff.Path)
	}

	if diff.Description != "Job added: test" {
		t.Errorf("Expected description 'Job added: test', got '%s'", diff.Description)
	}
}

func TestCompare_JobRemoved(t *testing.T) {
	oldConfig := &parser.GitLabConfig{
		Jobs: map[string]*parser.JobConfig{
			"build": {
				Stage:  "build",
				Script: []string{"make build"},
			},
			"test": {
				Stage:  "test",
				Script: []string{"make test"},
			},
		},
	}

	newConfig := &parser.GitLabConfig{
		Jobs: map[string]*parser.JobConfig{
			"build": {
				Stage:  "build",
				Script: []string{"make build"},
			},
		},
	}

	result := Compare(oldConfig, newConfig)

	if !result.HasChanges {
		t.Error("Expected changes, but HasChanges is false")
	}

	if len(result.Semantic) != 1 {
		t.Errorf("Expected 1 semantic change, got %d", len(result.Semantic))
	}

	diff := result.Semantic[0]
	if diff.Type != DiffTypeRemoved {
		t.Errorf("Expected DiffTypeRemoved, got %s", diff.Type)
	}

	if diff.Path != "jobs.test" {
		t.Errorf("Expected path 'jobs.test', got '%s'", diff.Path)
	}
}

func TestCompare_JobScriptChanged(t *testing.T) {
	oldConfig := &parser.GitLabConfig{
		Jobs: map[string]*parser.JobConfig{
			"test": {
				Stage:  "test",
				Script: []string{"npm test"},
			},
		},
	}

	newConfig := &parser.GitLabConfig{
		Jobs: map[string]*parser.JobConfig{
			"test": {
				Stage:  "test",
				Script: []string{"npm test", "npm run coverage"},
			},
		},
	}

	result := Compare(oldConfig, newConfig)

	if !result.HasChanges {
		t.Error("Expected changes, but HasChanges is false")
	}

	if len(result.Semantic) != 1 {
		t.Errorf("Expected 1 semantic change, got %d", len(result.Semantic))
	}

	diff := result.Semantic[0]
	if diff.Type != DiffTypeModified {
		t.Errorf("Expected DiffTypeModified, got %s", diff.Type)
	}

	if diff.Path != "jobs.test.script" {
		t.Errorf("Expected path 'jobs.test.script', got '%s'", diff.Path)
	}
}

func TestCompare_ImageChanged_PerformanceCategory(t *testing.T) {
	oldConfig := &parser.GitLabConfig{
		Jobs: map[string]*parser.JobConfig{
			"test": {
				Stage:  "test",
				Script: []string{"npm test"},
				Image:  "node:16",
			},
		},
	}

	newConfig := &parser.GitLabConfig{
		Jobs: map[string]*parser.JobConfig{
			"test": {
				Stage:  "test",
				Script: []string{"npm test"},
				Image:  "node:18",
			},
		},
	}

	result := Compare(oldConfig, newConfig)

	if !result.HasChanges {
		t.Error("Expected changes, but HasChanges is false")
	}

	if len(result.Performance) != 1 {
		t.Errorf("Expected 1 performance change, got %d", len(result.Performance))
	}

	if len(result.Semantic) != 0 {
		t.Errorf("Expected 0 semantic changes, got %d", len(result.Semantic))
	}

	diff := result.Performance[0]
	if diff.Type != DiffTypeModified {
		t.Errorf("Expected DiffTypeModified, got %s", diff.Type)
	}

	if diff.Path != "jobs.test.image" {
		t.Errorf("Expected path 'jobs.test.image', got '%s'", diff.Path)
	}
}

func TestCompare_DependenciesChanged(t *testing.T) {
	oldConfig := &parser.GitLabConfig{
		Jobs: map[string]*parser.JobConfig{
			"test": {
				Stage:        "test",
				Script:       []string{"npm test"},
				Dependencies: []string{"build"},
			},
		},
	}

	newConfig := &parser.GitLabConfig{
		Jobs: map[string]*parser.JobConfig{
			"test": {
				Stage:        "test",
				Script:       []string{"npm test"},
				Dependencies: []string{"build", "lint"},
			},
		},
	}

	result := Compare(oldConfig, newConfig)

	if !result.HasChanges {
		t.Error("Expected changes, but HasChanges is false")
	}

	if len(result.Dependencies) != 2 {
		t.Errorf("Expected 2 dependency changes, got %d", len(result.Dependencies))
	}

	// Should detect both job-level and dependency graph changes
	foundJobLevel := false
	foundGraphLevel := false

	for _, diff := range result.Dependencies {
		if diff.Type != DiffTypeModified {
			t.Errorf("Expected DiffTypeModified, got %s", diff.Type)
		}

		if diff.Path == "jobs.test.dependencies" {
			foundJobLevel = true
		} else if diff.Path == "dependency_graph.test" {
			foundGraphLevel = true
		}
	}

	if !foundJobLevel {
		t.Error("Expected to find job-level dependency change")
	}

	if !foundGraphLevel {
		t.Error("Expected to find dependency graph change")
	}
}

func TestCompare_VariablesAdded(t *testing.T) {
	oldConfig := &parser.GitLabConfig{
		Variables: map[string]interface{}{
			"NODE_VERSION": "16",
		},
		Jobs: make(map[string]*parser.JobConfig),
	}

	newConfig := &parser.GitLabConfig{
		Variables: map[string]interface{}{
			"NODE_VERSION": "16",
			"BUILD_ENV":    "production",
		},
		Jobs: make(map[string]*parser.JobConfig),
	}

	result := Compare(oldConfig, newConfig)

	if !result.HasChanges {
		t.Error("Expected changes, but HasChanges is false")
	}

	if len(result.Semantic) != 1 {
		t.Errorf("Expected 1 semantic change, got %d", len(result.Semantic))
	}

	diff := result.Semantic[0]
	if diff.Type != DiffTypeAdded {
		t.Errorf("Expected DiffTypeAdded, got %s", diff.Type)
	}

	if diff.Path != "variables.BUILD_ENV" {
		t.Errorf("Expected path 'variables.BUILD_ENV', got '%s'", diff.Path)
	}
}

func TestCompare_VariablesRemoved(t *testing.T) {
	oldConfig := &parser.GitLabConfig{
		Variables: map[string]interface{}{
			"NODE_VERSION": "16",
			"BUILD_ENV":    "production",
		},
		Jobs: make(map[string]*parser.JobConfig),
	}

	newConfig := &parser.GitLabConfig{
		Variables: map[string]interface{}{
			"NODE_VERSION": "16",
		},
		Jobs: make(map[string]*parser.JobConfig),
	}

	result := Compare(oldConfig, newConfig)

	if !result.HasChanges {
		t.Error("Expected changes, but HasChanges is false")
	}

	if len(result.Semantic) != 1 {
		t.Errorf("Expected 1 semantic change, got %d", len(result.Semantic))
	}

	diff := result.Semantic[0]
	if diff.Type != DiffTypeRemoved {
		t.Errorf("Expected DiffTypeRemoved, got %s", diff.Type)
	}

	if diff.Path != "variables.BUILD_ENV" {
		t.Errorf("Expected path 'variables.BUILD_ENV', got '%s'", diff.Path)
	}
}

func TestCompare_VariablesModified(t *testing.T) {
	oldConfig := &parser.GitLabConfig{
		Variables: map[string]interface{}{
			"NODE_VERSION": "16",
		},
		Jobs: make(map[string]*parser.JobConfig),
	}

	newConfig := &parser.GitLabConfig{
		Variables: map[string]interface{}{
			"NODE_VERSION": "18",
		},
		Jobs: make(map[string]*parser.JobConfig),
	}

	result := Compare(oldConfig, newConfig)

	if !result.HasChanges {
		t.Error("Expected changes, but HasChanges is false")
	}

	if len(result.Semantic) != 1 {
		t.Errorf("Expected 1 semantic change, got %d", len(result.Semantic))
	}

	diff := result.Semantic[0]
	if diff.Type != DiffTypeModified {
		t.Errorf("Expected DiffTypeModified, got %s", diff.Type)
	}

	if diff.Path != "variables.NODE_VERSION" {
		t.Errorf("Expected path 'variables.NODE_VERSION', got '%s'", diff.Path)
	}
}

func TestCompare_CacheChanged_PerformanceCategory(t *testing.T) {
	oldConfig := &parser.GitLabConfig{
		Jobs: map[string]*parser.JobConfig{
			"test": {
				Stage:  "test",
				Script: []string{"npm test"},
				Cache: &parser.Cache{
					Key:   "test-cache",
					Paths: []string{"node_modules/"},
				},
			},
		},
	}

	newConfig := &parser.GitLabConfig{
		Jobs: map[string]*parser.JobConfig{
			"test": {
				Stage:  "test",
				Script: []string{"npm test"},
				Cache: &parser.Cache{
					Key:   "test-cache-v2",
					Paths: []string{"node_modules/", ".npm/"},
				},
			},
		},
	}

	result := Compare(oldConfig, newConfig)

	if !result.HasChanges {
		t.Error("Expected changes, but HasChanges is false")
	}

	if len(result.Performance) != 1 {
		t.Errorf("Expected 1 performance change, got %d", len(result.Performance))
	}

	diff := result.Performance[0]
	if diff.Type != DiffTypeModified {
		t.Errorf("Expected DiffTypeModified, got %s", diff.Type)
	}

	if diff.Path != "jobs.test.cache" {
		t.Errorf("Expected path 'jobs.test.cache', got '%s'", diff.Path)
	}
}

func TestCompare_ComplexChanges(t *testing.T) {
	oldConfig := &parser.GitLabConfig{
		Stages: []string{"build", "test"},
		Jobs: map[string]*parser.JobConfig{
			"build": {
				Stage:  "build",
				Script: []string{"make build"},
				Image:  "golang:1.24",
			},
			"test": {
				Stage:        "test",
				Script:       []string{"make test"},
				Dependencies: []string{"build"},
			},
		},
		Variables: map[string]interface{}{
			"GO_VERSION": "1.24",
		},
	}

	newConfig := &parser.GitLabConfig{
		Stages: []string{"build", "test", "deploy"},
		Jobs: map[string]*parser.JobConfig{
			"build": {
				Stage:  "build",
				Script: []string{"make build", "make package"},
				Image:  "golang:1.20",
			},
			"test": {
				Stage:        "test",
				Script:       []string{"make test"},
				Dependencies: []string{"build"},
			},
			"deploy": {
				Stage:  "deploy",
				Script: []string{"make deploy"},
			},
		},
		Variables: map[string]interface{}{
			"GO_VERSION": "1.20",
			"DEPLOY_ENV": "staging",
		},
	}

	result := Compare(oldConfig, newConfig)

	if !result.HasChanges {
		t.Error("Expected changes, but HasChanges is false")
	}

	// Should have multiple categories of changes
	if len(result.Semantic) == 0 {
		t.Error("Expected semantic changes")
	}

	if len(result.Performance) == 0 {
		t.Error("Expected performance changes")
	}

	// Check that summary includes multiple change types
	if !contains(result.Summary, "semantic changes") {
		t.Errorf("Summary should mention semantic changes: %s", result.Summary)
	}

	if !contains(result.Summary, "performance-related changes") {
		t.Errorf("Summary should mention performance changes: %s", result.Summary)
	}
}
