package renderer

import (
	"encoding/json"
	"strings"
	"testing"
)

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
