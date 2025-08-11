package renderer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

// FormatComparison formats a pipeline comparison for display
func (r *Renderer) FormatComparison(comparison *PipelineComparison, format string) (string, error) {
	switch format {
	case "json":
		data, err := json.MarshalIndent(comparison, "", "  ")
		if err != nil {
			return "", err
		}
		return string(data), nil

	case "table", "":
		return r.formatComparisonTable(comparison), nil

	case "dot", "mermaid":
		// Visual formats require configuration data, which isn't available here
		// These should be handled by RenderVisualComparison instead
		return "", fmt.Errorf("visual format %s requires using RenderVisualComparison with configuration data", format)

	default:
		return "", fmt.Errorf("unsupported format: %s (supported: json, table, dot, mermaid)", format)
	}
}

// formatComparisonTable formats comparison as a table
func (r *Renderer) formatComparisonTable(comparison *PipelineComparison) string {
	var buf bytes.Buffer

	buf.WriteString("Pipeline Execution Comparison\n")
	buf.WriteString("============================\n\n")

	// Summary section
	summary := comparison.Summary
	buf.WriteString("Summary:\n")
	buf.WriteString("--------\n")
	buf.WriteString(fmt.Sprintf("  Total Jobs: %d\n", summary.TotalJobs))
	buf.WriteString(fmt.Sprintf("  Added Jobs: %d\n", summary.AddedJobs))
	buf.WriteString(fmt.Sprintf("  Removed Jobs: %d\n", summary.RemovedJobs))
	buf.WriteString(fmt.Sprintf("  Improved Jobs: %d\n", summary.ImprovedJobs))
	buf.WriteString(fmt.Sprintf("  Degraded Jobs: %d\n", summary.DegradedJobs))
	buf.WriteString(fmt.Sprintf("  Identical Jobs: %d\n", summary.IdenticalJobs))
	buf.WriteString(fmt.Sprintf("  Total Time Change: %.2fs\n", summary.TotalTimeChange))

	if summary.OverallImprovement {
		buf.WriteString("  Overall: ✓ Performance improved\n")
	} else {
		buf.WriteString("  Overall: ⚠ Performance degraded\n")
	}

	// Performance metrics
	perf := comparison.PerformanceGain
	buf.WriteString("\nPerformance Metrics:\n")
	buf.WriteString("-------------------\n")
	buf.WriteString(fmt.Sprintf("  Pipeline Duration Change: %.2fs\n", perf.TotalPipelineDuration))
	buf.WriteString(fmt.Sprintf("  Average Job Duration Change: %.2fs\n", perf.AverageJobDuration))
	buf.WriteString(fmt.Sprintf("  Parallelism Improvement: %d jobs\n", perf.ParallelismImprovement))
	buf.WriteString(fmt.Sprintf("  Startup Time Reduction: %.2fs\n", perf.StartupTimeReduction))

	// Job-by-job comparison
	buf.WriteString("\nJob Comparisons:\n")
	buf.WriteString("---------------\n")

	for _, jobComp := range comparison.JobComparisons {
		status := r.formatJobStatus(jobComp.Status)
		buf.WriteString(fmt.Sprintf("  [%s] %s: ", status, jobComp.JobName))

		switch jobComp.Status {
		case StatusAdded:
			buf.WriteString("Job added\n")
		case StatusRemoved:
			buf.WriteString("Job removed\n")
		case StatusIdentical:
			buf.WriteString("No changes\n")
		default:
			buf.WriteString(fmt.Sprintf("Duration change: %.2fs", jobComp.DurationChange))
			if len(jobComp.Changes) > 0 {
				buf.WriteString(fmt.Sprintf(" (%s)", strings.Join(jobComp.Changes, ", ")))
			}
			buf.WriteString("\n")
		}
	}

	return buf.String()
}

func (r *Renderer) formatJobStatus(status CompareStatus) string {
	switch status {
	case StatusIdentical:
		return "="
	case StatusImproved:
		return "✓"
	case StatusDegraded:
		return "⚠"
	case StatusAdded:
		return "+"
	case StatusRemoved:
		return "-"
	case StatusRestructured:
		return "~"
	default:
		return "?"
	}
}
