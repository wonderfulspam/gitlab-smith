package parser

import (
	"strings"
)

// SimulateMainBranchPipeline simulates which jobs would run on main branch
func (c *GitLabConfig) SimulateMainBranchPipeline() map[string]bool {
	context := DefaultPipelineContext()
	return c.SimulatePipeline(context)
}

// SimulateMergeRequestPipeline simulates which jobs would run in a merge request
func (c *GitLabConfig) SimulateMergeRequestPipeline(sourceBranch string) map[string]bool {
	context := MergeRequestPipelineContext(sourceBranch)
	return c.SimulatePipeline(context)
}

// SimulatePipeline simulates which jobs would run in the given pipeline context
func (c *GitLabConfig) SimulatePipeline(context *PipelineContext) map[string]bool {
	result := make(map[string]bool)

	// First check if pipeline should be created at all
	evaluator := NewWorkflowEvaluator(c, context)
	if !evaluator.ShouldCreatePipeline() {
		// No jobs run if pipeline is not created
		return result
	}

	// Evaluate each job's rules to see if it should run
	for jobName, job := range c.Jobs {
		result[jobName] = c.shouldJobRun(job, context)
	}

	return result
}

// shouldJobRun evaluates if a job should run in the given context
func (c *GitLabConfig) shouldJobRun(job *JobConfig, context *PipelineContext) bool {
	// If job has rules, evaluate them
	if len(job.Rules) > 0 {
		return c.evaluateJobRules(job, context)
	}

	// If job has only/except, evaluate them (legacy)
	if job.Only != nil || job.Except != nil {
		return c.evaluateOnlyExcept(job, context)
	}

	// Default behavior: job runs
	return true
}

// evaluateJobRules evaluates job rules to determine if job should run
func (c *GitLabConfig) evaluateJobRules(job *JobConfig, context *PipelineContext) bool {
	for _, rule := range job.Rules {
		if c.ruleMatches(&rule, context) {
			switch rule.When {
			case "never":
				return false
			case "always", "":
				return true
			case "on_success", "on_failure", "manual", "delayed":
				// These depend on previous job status, assume true for simulation
				return true
			}
		}
	}

	// No rule matched, default behavior is job doesn't run
	return false
}

// ruleMatches checks if a rule matches the current context (simplified)
func (c *GitLabConfig) ruleMatches(rule *Rule, context *PipelineContext) bool {
	// If no conditions, rule matches
	if rule.If == "" && len(rule.Changes) == 0 && len(rule.Exists) == 0 {
		return true
	}

	// Simple if condition evaluation
	if rule.If != "" {
		return c.evaluateSimpleIfCondition(rule.If, context)
	}

	// For changes/exists, we can't evaluate without file system, assume true
	return len(rule.Changes) == 0 && len(rule.Exists) == 0
}

// evaluateSimpleIfCondition provides basic evaluation of if conditions
func (c *GitLabConfig) evaluateSimpleIfCondition(condition string, context *PipelineContext) bool {
	// This is a simplified version - in practice GitLab has complex expression evaluation
	condition = strings.TrimSpace(condition)

	// Common patterns
	if strings.Contains(condition, "$CI_PIPELINE_SOURCE == \"push\"") {
		return context.Event == "push"
	}
	if strings.Contains(condition, "$CI_PIPELINE_SOURCE == \"merge_request_event\"") {
		return context.Event == "merge_request_event"
	}
	if strings.Contains(condition, "$CI_COMMIT_BRANCH == \"main\"") ||
		strings.Contains(condition, "$CI_COMMIT_BRANCH == \"master\"") {
		return context.IsMainBranch
	}
	if strings.Contains(condition, "$CI_MERGE_REQUEST_ID") {
		return context.IsMR
	}

	// Default to true for unknown conditions
	return true
}

// evaluateOnlyExcept evaluates legacy only/except directives
func (c *GitLabConfig) evaluateOnlyExcept(job *JobConfig, context *PipelineContext) bool {
	// This is a simplified implementation of only/except logic
	// In practice, GitLab has complex matching rules for refs, variables, etc.

	// If only is specified, job runs only if conditions match
	if job.Only != nil {
		return c.matchesOnlyExcept(job.Only, context, true)
	}

	// If except is specified, job runs unless conditions match
	if job.Except != nil {
		return !c.matchesOnlyExcept(job.Except, context, false)
	}

	return true
}

// matchesOnlyExcept checks if only/except conditions match
func (c *GitLabConfig) matchesOnlyExcept(condition interface{}, context *PipelineContext, isOnly bool) bool {
	switch v := condition.(type) {
	case []interface{}:
		// Array of conditions
		for _, item := range v {
			if str, ok := item.(string); ok {
				if c.matchesSingleCondition(str, context) {
					return true
				}
			}
		}
	case []string:
		for _, str := range v {
			if c.matchesSingleCondition(str, context) {
				return true
			}
		}
	case string:
		return c.matchesSingleCondition(v, context)
	}

	return false
}

// matchesSingleCondition checks if a single condition string matches
func (c *GitLabConfig) matchesSingleCondition(condition string, context *PipelineContext) bool {
	switch condition {
	case "master", "main":
		return context.IsMainBranch
	case "merge_requests":
		return context.IsMR
	case "pushes":
		return context.Event == "push"
	default:
		// Could be a branch name or pattern
		return condition == context.Branch
	}
}
