package renderer

import (
	"testing"

	"github.com/wonderfulspam/gitlab-smith/pkg/parser"
)

func TestRenderer_SimulatePipelineExecution(t *testing.T) {
	renderer := New(nil)

	config := &parser.GitLabConfig{
		Stages: []string{"build", "test", "deploy"},
		Variables: map[string]interface{}{
			"NODE_ENV": "production",
		},
		Jobs: map[string]*parser.JobConfig{
			"build": {
				Stage:  "build",
				Script: []string{"npm run build"},
			},
			"test": {
				Stage:        "test",
				Script:       []string{"npm test"},
				Dependencies: []string{"build"},
			},
			"deploy": {
				Stage:  "deploy",
				Script: []string{"npm run deploy"},
				Needs:  []interface{}{map[string]interface{}{"job": "test"}},
			},
		},
	}

	execution := renderer.simulatePipelineExecution(config)

	if execution == nil {
		t.Fatal("Expected non-nil pipeline execution")
	}

	if execution.Status != "simulated" {
		t.Errorf("Expected status 'simulated', got %s", execution.Status)
	}

	if len(execution.Jobs) != 3 {
		t.Errorf("Expected 3 jobs, got %d", len(execution.Jobs))
	}

	// Verify jobs are ordered by stage
	expectedOrder := []string{"build", "test", "deploy"}
	for i, expectedStage := range expectedOrder {
		if execution.Jobs[i].Stage != expectedStage {
			t.Errorf("Expected job %d to be in stage %s, got %s", i, expectedStage, execution.Jobs[i].Stage)
		}
	}

	// Verify variables are copied
	if execution.Variables["NODE_ENV"] != "production" {
		t.Errorf("Expected NODE_ENV=production, got %s", execution.Variables["NODE_ENV"])
	}
}

func TestHelperFunctions(t *testing.T) {
	// Test extractJobNames
	needs := []interface{}{
		map[string]interface{}{"job": "build"},
		map[string]interface{}{"job": "test"},
	}
	names := extractJobNames(needs)
	expected := []string{"build", "test"}

	if len(names) != len(expected) {
		t.Errorf("Expected %d names, got %d", len(expected), len(names))
	}

	for i, name := range names {
		if name != expected[i] {
			t.Errorf("Expected name %s at index %d, got %s", expected[i], i, name)
		}
	}

	// Test estimateJobDuration
	job := &parser.JobConfig{
		Script:   []string{"echo hello", "npm test", "npm build"},
		Services: []string{"postgres:13"},
	}
	duration := estimateJobDuration(job)
	expectedDuration := 30.0 + (3.0 * 2.0) + 15.0 // base + scripts + services = 51.0

	if duration != expectedDuration {
		t.Errorf("Expected duration %f, got %f", expectedDuration, duration)
	}

	// Test getStageOrder
	stages := []string{"build", "test", "deploy"}
	order := getStageOrder("test", stages)
	if order != 1 {
		t.Errorf("Expected order 1 for 'test' stage, got %d", order)
	}

	unknownOrder := getStageOrder("unknown", stages)
	if unknownOrder != 999 {
		t.Errorf("Expected order 999 for unknown stage, got %d", unknownOrder)
	}

	// Test equalStringSlices
	slice1 := []string{"a", "b", "c"}
	slice2 := []string{"c", "a", "b"}
	slice3 := []string{"a", "b"}

	if !equalStringSlices(slice1, slice2) {
		t.Error("Expected slices with same elements to be equal")
	}

	if equalStringSlices(slice1, slice3) {
		t.Error("Expected slices with different lengths to be unequal")
	}
}

// Helper function for floating point comparison
func abs(f float64) float64 {
	if f < 0 {
		return -f
	}
	return f
}
