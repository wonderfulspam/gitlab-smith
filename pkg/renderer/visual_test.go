package renderer

import (
	"strings"
	"testing"

	"github.com/wonderfulspam/gitlab-smith/pkg/parser"
)

func TestVisualRenderer_RenderPipelineGraph_Mermaid(t *testing.T) {
	// Create a test configuration
	config := &parser.GitLabConfig{
		Stages: []string{"build", "test", "deploy"},
		Jobs: map[string]*parser.JobConfig{
			"build": {
				Stage:  "build",
				Script: []string{"echo 'building'"},
			},
			"test:unit": {
				Stage:        "test",
				Script:       []string{"echo 'unit tests'"},
				Dependencies: []string{"build"},
			},
			"test:integration": {
				Stage:        "test",
				Script:       []string{"echo 'integration tests'"},
				Dependencies: []string{"build"},
			},
			"deploy:staging": {
				Stage:        "deploy",
				Script:       []string{"echo 'deploying to staging'"},
				Dependencies: []string{"test:unit", "test:integration"},
			},
		},
	}

	vr := NewVisualRenderer()
	result, err := vr.RenderPipelineGraph(config, FormatMermaid)

	if err != nil {
		t.Fatalf("RenderPipelineGraph failed: %v", err)
	}

	// Check that it starts with flowchart directive
	if !strings.HasPrefix(result, "flowchart TD") {
		t.Errorf("Expected Mermaid flowchart to start with 'flowchart TD', got: %s", result[:20])
	}

	// Check that all stages are represented as subgraphs
	expectedStages := []string{"build", "test", "deploy"}
	for _, stage := range expectedStages {
		if !strings.Contains(result, stage) {
			t.Errorf("Expected to find stage '%s' in Mermaid output", stage)
		}
	}

	// Check that all jobs are present
	expectedJobs := []string{"build", "test_unit", "test_integration", "deploy_staging"}
	for _, job := range expectedJobs {
		if !strings.Contains(result, job) {
			t.Errorf("Expected to find job reference '%s' in Mermaid output", job)
		}
	}

	// Check for dependency arrows
	if !strings.Contains(result, "build --> test_unit") {
		t.Error("Expected to find dependency 'build --> test_unit' in Mermaid output")
	}
	if !strings.Contains(result, "build --> test_integration") {
		t.Error("Expected to find dependency 'build --> test_integration' in Mermaid output")
	}
}

func TestVisualRenderer_RenderPipelineGraph_DOT(t *testing.T) {
	config := &parser.GitLabConfig{
		Stages: []string{"build", "test"},
		Jobs: map[string]*parser.JobConfig{
			"build": {
				Stage:  "build",
				Script: []string{"make build"},
			},
			"test": {
				Stage:        "test",
				Script:       []string{"make test"},
				Dependencies: []string{"build"},
			},
		},
	}

	vr := NewVisualRenderer()
	result, err := vr.RenderPipelineGraph(config, FormatDOT)

	if err != nil {
		t.Fatalf("RenderPipelineGraph failed: %v", err)
	}

	// Check DOT graph structure
	if !strings.HasPrefix(result, "digraph pipeline {") {
		t.Errorf("Expected DOT graph to start with 'digraph pipeline {', got: %s", result[:30])
	}

	if !strings.HasSuffix(strings.TrimSpace(result), "}") {
		t.Error("Expected DOT graph to end with '}'")
	}

	// Check for stage subgraphs
	if !strings.Contains(result, "subgraph cluster_") {
		t.Error("Expected to find stage subgraphs in DOT output")
	}

	// Check for job nodes
	if !strings.Contains(result, `"build"`) {
		t.Error("Expected to find build job node in DOT output")
	}
	if !strings.Contains(result, `"test"`) {
		t.Error("Expected to find test job node in DOT output")
	}

	// Check for dependency edge
	if !strings.Contains(result, `"build" -> "test"`) {
		t.Error("Expected to find dependency edge 'build -> test' in DOT output")
	}
}

func TestVisualRenderer_RenderComparisonGraph_Mermaid(t *testing.T) {
	oldConfig := &parser.GitLabConfig{
		Stages: []string{"build", "test"},
		Jobs: map[string]*parser.JobConfig{
			"build": {
				Stage:  "build",
				Script: []string{"make build"},
			},
			"test": {
				Stage:        "test",
				Script:       []string{"make test"},
				Dependencies: []string{"build"},
			},
		},
	}

	newConfig := &parser.GitLabConfig{
		Stages: []string{"build", "test", "deploy"},
		Jobs: map[string]*parser.JobConfig{
			"build": {
				Stage:  "build",
				Script: []string{"make build"},
			},
			"test:unit": {
				Stage:        "test",
				Script:       []string{"make test"},
				Dependencies: []string{"build"},
			},
			"test:integration": {
				Stage:        "test",
				Script:       []string{"make integration-test"},
				Dependencies: []string{"build"},
			},
			"deploy": {
				Stage:        "deploy",
				Script:       []string{"make deploy"},
				Dependencies: []string{"test:unit", "test:integration"},
			},
		},
	}

	// Create a mock comparison
	comparison := &PipelineComparison{
		JobComparisons: []JobComparison{
			{
				JobName: "build",
				Status:  StatusIdentical,
			},
			{
				JobName: "test",
				Status:  StatusRemoved,
			},
			{
				JobName: "test:unit",
				Status:  StatusAdded,
			},
			{
				JobName: "test:integration",
				Status:  StatusAdded,
			},
			{
				JobName: "deploy",
				Status:  StatusAdded,
			},
		},
	}

	vr := NewVisualRenderer()
	result, err := vr.RenderComparisonGraph(oldConfig, newConfig, comparison, FormatMermaid)

	if err != nil {
		t.Fatalf("RenderComparisonGraph failed: %v", err)
	}

	// Check that it contains both "Before" and "After" subgraphs
	if !strings.Contains(result, `subgraph B["Before"]`) {
		t.Error("Expected to find 'Before' subgraph in comparison")
	}
	if !strings.Contains(result, `subgraph A["After"]`) {
		t.Error("Expected to find 'After' subgraph in comparison")
	}

	// Check for job nodes with prefixes
	if !strings.Contains(result, "bbuild") { // Before: build
		t.Error("Expected to find 'bbuild' (before build job) in comparison")
	}
	if !strings.Contains(result, "abuild") { // After: build
		t.Error("Expected to find 'abuild' (after build job) in comparison")
	}

	// Check for comparison styling classes
	expectedClasses := []string{"improved", "degraded", "identical", "added", "removed"}
	foundClasses := 0
	for _, class := range expectedClasses {
		if strings.Contains(result, "classDef "+class) {
			foundClasses++
		}
	}
	if foundClasses == 0 {
		t.Error("Expected to find comparison styling classes in Mermaid output")
	}
}

func TestVisualRenderer_RenderComparisonGraph_DOT(t *testing.T) {
	oldConfig := &parser.GitLabConfig{
		Stages: []string{"test"},
		Jobs: map[string]*parser.JobConfig{
			"slow_test": {
				Stage:  "test",
				Script: []string{"sleep 60 && test"},
			},
		},
	}

	newConfig := &parser.GitLabConfig{
		Stages: []string{"test"},
		Jobs: map[string]*parser.JobConfig{
			"fast_test": {
				Stage:  "test",
				Script: []string{"test --parallel"},
			},
		},
	}

	comparison := &PipelineComparison{
		JobComparisons: []JobComparison{
			{
				JobName: "slow_test",
				Status:  StatusRemoved,
			},
			{
				JobName: "fast_test",
				Status:  StatusAdded,
			},
		},
	}

	vr := NewVisualRenderer()
	result, err := vr.RenderComparisonGraph(oldConfig, newConfig, comparison, FormatDOT)

	if err != nil {
		t.Fatalf("RenderComparisonGraph failed: %v", err)
	}

	// Check for before/after clusters
	if !strings.Contains(result, `subgraph cluster_old`) {
		t.Error("Expected to find 'cluster_old' subgraph in DOT comparison")
	}
	if !strings.Contains(result, `subgraph cluster_new`) {
		t.Error("Expected to find 'cluster_new' subgraph in DOT comparison")
	}

	// Check cluster labels
	if !strings.Contains(result, `label="Before"`) {
		t.Error("Expected to find 'Before' label in DOT comparison")
	}
	if !strings.Contains(result, `label="After"`) {
		t.Error("Expected to find 'After' label in DOT comparison")
	}
}

func TestVisualRenderer_GroupJobsByStage(t *testing.T) {
	config := &parser.GitLabConfig{
		Stages: []string{"build", "test", "deploy"},
		Jobs: map[string]*parser.JobConfig{
			"build:frontend": {Stage: "build"},
			"build:backend":  {Stage: "build"},
			"test:unit":      {Stage: "test"},
			"deploy:prod":    {Stage: "deploy"},
			".template":      {Stage: "build"}, // Should be ignored
		},
	}

	vr := NewVisualRenderer()
	stageJobs := vr.groupJobsByStage(config)

	// Check that template jobs are excluded
	buildJobs, ok := stageJobs["build"]
	if !ok {
		t.Fatal("Expected 'build' stage in grouped jobs")
	}

	expectedBuildJobs := []string{"build:backend", "build:frontend"} // Should be sorted
	if len(buildJobs) != len(expectedBuildJobs) {
		t.Errorf("Expected %d build jobs, got %d", len(expectedBuildJobs), len(buildJobs))
	}

	for i, expected := range expectedBuildJobs {
		if buildJobs[i] != expected {
			t.Errorf("Expected build job %d to be '%s', got '%s'", i, expected, buildJobs[i])
		}
	}

	// Check that template job is excluded
	for _, jobs := range stageJobs {
		for _, job := range jobs {
			if strings.HasPrefix(job, ".") {
				t.Errorf("Template job '%s' should not be included in stage grouping", job)
			}
		}
	}
}

func TestVisualRenderer_SanitizeMermaidID(t *testing.T) {
	vr := NewVisualRenderer()

	testCases := []struct {
		input    string
		expected string
	}{
		{"simple", "simple"},
		{"job:with:colons", "job_with_colons"},
		{"job-with-dashes", "job_with_dashes"},
		{"job.with.dots", "job_with_dots"},
		{"job with spaces", "job_with_spaces"},
		{"complex:job-name.with_all", "complex_job_name_with_all"},
	}

	for _, tc := range testCases {
		result := vr.sanitizeMermaidID(tc.input)
		if result != tc.expected {
			t.Errorf("sanitizeMermaidID(%q) = %q, expected %q", tc.input, result, tc.expected)
		}
	}
}

func TestVisualRenderer_GetJobNodeColor(t *testing.T) {
	vr := NewVisualRenderer()

	testCases := []struct {
		stage    string
		expected string
	}{
		{"build", "lightblue"},
		{"test", "lightpink"},
		{"deploy", "lightgreen"},
		{"custom", "lightyellow"},
		{"BUILD", "lightblue"}, // Case insensitive
		{"TEST", "lightpink"},
		{"DEPLOY", "lightgreen"},
	}

	for _, tc := range testCases {
		job := &parser.JobConfig{Stage: tc.stage}
		result := vr.getJobNodeColor(job)
		if result != tc.expected {
			t.Errorf("getJobNodeColor(stage=%q) = %q, expected %q", tc.stage, result, tc.expected)
		}
	}
}

func TestVisualRenderer_GetComparisonEdgeColor(t *testing.T) {
	vr := NewVisualRenderer()

	testCases := []struct {
		status   CompareStatus
		expected string
	}{
		{StatusImproved, "green"},
		{StatusDegraded, "red"},
		{StatusIdentical, "blue"},
		{StatusRestructured, "orange"},
		{StatusAdded, "gray"},
		{StatusRemoved, "gray"},
	}

	for _, tc := range testCases {
		result := vr.getComparisonEdgeColor(tc.status)
		if result != tc.expected {
			t.Errorf("getComparisonEdgeColor(%v) = %q, expected %q", tc.status, result, tc.expected)
		}
	}
}

func TestVisualRenderer_UnsupportedFormat(t *testing.T) {
	vr := NewVisualRenderer()
	config := &parser.GitLabConfig{
		Jobs: map[string]*parser.JobConfig{
			"test": {Stage: "test"},
		},
	}

	_, err := vr.RenderPipelineGraph(config, "unsupported")
	if err == nil {
		t.Error("Expected error for unsupported format")
	}

	expectedError := "unsupported visual format: unsupported"
	if err.Error() != expectedError {
		t.Errorf("Expected error message '%s', got '%s'", expectedError, err.Error())
	}
}

func TestVisualRenderer_EmptyConfiguration(t *testing.T) {
	vr := NewVisualRenderer()
	config := &parser.GitLabConfig{
		Stages: []string{},
		Jobs:   map[string]*parser.JobConfig{},
	}

	// Should not error with empty configuration
	result, err := vr.RenderPipelineGraph(config, FormatMermaid)
	if err != nil {
		t.Fatalf("Expected no error for empty config, got: %v", err)
	}

	// Should still generate valid Mermaid syntax
	if !strings.Contains(result, "flowchart TD") {
		t.Error("Expected valid Mermaid flowchart even for empty config")
	}
}
