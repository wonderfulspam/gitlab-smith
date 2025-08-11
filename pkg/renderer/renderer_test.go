package renderer

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/emt/gitlab-smith/pkg/parser"
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

func TestRenderer_CompareExecutions(t *testing.T) {
	renderer := New(nil)

	oldPipeline := &PipelineExecution{
		ID:       1,
		Duration: 300,
		Jobs: []JobExecution{
			{Name: "build", Stage: "build", Duration: 60.0, Status: "success"},
			{Name: "test", Stage: "test", Duration: 120.0, Status: "success"},
			{Name: "deploy", Stage: "deploy", Duration: 120.0, Status: "success"},
		},
	}

	newPipeline := &PipelineExecution{
		ID:       2,
		Duration: 250,
		Jobs: []JobExecution{
			{Name: "build", Stage: "build", Duration: 45.0, Status: "success"},   // Improved
			{Name: "test", Stage: "test", Duration: 120.0, Status: "success"},    // Same
			{Name: "deploy", Stage: "deploy", Duration: 85.0, Status: "success"}, // Improved
			{Name: "lint", Stage: "test", Duration: 30.0, Status: "success"},     // Added
		},
	}

	comparison := renderer.compareExecutions(oldPipeline, newPipeline)

	if comparison == nil {
		t.Fatal("Expected non-nil comparison")
	}

	// Verify summary
	summary := comparison.Summary
	if summary.TotalJobs != 4 {
		t.Errorf("Expected 4 total jobs, got %d", summary.TotalJobs)
	}

	if summary.AddedJobs != 1 {
		t.Errorf("Expected 1 added job, got %d", summary.AddedJobs)
	}

	if summary.ImprovedJobs != 2 { // build and deploy improved
		t.Errorf("Expected 2 improved jobs, got %d", summary.ImprovedJobs)
	}

	if summary.IdenticalJobs != 1 { // test unchanged
		t.Errorf("Expected 1 identical job, got %d", summary.IdenticalJobs)
	}

	if !summary.OverallImprovement {
		t.Error("Expected overall improvement to be true")
	}

	// Verify job comparisons
	if len(comparison.JobComparisons) != 4 {
		t.Errorf("Expected 4 job comparisons, got %d", len(comparison.JobComparisons))
	}

	// Find specific job comparisons
	var buildComp, testComp, deployComp, lintComp *JobComparison
	for i := range comparison.JobComparisons {
		comp := &comparison.JobComparisons[i]
		switch comp.JobName {
		case "build":
			buildComp = comp
		case "test":
			testComp = comp
		case "deploy":
			deployComp = comp
		case "lint":
			lintComp = comp
		}
	}

	if buildComp.Status != StatusImproved {
		t.Errorf("Expected build job to be improved, got %s", buildComp.Status)
	}

	if buildComp.DurationChange != -15.0 { // 45 - 60 = -15
		t.Errorf("Expected build duration change of -15.0, got %f", buildComp.DurationChange)
	}

	if testComp.Status != StatusIdentical {
		t.Errorf("Expected test job to be identical, got %s", testComp.Status)
	}

	if deployComp.Status != StatusImproved {
		t.Errorf("Expected deploy job to be improved, got %s", deployComp.Status)
	}

	if lintComp.Status != StatusAdded {
		t.Errorf("Expected lint job to be added, got %s", lintComp.Status)
	}
}

func TestRenderer_CompareJobs(t *testing.T) {
	renderer := New(nil)

	tests := []struct {
		name           string
		oldJob         *JobExecution
		newJob         *JobExecution
		expectedStatus CompareStatus
		expectChanges  bool
	}{
		{
			name:           "identical jobs",
			oldJob:         &JobExecution{Name: "test", Stage: "test", Duration: 60.0},
			newJob:         &JobExecution{Name: "test", Stage: "test", Duration: 60.0},
			expectedStatus: StatusIdentical,
			expectChanges:  false,
		},
		{
			name:           "improved job",
			oldJob:         &JobExecution{Name: "test", Stage: "test", Duration: 60.0},
			newJob:         &JobExecution{Name: "test", Stage: "test", Duration: 50.0},
			expectedStatus: StatusImproved,
			expectChanges:  false,
		},
		{
			name:           "degraded job",
			oldJob:         &JobExecution{Name: "test", Stage: "test", Duration: 60.0},
			newJob:         &JobExecution{Name: "test", Stage: "test", Duration: 80.0},
			expectedStatus: StatusDegraded,
			expectChanges:  false,
		},
		{
			name:           "added job",
			oldJob:         nil,
			newJob:         &JobExecution{Name: "test", Stage: "test", Duration: 60.0},
			expectedStatus: StatusAdded,
			expectChanges:  true,
		},
		{
			name:           "removed job",
			oldJob:         &JobExecution{Name: "test", Stage: "test", Duration: 60.0},
			newJob:         nil,
			expectedStatus: StatusRemoved,
			expectChanges:  true,
		},
		{
			name:           "restructured job",
			oldJob:         &JobExecution{Name: "test", Stage: "test", Duration: 60.0},
			newJob:         &JobExecution{Name: "test", Stage: "build", Duration: 62.0},
			expectedStatus: StatusRestructured,
			expectChanges:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			comparison := renderer.compareJobs("test", tt.oldJob, tt.newJob)

			if comparison.Status != tt.expectedStatus {
				t.Errorf("Expected status %s, got %s", tt.expectedStatus, comparison.Status)
			}

			if tt.expectChanges && len(comparison.Changes) == 0 {
				t.Error("Expected changes but got none")
			} else if !tt.expectChanges && len(comparison.Changes) > 0 {
				t.Errorf("Expected no changes but got: %v", comparison.Changes)
			}
		})
	}
}

func TestRenderer_FormatComparison(t *testing.T) {
	renderer := New(nil)

	comparison := &PipelineComparison{
		Summary: ComparisonSummary{
			TotalJobs:          3,
			AddedJobs:          1,
			ImprovedJobs:       1,
			IdenticalJobs:      1,
			OverallImprovement: true,
			TotalTimeChange:    -15.5,
		},
		PerformanceGain: PerformanceMetrics{
			TotalPipelineDuration:  -30.0,
			AverageJobDuration:     -5.0,
			ParallelismImprovement: 2,
			StartupTimeReduction:   10.0,
		},
		JobComparisons: []JobComparison{
			{
				JobName:        "build",
				Status:         StatusImproved,
				DurationChange: -15.0,
				Changes:        []string{},
			},
			{
				JobName: "lint",
				Status:  StatusAdded,
				Changes: []string{"Job added to pipeline"},
			},
		},
	}

	// Test JSON format
	jsonOutput, err := renderer.FormatComparison(comparison, "json")
	if err != nil {
		t.Errorf("Expected no error for JSON format, got: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(jsonOutput), &parsed); err != nil {
		t.Errorf("Expected valid JSON output, got error: %v", err)
	}

	// Test table format
	tableOutput, err := renderer.FormatComparison(comparison, "table")
	if err != nil {
		t.Errorf("Expected no error for table format, got: %v", err)
	}

	if !strings.Contains(tableOutput, "Pipeline Execution Comparison") {
		t.Error("Expected table output to contain title")
	}

	if !strings.Contains(tableOutput, "Total Jobs: 3") {
		t.Error("Expected table output to contain job count")
	}

	if !strings.Contains(tableOutput, "✓ Performance improved") {
		t.Error("Expected table output to indicate improvement")
	}

	// Test invalid format
	_, err = renderer.FormatComparison(comparison, "invalid")
	if err == nil {
		t.Error("Expected error for invalid format")
	}
}

func TestRenderer_GitLabClientIntegration(t *testing.T) {
	// Mock GitLab API server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/pipelines/123") && !strings.Contains(r.URL.Path, "/jobs") {
			// Pipeline endpoint
			pipeline := PipelineExecution{
				ID:       123,
				Status:   "success",
				Ref:      "main",
				SHA:      "abc123",
				Duration: 300,
			}
			json.NewEncoder(w).Encode(pipeline)
		} else if strings.Contains(r.URL.Path, "/pipelines/123/jobs") {
			// Jobs endpoint
			jobs := []JobExecution{
				{
					ID:       1,
					Name:     "build",
					Stage:    "build",
					Status:   "success",
					Duration: 60.0,
				},
				{
					ID:       2,
					Name:     "test",
					Stage:    "test",
					Status:   "success",
					Duration: 120.0,
				},
			}
			json.NewEncoder(w).Encode(jobs)
		} else {
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := NewGitLabClient(server.URL, "test-token", "123")
	renderer := New(client)

	ctx := context.Background()
	pipeline, err := renderer.RenderPipeline(ctx, 123)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if pipeline.ID != 123 {
		t.Errorf("Expected pipeline ID 123, got %d", pipeline.ID)
	}

	if len(pipeline.Jobs) != 2 {
		t.Errorf("Expected 2 jobs, got %d", len(pipeline.Jobs))
	}

	if pipeline.Jobs[0].Name != "build" {
		t.Errorf("Expected first job to be 'build', got %s", pipeline.Jobs[0].Name)
	}
}

func TestRenderer_CompareConfigurations(t *testing.T) {
	renderer := New(nil)

	oldConfig := &parser.GitLabConfig{
		Stages: []string{"build", "test"},
		Jobs: map[string]*parser.JobConfig{
			"build": {Stage: "build", Script: []string{"make build"}},
			"test":  {Stage: "test", Script: []string{"make test"}},
		},
	}

	newConfig := &parser.GitLabConfig{
		Stages: []string{"build", "test", "deploy"},
		Jobs: map[string]*parser.JobConfig{
			"build":  {Stage: "build", Script: []string{"make build"}},
			"test":   {Stage: "test", Script: []string{"make test", "make coverage"}}, // More scripts
			"deploy": {Stage: "deploy", Script: []string{"make deploy"}},              // New job
		},
	}

	comparison, err := renderer.CompareConfigurations(oldConfig, newConfig)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if comparison.Summary.TotalJobs != 3 {
		t.Errorf("Expected 3 total jobs, got %d", comparison.Summary.TotalJobs)
	}

	if comparison.Summary.AddedJobs != 1 {
		t.Errorf("Expected 1 added job, got %d", comparison.Summary.AddedJobs)
	}

	// Find the deploy job comparison
	var deployComp *JobComparison
	for i := range comparison.JobComparisons {
		if comparison.JobComparisons[i].JobName == "deploy" {
			deployComp = &comparison.JobComparisons[i]
			break
		}
	}

	if deployComp == nil {
		t.Fatal("Expected to find deploy job comparison")
	}

	if deployComp.Status != StatusAdded {
		t.Errorf("Expected deploy job to be added, got %s", deployComp.Status)
	}
}

func TestRenderer_CalculatePerformanceMetrics(t *testing.T) {
	renderer := New(nil)

	oldPipeline := &PipelineExecution{
		Duration: 300,
		Jobs: []JobExecution{
			{Duration: 60.0, QueuedDuration: 10.0},
			{Duration: 120.0, QueuedDuration: 15.0},
			{Duration: 120.0, QueuedDuration: 20.0},
		},
	}

	newPipeline := &PipelineExecution{
		Duration: 250,
		Jobs: []JobExecution{
			{Duration: 45.0, QueuedDuration: 5.0},
			{Duration: 100.0, QueuedDuration: 8.0},
			{Duration: 105.0, QueuedDuration: 7.0},
		},
	}

	metrics := renderer.calculatePerformanceMetrics(oldPipeline, newPipeline)

	expectedPipelineDurationChange := float64(250 - 300) // -50
	if metrics.TotalPipelineDuration != expectedPipelineDurationChange {
		t.Errorf("Expected pipeline duration change of %f, got %f",
			expectedPipelineDurationChange, metrics.TotalPipelineDuration)
	}

	// Check average job duration calculation
	oldAvg := (60.0 + 120.0 + 120.0) / 3.0 // 100.0
	newAvg := (45.0 + 100.0 + 105.0) / 3.0 // 83.33
	expectedAvgChange := newAvg - oldAvg

	if abs(metrics.AverageJobDuration-expectedAvgChange) > 0.01 {
		t.Errorf("Expected average job duration change of %f, got %f",
			expectedAvgChange, metrics.AverageJobDuration)
	}

	// Check startup time reduction
	oldAvgQueue := (10.0 + 15.0 + 20.0) / 3.0           // 15.0
	newAvgQueue := (5.0 + 8.0 + 7.0) / 3.0              // 6.67
	expectedQueueReduction := oldAvgQueue - newAvgQueue // 8.33

	if abs(metrics.StartupTimeReduction-expectedQueueReduction) > 0.01 {
		t.Errorf("Expected startup time reduction of %f, got %f",
			expectedQueueReduction, metrics.StartupTimeReduction)
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
