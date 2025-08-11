package parser

import (
	"testing"
)

func TestWorkflowParsing(t *testing.T) {
	yamlContent := `
stages:
  - test
  - deploy

workflow:
  rules:
    - if: $CI_PIPELINE_SOURCE == "push"
      when: always
    - if: $CI_MERGE_REQUEST_ID
      when: never

test-job:
  stage: test
  script:
    - echo "testing"

deploy-job:
  stage: deploy
  script:
    - echo "deploying"
  rules:
    - if: $CI_COMMIT_BRANCH == "main"
`

	config, err := Parse([]byte(yamlContent))
	if err != nil {
		t.Fatalf("Failed to parse YAML: %v", err)
	}

	// Test workflow parsing
	if config.Workflow == nil {
		t.Fatal("Workflow should be parsed")
	}

	if len(config.Workflow.Rules) != 2 {
		t.Fatalf("Expected 2 workflow rules, got %d", len(config.Workflow.Rules))
	}

	// Test first rule
	firstRule := config.Workflow.Rules[0]
	if firstRule.If != `$CI_PIPELINE_SOURCE == "push"` {
		t.Errorf("Expected if condition to be '$CI_PIPELINE_SOURCE == \"push\"', got '%s'", firstRule.If)
	}
	if firstRule.When != "always" {
		t.Errorf("Expected when to be 'always', got '%s'", firstRule.When)
	}

	// Test second rule
	secondRule := config.Workflow.Rules[1]
	if secondRule.If != "$CI_MERGE_REQUEST_ID" {
		t.Errorf("Expected if condition to be '$CI_MERGE_REQUEST_ID', got '%s'", secondRule.If)
	}
	if secondRule.When != "never" {
		t.Errorf("Expected when to be 'never', got '%s'", secondRule.When)
	}
}

func TestWorkflowEvaluator(t *testing.T) {
	tests := []struct {
		name       string
		workflow   *Workflow
		context    *PipelineContext
		shouldRun  bool
	}{
		{
			name: "No workflow - should create pipeline",
			workflow: nil,
			context: DefaultPipelineContext(),
			shouldRun: true,
		},
		{
			name: "Push event allowed",
			workflow: &Workflow{
				Rules: []Rule{
					{If: `$CI_PIPELINE_SOURCE == "push"`, When: "always"},
					{When: "never"},
				},
			},
			context: &PipelineContext{
				Event: "push",
				Branch: "main",
			},
			shouldRun: true,
		},
		{
			name: "MR event blocked",
			workflow: &Workflow{
				Rules: []Rule{
					{If: `$CI_PIPELINE_SOURCE == "push"`, When: "always"},
					{If: `$CI_MERGE_REQUEST_ID`, When: "never"},
				},
			},
			context: &PipelineContext{
				Event: "merge_request_event",
				IsMR: true,
			},
			shouldRun: false,
		},
		{
			name: "Main branch allowed",
			workflow: &Workflow{
				Rules: []Rule{
					{If: `$CI_COMMIT_BRANCH == "main"`, When: "always"},
					{When: "never"},
				},
			},
			context: &PipelineContext{
				Branch: "main",
				IsMainBranch: true,
			},
			shouldRun: true,
		},
		{
			name: "Feature branch blocked",
			workflow: &Workflow{
				Rules: []Rule{
					{If: `$CI_COMMIT_BRANCH == "main"`, When: "always"},
					{When: "never"},
				},
			},
			context: &PipelineContext{
				Branch: "feature-branch",
				IsMainBranch: false,
			},
			shouldRun: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &GitLabConfig{
				Workflow: tt.workflow,
			}
			evaluator := NewWorkflowEvaluator(config, tt.context)
			result := evaluator.ShouldCreatePipeline()
			
			if result != tt.shouldRun {
				t.Errorf("Expected pipeline creation %v, got %v", tt.shouldRun, result)
			}
		})
	}
}

func TestPipelineSimulation(t *testing.T) {
	yamlContent := `
stages:
  - test
  - deploy

workflow:
  rules:
    - if: $CI_PIPELINE_SOURCE == "push"
      when: always
    - if: $CI_MERGE_REQUEST_ID
      when: always

test-job:
  stage: test
  script:
    - echo "testing"

deploy-job:
  stage: deploy
  script:
    - echo "deploying"
  rules:
    - if: $CI_COMMIT_BRANCH == "main"

mr-only-job:
  stage: test
  script:
    - echo "mr testing"
  rules:
    - if: $CI_MERGE_REQUEST_ID
`

	config, err := Parse([]byte(yamlContent))
	if err != nil {
		t.Fatalf("Failed to parse YAML: %v", err)
	}

	// Test main branch simulation
	mainJobs := config.SimulateMainBranchPipeline()
	
	if !mainJobs["test-job"] {
		t.Error("test-job should run on main branch")
	}
	if !mainJobs["deploy-job"] {
		t.Error("deploy-job should run on main branch")
	}
	if mainJobs["mr-only-job"] {
		t.Error("mr-only-job should not run on main branch")
	}

	// Test MR simulation
	mrJobs := config.SimulateMergeRequestPipeline("feature-branch")
	
	if !mrJobs["test-job"] {
		t.Error("test-job should run in MR pipeline")
	}
	if mrJobs["deploy-job"] {
		t.Error("deploy-job should not run in MR pipeline")
	}
	if !mrJobs["mr-only-job"] {
		t.Error("mr-only-job should run in MR pipeline")
	}
}

func TestJobRulesEvaluation(t *testing.T) {
	config := &GitLabConfig{
		Jobs: map[string]*JobConfig{
			"always-job": {
				Rules: []Rule{
					{When: "always"},
				},
			},
			"never-job": {
				Rules: []Rule{
					{When: "never"},
				},
			},
			"branch-job": {
				Rules: []Rule{
					{If: `$CI_COMMIT_BRANCH == "main"`},
				},
			},
			"mr-job": {
				Rules: []Rule{
					{If: `$CI_MERGE_REQUEST_ID`},
				},
			},
			"no-rules-job": {},
		},
	}

	mainContext := DefaultPipelineContext()
	mrContext := MergeRequestPipelineContext("feature")

	// Test main branch context
	if !config.shouldJobRun(config.Jobs["always-job"], mainContext) {
		t.Error("always-job should run on main branch")
	}
	if config.shouldJobRun(config.Jobs["never-job"], mainContext) {
		t.Error("never-job should not run on main branch")
	}
	if !config.shouldJobRun(config.Jobs["branch-job"], mainContext) {
		t.Error("branch-job should run on main branch")
	}
	if config.shouldJobRun(config.Jobs["mr-job"], mainContext) {
		t.Error("mr-job should not run on main branch")
	}
	if !config.shouldJobRun(config.Jobs["no-rules-job"], mainContext) {
		t.Error("no-rules-job should run by default")
	}

	// Test MR context
	if !config.shouldJobRun(config.Jobs["always-job"], mrContext) {
		t.Error("always-job should run in MR")
	}
	if config.shouldJobRun(config.Jobs["never-job"], mrContext) {
		t.Error("never-job should not run in MR")
	}
	if config.shouldJobRun(config.Jobs["branch-job"], mrContext) {
		t.Error("branch-job should not run in MR")
	}
	if !config.shouldJobRun(config.Jobs["mr-job"], mrContext) {
		t.Error("mr-job should run in MR")
	}
}

func TestOnlyExceptEvaluation(t *testing.T) {
	config := &GitLabConfig{
		Jobs: map[string]*JobConfig{
			"main-only-job": {
				Only: "main",
			},
			"mr-only-job": {
				Only: "merge_requests",
			},
			"except-main-job": {
				Except: []interface{}{"main", "master"},
			},
		},
	}

	mainContext := DefaultPipelineContext()
	mrContext := MergeRequestPipelineContext("feature")

	// Test main-only job
	if !config.shouldJobRun(config.Jobs["main-only-job"], mainContext) {
		t.Error("main-only-job should run on main branch")
	}
	if config.shouldJobRun(config.Jobs["main-only-job"], mrContext) {
		t.Error("main-only-job should not run in MR")
	}

	// Test MR-only job
	if config.shouldJobRun(config.Jobs["mr-only-job"], mainContext) {
		t.Error("mr-only-job should not run on main branch")
	}
	if !config.shouldJobRun(config.Jobs["mr-only-job"], mrContext) {
		t.Error("mr-only-job should run in MR")
	}

	// Test except-main job
	if config.shouldJobRun(config.Jobs["except-main-job"], mainContext) {
		t.Error("except-main-job should not run on main branch")
	}
	if !config.shouldJobRun(config.Jobs["except-main-job"], mrContext) {
		t.Error("except-main-job should run in MR")
	}
}