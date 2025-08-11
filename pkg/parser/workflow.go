package parser

import (
	"regexp"
	"strings"
)

// PipelineContext represents the context in which a pipeline is running
type PipelineContext struct {
	Branch      string            // Current branch name
	Variables   map[string]string // GitLab predefined and custom variables
	Event       string            // push, merge_request_event, schedule, api, etc.
	IsMR        bool              // Whether this is a merge request pipeline
	IsMainBranch bool             // Whether this is the main/default branch
}

// WorkflowEvaluator evaluates workflow rules to determine if a pipeline should be created
type WorkflowEvaluator struct {
	config  *GitLabConfig
	context *PipelineContext
}

// NewWorkflowEvaluator creates a new workflow evaluator
func NewWorkflowEvaluator(config *GitLabConfig, context *PipelineContext) *WorkflowEvaluator {
	return &WorkflowEvaluator{
		config:  config,
		context: context,
	}
}

// ShouldCreatePipeline evaluates workflow rules to determine if a pipeline should be created
func (w *WorkflowEvaluator) ShouldCreatePipeline() bool {
	// If no workflow is defined, default behavior is to create pipeline for all events
	if w.config.Workflow == nil || len(w.config.Workflow.Rules) == 0 {
		return true
	}

	// Evaluate each rule in order
	for _, rule := range w.config.Workflow.Rules {
		result := w.evaluateRule(&rule)
		
		// Rules are evaluated in order, first match wins
		if result != nil {
			return *result
		}
	}

	// If no rule matches, default to not creating pipeline
	return false
}

// evaluateRule evaluates a single workflow rule and returns true/false or nil if rule doesn't match
func (w *WorkflowEvaluator) evaluateRule(rule *Rule) *bool {
	// Check if condition matches
	if !w.ruleConditionMatches(rule) {
		return nil // Rule doesn't apply
	}

	// If condition matches, check the when clause
	switch rule.When {
	case "never":
		result := false
		return &result
	case "always", "":
		result := true
		return &result
	default:
		// For any other when value, default to creating pipeline
		result := true
		return &result
	}
}

// ruleConditionMatches checks if a rule's conditions match the current context
func (w *WorkflowEvaluator) ruleConditionMatches(rule *Rule) bool {
	// If no conditions are specified, rule matches all contexts
	if rule.If == "" && len(rule.Changes) == 0 && len(rule.Exists) == 0 {
		return true
	}

	// Evaluate 'if' condition
	if rule.If != "" {
		if !w.evaluateIfCondition(rule.If) {
			return false
		}
	}

	// For changes and exists, we can't fully evaluate without file system access
	// For now, we'll assume they don't match if specified (conservative approach)
	if len(rule.Changes) > 0 || len(rule.Exists) > 0 {
		return false
	}

	return true
}

// evaluateIfCondition evaluates a GitLab CI 'if' expression
func (w *WorkflowEvaluator) evaluateIfCondition(condition string) bool {
	// This is a simplified implementation of GitLab's if condition evaluation
	// In a full implementation, this would need to parse complex expressions
	
	// Handle common patterns
	condition = strings.TrimSpace(condition)
	
	// Handle variable comparisons
	if strings.Contains(condition, "$CI_PIPELINE_SOURCE") {
		return w.evaluateSourceCondition(condition)
	}
	
	if strings.Contains(condition, "$CI_COMMIT_BRANCH") {
		return w.evaluateBranchCondition(condition)
	}
	
	if strings.Contains(condition, "$CI_MERGE_REQUEST_ID") {
		return w.evaluateMRCondition(condition)
	}
	
	// Default to true for unknown conditions (conservative approach)
	return true
}

// evaluateSourceCondition evaluates conditions involving $CI_PIPELINE_SOURCE
func (w *WorkflowEvaluator) evaluateSourceCondition(condition string) bool {
	source := w.context.Event
	if source == "" {
		source = "push" // Default
	}
	
	// Handle equality checks
	if strings.Contains(condition, "==") {
		// Extract the comparison value
		re := regexp.MustCompile(`\$CI_PIPELINE_SOURCE\s*==\s*"([^"]+)"`)
		matches := re.FindStringSubmatch(condition)
		if len(matches) > 1 {
			return source == matches[1]
		}
	}
	
	// Handle inequality checks
	if strings.Contains(condition, "!=") {
		re := regexp.MustCompile(`\$CI_PIPELINE_SOURCE\s*!=\s*"([^"]+)"`)
		matches := re.FindStringSubmatch(condition)
		if len(matches) > 1 {
			return source != matches[1]
		}
	}
	
	return true
}

// evaluateBranchCondition evaluates conditions involving $CI_COMMIT_BRANCH
func (w *WorkflowEvaluator) evaluateBranchCondition(condition string) bool {
	branch := w.context.Branch
	if branch == "" {
		return false
	}
	
	// Handle equality checks
	if strings.Contains(condition, "==") {
		re := regexp.MustCompile(`\$CI_COMMIT_BRANCH\s*==\s*"([^"]+)"`)
		matches := re.FindStringSubmatch(condition)
		if len(matches) > 1 {
			return branch == matches[1]
		}
	}
	
	// Handle inequality checks  
	if strings.Contains(condition, "!=") {
		re := regexp.MustCompile(`\$CI_COMMIT_BRANCH\s*!=\s*"([^"]+)"`)
		matches := re.FindStringSubmatch(condition)
		if len(matches) > 1 {
			return branch != matches[1]
		}
	}
	
	return true
}

// evaluateMRCondition evaluates conditions involving merge request variables
func (w *WorkflowEvaluator) evaluateMRCondition(condition string) bool {
	// Simple implementation: check if this is a merge request pipeline
	if strings.Contains(condition, "$CI_MERGE_REQUEST_ID") {
		return w.context.IsMR
	}
	
	return true
}

// DefaultPipelineContext creates a default pipeline context for main branch push
func DefaultPipelineContext() *PipelineContext {
	return &PipelineContext{
		Branch:       "main",
		Variables:    map[string]string{},
		Event:        "push",
		IsMR:         false,
		IsMainBranch: true,
	}
}

// MergeRequestPipelineContext creates a pipeline context for merge request
func MergeRequestPipelineContext(sourceBranch string) *PipelineContext {
	return &PipelineContext{
		Branch:       sourceBranch,
		Variables:    map[string]string{},
		Event:        "merge_request_event",
		IsMR:         true,
		IsMainBranch: false,
	}
}