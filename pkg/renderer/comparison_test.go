package renderer

import (
	"testing"

	"github.com/wonderfulspam/gitlab-smith/pkg/parser"
)

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
