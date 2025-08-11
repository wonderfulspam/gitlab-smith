package renderer

import (
	"context"
	"fmt"
	"sort"
)

// ComparePipelines compares two pipeline executions and provides detailed analysis
func (r *Renderer) ComparePipelines(ctx context.Context, oldPipelineID, newPipelineID int) (*PipelineComparison, error) {
	oldPipeline, err := r.RenderPipeline(ctx, oldPipelineID)
	if err != nil {
		return nil, fmt.Errorf("failed to render old pipeline: %w", err)
	}

	newPipeline, err := r.RenderPipeline(ctx, newPipelineID)
	if err != nil {
		return nil, fmt.Errorf("failed to render new pipeline: %w", err)
	}

	return r.compareExecutions(oldPipeline, newPipeline), nil
}

// compareExecutions performs detailed comparison between two pipeline executions
func (r *Renderer) compareExecutions(oldPipeline, newPipeline *PipelineExecution) *PipelineComparison {
	comparison := &PipelineComparison{
		OldExecution:   oldPipeline,
		NewExecution:   newPipeline,
		JobComparisons: make([]JobComparison, 0),
	}

	// Create job lookup maps
	oldJobs := make(map[string]*JobExecution)
	newJobs := make(map[string]*JobExecution)

	for i := range oldPipeline.Jobs {
		oldJobs[oldPipeline.Jobs[i].Name] = &oldPipeline.Jobs[i]
	}

	for i := range newPipeline.Jobs {
		newJobs[newPipeline.Jobs[i].Name] = &newPipeline.Jobs[i]
	}

	// Get all unique job names
	allJobNames := make(map[string]bool)
	for name := range oldJobs {
		allJobNames[name] = true
	}
	for name := range newJobs {
		allJobNames[name] = true
	}

	// Compare each job
	var totalTimeChange float64
	summary := ComparisonSummary{}

	for jobName := range allJobNames {
		oldJob := oldJobs[jobName]
		newJob := newJobs[jobName]

		jobComparison := r.compareJobs(jobName, oldJob, newJob)
		comparison.JobComparisons = append(comparison.JobComparisons, jobComparison)

		// Update summary statistics
		switch jobComparison.Status {
		case StatusAdded:
			summary.AddedJobs++
		case StatusRemoved:
			summary.RemovedJobs++
		case StatusImproved:
			summary.ImprovedJobs++
		case StatusDegraded:
			summary.DegradedJobs++
		case StatusIdentical:
			summary.IdenticalJobs++
		}

		totalTimeChange += jobComparison.DurationChange
		summary.TotalJobs++
	}

	summary.TotalTimeChange = totalTimeChange
	summary.OverallImprovement = totalTimeChange < 0 // Negative means faster

	comparison.Summary = summary
	comparison.PerformanceGain = r.calculatePerformanceMetrics(oldPipeline, newPipeline)

	return comparison
}

// compareJobs compares two individual job executions
func (r *Renderer) compareJobs(jobName string, oldJob, newJob *JobExecution) JobComparison {
	comparison := JobComparison{
		JobName: jobName,
		OldJob:  oldJob,
		NewJob:  newJob,
		Changes: make([]string, 0),
	}

	if oldJob == nil && newJob != nil {
		comparison.Status = StatusAdded
		comparison.DurationChange = newJob.Duration
		comparison.Changes = append(comparison.Changes, "Job added to pipeline")
	} else if oldJob != nil && newJob == nil {
		comparison.Status = StatusRemoved
		comparison.DurationChange = -oldJob.Duration
		comparison.Changes = append(comparison.Changes, "Job removed from pipeline")
	} else {
		// Both jobs exist, compare them
		comparison.DurationChange = newJob.Duration - oldJob.Duration
		comparison.QueueTimeChange = newJob.QueuedDuration - oldJob.QueuedDuration

		if oldJob.Stage != newJob.Stage {
			comparison.Changes = append(comparison.Changes, fmt.Sprintf("Stage changed from %s to %s", oldJob.Stage, newJob.Stage))
		}

		if !equalStringSlices(oldJob.Dependencies, newJob.Dependencies) {
			comparison.Changes = append(comparison.Changes, "Dependencies changed")
		}

		if !equalStringSlices(oldJob.Needs, newJob.Needs) {
			comparison.Changes = append(comparison.Changes, "Needs relationships changed")
		}

		// Determine overall status
		if len(comparison.Changes) == 0 && comparison.DurationChange == 0 {
			comparison.Status = StatusIdentical
		} else if comparison.DurationChange < -5 { // More than 5 seconds improvement
			comparison.Status = StatusImproved
		} else if comparison.DurationChange > 5 { // More than 5 seconds degradation
			comparison.Status = StatusDegraded
		} else {
			comparison.Status = StatusRestructured
		}
	}

	return comparison
}

// calculatePerformanceMetrics calculates performance metrics between pipelines
func (r *Renderer) calculatePerformanceMetrics(oldPipeline, newPipeline *PipelineExecution) PerformanceMetrics {
	oldTotalDuration := float64(oldPipeline.Duration)
	newTotalDuration := float64(newPipeline.Duration)

	oldAvgDuration := r.calculateAverageJobDuration(oldPipeline.Jobs)
	newAvgDuration := r.calculateAverageJobDuration(newPipeline.Jobs)

	return PerformanceMetrics{
		TotalPipelineDuration:  newTotalDuration - oldTotalDuration,
		AverageJobDuration:     newAvgDuration - oldAvgDuration,
		ParallelismImprovement: r.calculateParallelismImprovement(oldPipeline, newPipeline),
		StartupTimeReduction:   r.calculateStartupTimeReduction(oldPipeline, newPipeline),
	}
}

func (r *Renderer) calculateAverageJobDuration(jobs []JobExecution) float64 {
	if len(jobs) == 0 {
		return 0
	}

	total := 0.0
	for _, job := range jobs {
		total += job.Duration
	}

	return total / float64(len(jobs))
}

func (r *Renderer) calculateParallelismImprovement(oldPipeline, newPipeline *PipelineExecution) int {
	// Simple heuristic: count maximum concurrent jobs per stage
	oldParallel := r.countMaxConcurrentJobs(oldPipeline.Jobs)
	newParallel := r.countMaxConcurrentJobs(newPipeline.Jobs)
	return newParallel - oldParallel
}

func (r *Renderer) calculateStartupTimeReduction(oldPipeline, newPipeline *PipelineExecution) float64 {
	oldAvgQueue := 0.0
	newAvgQueue := 0.0

	if len(oldPipeline.Jobs) > 0 {
		for _, job := range oldPipeline.Jobs {
			oldAvgQueue += job.QueuedDuration
		}
		oldAvgQueue /= float64(len(oldPipeline.Jobs))
	}

	if len(newPipeline.Jobs) > 0 {
		for _, job := range newPipeline.Jobs {
			newAvgQueue += job.QueuedDuration
		}
		newAvgQueue /= float64(len(newPipeline.Jobs))
	}

	return oldAvgQueue - newAvgQueue
}

func (r *Renderer) countMaxConcurrentJobs(jobs []JobExecution) int {
	stageJobs := make(map[string]int)
	for _, job := range jobs {
		stageJobs[job.Stage]++
	}

	max := 0
	for _, count := range stageJobs {
		if count > max {
			max = count
		}
	}

	return max
}

func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	aCopy := make([]string, len(a))
	bCopy := make([]string, len(b))
	copy(aCopy, a)
	copy(bCopy, b)

	sort.Strings(aCopy)
	sort.Strings(bCopy)

	for i := range aCopy {
		if aCopy[i] != bCopy[i] {
			return false
		}
	}

	return true
}
