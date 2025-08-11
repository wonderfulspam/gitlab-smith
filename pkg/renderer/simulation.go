package renderer

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/wonderfulspam/gitlab-smith/pkg/parser"
)

// CompareConfigurations simulates pipeline execution based on configurations
func (r *Renderer) CompareConfigurations(oldConfig, newConfig *parser.GitLabConfig) (*PipelineComparison, error) {
	oldSimulation := r.simulatePipelineExecution(oldConfig)
	newSimulation := r.simulatePipelineExecution(newConfig)

	return r.compareExecutions(oldSimulation, newSimulation), nil
}

// simulatePipelineExecution creates a simulated pipeline execution from a config
func (r *Renderer) simulatePipelineExecution(config *parser.GitLabConfig) *PipelineExecution {
	pipeline := &PipelineExecution{
		ID:        0, // Simulated
		Status:    "simulated",
		Ref:       "main",
		SHA:       "simulated",
		Jobs:      make([]JobExecution, 0),
		Variables: convertVariables(config.Variables),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Convert parsed jobs to job executions
	for jobName, job := range config.Jobs {
		if job == nil {
			continue
		}

		// Skip template jobs (starting with .) as they don't run independently
		if strings.HasPrefix(jobName, ".") {
			continue
		}

		jobExec := JobExecution{
			ID:             0, // Simulated
			Name:           jobName,
			Stage:          job.Stage,
			Status:         "simulated",
			Dependencies:   job.Dependencies,
			Needs:          extractJobNames(job.Needs),
			Duration:       estimateJobDurationWithContext(job, config.Jobs),
			QueuedDuration: 0,
		}

		pipeline.Jobs = append(pipeline.Jobs, jobExec)
	}

	// Sort jobs by stage order
	sort.Slice(pipeline.Jobs, func(i, j int) bool {
		return getStageOrder(pipeline.Jobs[i].Stage, config.Stages) <
			getStageOrder(pipeline.Jobs[j].Stage, config.Stages)
	})

	return pipeline
}

func convertVariables(vars map[string]interface{}) map[string]string {
	if vars == nil {
		return make(map[string]string)
	}

	result := make(map[string]string, len(vars))
	for k, v := range vars {
		if str, ok := v.(string); ok {
			result[k] = str
		} else {
			result[k] = fmt.Sprintf("%v", v)
		}
	}
	return result
}

func extractJobNames(needs interface{}) []string {
	if needs == nil {
		return []string{}
	}

	// Handle different need formats
	switch v := needs.(type) {
	case []string:
		return v
	case []interface{}:
		names := make([]string, 0, len(v))
		for _, item := range v {
			if str, ok := item.(string); ok {
				names = append(names, str)
			} else if job, ok := item.(map[string]interface{}); ok {
				if jobName, exists := job["job"]; exists {
					if str, ok := jobName.(string); ok {
						names = append(names, str)
					}
				}
			}
		}
		return names
	case string:
		return []string{v}
	default:
		return []string{}
	}
}

func estimateJobDuration(job *parser.JobConfig) float64 {
	// Simple heuristic: base duration + script length factor
	baseDuration := 30.0                           // 30 seconds base
	scriptFactor := float64(len(job.Script)) * 2.0 // 2 seconds per script line

	// Add before_script overhead (typically setup commands)
	beforeScriptFactor := float64(len(job.BeforeScript)) * 2.0 // 2 seconds per before_script line

	if len(job.Services) > 0 {
		baseDuration += 15.0 // Additional time for services
	}

	return baseDuration + scriptFactor + beforeScriptFactor
}

// estimateJobDurationWithContext considers template inheritance for more accurate estimation
func estimateJobDurationWithContext(job *parser.JobConfig, allJobs map[string]*parser.JobConfig) float64 {
	baseDuration := 30.0                           // 30 seconds base
	scriptFactor := float64(len(job.Script)) * 2.0 // 2 seconds per script line

	// Calculate before_script - either direct or from template
	beforeScriptLines := len(job.BeforeScript)

	// If job uses extends, get before_script from template
	extendsTemplates := extractExtendsTemplates(job.Extends)
	if len(extendsTemplates) > 0 {
		for _, templateName := range extendsTemplates {
			if template, exists := allJobs[templateName]; exists && template != nil {
				beforeScriptLines += len(template.BeforeScript)
			}
		}
	}

	beforeScriptFactor := float64(beforeScriptLines) * 2.0

	if len(job.Services) > 0 {
		baseDuration += 15.0 // Additional time for services
	}

	// Optimization bonus: if using templates, reduce overhead slightly due to better caching/reuse
	optimizationBonus := 0.0
	if len(extendsTemplates) > 0 {
		optimizationBonus = 3.0 // Small improvement from template reuse
	}

	duration := baseDuration + scriptFactor + beforeScriptFactor - optimizationBonus
	if duration < 10.0 {
		duration = 10.0 // Minimum duration
	}

	return duration
}

func extractExtendsTemplates(extends interface{}) []string {
	if extends == nil {
		return []string{}
	}

	// Handle different extend formats
	switch v := extends.(type) {
	case string:
		return []string{v}
	case []string:
		return v
	case []interface{}:
		templates := make([]string, 0, len(v))
		for _, item := range v {
			if str, ok := item.(string); ok {
				templates = append(templates, str)
			}
		}
		return templates
	default:
		return []string{}
	}
}

func getStageOrder(stageName string, stages []string) int {
	for i, stage := range stages {
		if stage == stageName {
			return i
		}
	}
	return 999 // Unknown stage goes last
}
