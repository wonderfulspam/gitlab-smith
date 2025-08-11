package parser

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestParse(t *testing.T) {
	data, err := os.ReadFile("../../test/fixtures/simple.gitlab-ci.yml")
	if err != nil {
		t.Fatalf("reading test fixture: %v", err)
	}

	config, err := Parse(data)
	if err != nil {
		t.Fatalf("parsing config: %v", err)
	}

	if len(config.Stages) != 3 {
		t.Errorf("expected 3 stages, got %d", len(config.Stages))
	}

	expectedStages := []string{"build", "test", "deploy"}
	for i, stage := range expectedStages {
		if i >= len(config.Stages) || config.Stages[i] != stage {
			t.Errorf("expected stage %d to be %s, got %v", i, stage, config.Stages[i])
		}
	}

	if len(config.Variables) != 2 {
		t.Errorf("expected 2 variables, got %d", len(config.Variables))
	}

	if config.Default == nil {
		t.Error("expected default config to be set")
	} else {
		if config.Default.Image != "node:16" {
			t.Errorf("expected default image to be node:16, got %s", config.Default.Image)
		}
	}

	if len(config.Jobs) != 5 {
		t.Errorf("expected 5 jobs, got %d", len(config.Jobs))
	}

	buildJob, exists := config.Jobs["build"]
	if !exists {
		t.Error("expected 'build' job to exist")
	} else {
		if buildJob.Stage != "build" {
			t.Errorf("expected build job stage to be 'build', got %s", buildJob.Stage)
		}
		if buildJob.Artifacts == nil || len(buildJob.Artifacts.Paths) != 1 {
			t.Error("expected build job to have artifacts with 1 path")
		}
	}

	testUnitJob, exists := config.Jobs["test:unit"]
	if !exists {
		t.Error("expected 'test:unit' job to exist")
	} else {
		if testUnitJob.Needs == nil {
			t.Error("expected test:unit to have needs")
		} else if needsSlice, ok := testUnitJob.Needs.([]interface{}); ok {
			if len(needsSlice) != 1 || needsSlice[0] != "build" {
				t.Errorf("expected test:unit to need 'build', got %v", needsSlice)
			}
		} else {
			t.Errorf("unexpected needs type: %T", testUnitJob.Needs)
		}
	}

	deployProdJob, exists := config.Jobs["deploy:production"]
	if !exists {
		t.Error("expected 'deploy:production' job to exist")
	} else {
		if deployProdJob.When != "manual" {
			t.Errorf("expected deploy:production when to be 'manual', got %s", deployProdJob.When)
		}
		if len(deployProdJob.Rules) != 1 {
			t.Errorf("expected deploy:production to have 1 rule, got %d", len(deployProdJob.Rules))
		}
	}
}

func TestGetDependencyGraph(t *testing.T) {
	data, err := os.ReadFile("../../test/fixtures/simple.gitlab-ci.yml")
	if err != nil {
		t.Fatalf("reading test fixture: %v", err)
	}

	config, err := Parse(data)
	if err != nil {
		t.Fatalf("parsing config: %v", err)
	}

	graph := config.GetDependencyGraph()

	buildDeps := graph["build"]
	if len(buildDeps) != 0 {
		t.Errorf("expected build job to have no dependencies, got %v", buildDeps)
	}

	testUnitDeps := graph["test:unit"]
	if len(testUnitDeps) != 1 || testUnitDeps[0] != "build" {
		t.Errorf("expected test:unit to depend on 'build', got %v", testUnitDeps)
	}

	testIntegrationDeps := graph["test:integration"]
	if len(testIntegrationDeps) != 1 || testIntegrationDeps[0] != "build" {
		t.Errorf("expected test:integration to depend on 'build', got %v", testIntegrationDeps)
	}

	deployStagingDeps := graph["deploy:staging"]
	if len(deployStagingDeps) != 2 {
		t.Errorf("expected deploy:staging to have 2 dependencies, got %v", deployStagingDeps)
	}

	deployProdDeps := graph["deploy:production"]
	if len(deployProdDeps) != 2 {
		t.Errorf("expected deploy:production to have 2 dependencies, got %v", deployProdDeps)
	}
}

func TestParseMinimal(t *testing.T) {
	yaml := `
test:
  script:
    - echo "Hello, World!"
`
	config, err := Parse([]byte(yaml))
	if err != nil {
		t.Fatalf("parsing minimal config: %v", err)
	}

	if len(config.Jobs) != 1 {
		t.Errorf("expected 1 job, got %d", len(config.Jobs))
	}

	testJob, exists := config.Jobs["test"]
	if !exists {
		t.Error("expected 'test' job to exist")
	} else {
		if len(testJob.Script) != 1 || testJob.Script[0] != "echo \"Hello, World!\"" {
			t.Errorf("unexpected script: %v", testJob.Script)
		}
	}
}

func TestParseComplexNeeds(t *testing.T) {
	yaml := `
build:
  script:
    - make build

test:
  script:
    - make test
  needs:
    - job: build
      optional: true

deploy:
  script:
    - make deploy
  needs:
    - build
    - test
`
	config, err := Parse([]byte(yaml))
	if err != nil {
		t.Fatalf("parsing config with complex needs: %v", err)
	}

	testJob := config.Jobs["test"]
	if needsSlice, ok := testJob.Needs.([]interface{}); ok {
		if len(needsSlice) != 1 {
			t.Errorf("expected test job to have 1 need, got %d", len(needsSlice))
		}
		if needMap, ok := needsSlice[0].(map[string]interface{}); ok {
			if needMap["job"] != "build" || needMap["optional"] != true {
				t.Errorf("unexpected needs for test job: %+v", needMap)
			}
		}
	} else {
		t.Errorf("unexpected needs type: %T", testJob.Needs)
	}

	deployJob := config.Jobs["deploy"]
	if needsSlice, ok := deployJob.Needs.([]interface{}); ok {
		if len(needsSlice) != 2 {
			t.Errorf("expected deploy job to have 2 needs, got %d", len(needsSlice))
		}
	}
}

func TestParseInvalidYAML(t *testing.T) {
	yaml := `
invalid: yaml: content:
  - with: malformed
    structure
`
	_, err := Parse([]byte(yaml))
	if err == nil {
		t.Error("expected error for invalid YAML, got nil")
	}
}

func TestParseEmptyConfig(t *testing.T) {
	yaml := ``
	config, err := Parse([]byte(yaml))
	if err != nil {
		t.Fatalf("parsing empty config: %v", err)
	}

	if config == nil {
		t.Error("expected config to be non-nil for empty YAML")
	}

	if len(config.Jobs) != 0 {
		t.Errorf("expected 0 jobs for empty config, got %d", len(config.Jobs))
	}
}

func TestParseVariablesTypes(t *testing.T) {
	yaml := `
variables:
  STRING_VAR: "hello"
  NUMBER_VAR: 42
  BOOLEAN_VAR: true
  NULL_VAR: null
  
test:
  script:
    - echo "test"
  variables:
    LOCAL_VAR: "local value"
    OVERRIDE_VAR: 123
`
	config, err := Parse([]byte(yaml))
	if err != nil {
		t.Fatalf("parsing config with variable types: %v", err)
	}

	if config.Variables["STRING_VAR"] != "hello" {
		t.Errorf("expected STRING_VAR to be 'hello', got %v", config.Variables["STRING_VAR"])
	}

	if config.Variables["NUMBER_VAR"] != 42 {
		t.Errorf("expected NUMBER_VAR to be 42, got %v", config.Variables["NUMBER_VAR"])
	}

	if config.Variables["BOOLEAN_VAR"] != true {
		t.Errorf("expected BOOLEAN_VAR to be true, got %v", config.Variables["BOOLEAN_VAR"])
	}

	if config.Variables["NULL_VAR"] != nil {
		t.Errorf("expected NULL_VAR to be nil, got %v", config.Variables["NULL_VAR"])
	}

	testJob := config.Jobs["test"]
	if testJob.Variables["LOCAL_VAR"] != "local value" {
		t.Errorf("expected LOCAL_VAR to be 'local value', got %v", testJob.Variables["LOCAL_VAR"])
	}
}

func TestParseJobsWithSpecialNames(t *testing.T) {
	yaml := `
"job with spaces":
  script:
    - echo "spaces"

job-with-dashes:
  script:
    - echo "dashes"

job_with_underscores:
  script:
    - echo "underscores"

"job:with:colons":
  script:
    - echo "colons"

"123numeric-start":
  script:
    - echo "numeric"

".hidden-job":
  script:
    - echo "hidden"
`
	config, err := Parse([]byte(yaml))
	if err != nil {
		t.Fatalf("parsing config with special job names: %v", err)
	}

	expectedJobs := []string{
		"job with spaces",
		"job-with-dashes",
		"job_with_underscores",
		"job:with:colons",
		"123numeric-start",
		// Note: ".hidden-job" might be filtered out by GitLab CI parser
	}

	if len(config.Jobs) < len(expectedJobs) {
		t.Errorf("expected at least %d jobs, got %d", len(expectedJobs), len(config.Jobs))
	}

	for _, jobName := range expectedJobs {
		if _, exists := config.Jobs[jobName]; !exists {
			t.Errorf("expected job '%s' to exist", jobName)
		}
	}
}

func TestIncludeResolver_RemoteInclude(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/common.yml" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`
stages:
  - build
  - test

common_job:
  stage: build
  script:
    - echo "common job"
`))
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	resolver := NewIncludeResolver("", "")
	
	// Test successful remote include
	data, err := resolver.resolveRemoteInclude(server.URL + "/common.yml")
	if err != nil {
		t.Fatalf("resolveRemoteInclude failed: %v", err)
	}

	if len(data) == 0 {
		t.Error("expected remote include data to be non-empty")
	}

	// Test that result is cached
	data2, err := resolver.resolveRemoteInclude(server.URL + "/common.yml")
	if err != nil {
		t.Fatalf("cached resolveRemoteInclude failed: %v", err)
	}

	if string(data) != string(data2) {
		t.Error("cached result should be identical")
	}

	// Test 404 error
	_, err = resolver.resolveRemoteInclude(server.URL + "/nonexistent.yml")
	if err == nil {
		t.Error("expected error for 404 response")
	}
}

func TestIncludeResolver_TemplateInclude(t *testing.T) {
	// Create a test server to mock GitLab.com
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/Android.yml" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`
# Android template
stages:
  - build
  - test

android_build:
  stage: build
  script:
    - ./gradlew build
`))
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	resolver := NewIncludeResolver("", "")
	
	// Mock the template resolution by directly testing the remote include functionality
	data, err := resolver.resolveRemoteInclude(server.URL + "/Android.yml")
	if err != nil {
		t.Fatalf("template include failed: %v", err)
	}

	if len(data) == 0 {
		t.Error("expected template data to be non-empty")
	}

	// Verify the content contains expected template content
	content := string(data)
	if !containsAll(content, "android_build", "gradlew") {
		t.Error("template content should contain Android-specific elements")
	}
}

func TestIncludeResolver_ProjectInclude(t *testing.T) {
	// Create a test server to mock GitLab API
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Mock GitLab API endpoint for file content
		// The HTTP server automatically decodes URL paths, so we check the decoded version
		if r.URL.Path == "/projects/group/project/repository/files/.gitlab-ci.yml/raw" && r.URL.RawQuery == "ref=main" {
			auth := r.Header.Get("Authorization")
			if auth != "Bearer test-token" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`
shared_job:
  stage: deploy
  script:
    - deploy_script.sh
`))
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	resolver := NewIncludeResolver(server.URL, "test-token")
	
	// Test successful project include
	data, err := resolver.resolveProjectInclude("group/project", ".gitlab-ci.yml", "main")
	if err != nil {
		t.Fatalf("resolveProjectInclude failed: %v", err)
	}

	if len(data) == 0 {
		t.Error("expected project include data to be non-empty")
	}

	// Verify the content
	content := string(data)
	if !containsAll(content, "shared_job", "deploy_script.sh") {
		t.Error("project include content should contain expected job")
	}

	// Test without API URL configured
	resolverNoAPI := NewIncludeResolver("", "")
	_, err = resolverNoAPI.resolveProjectInclude("group/project", ".gitlab-ci.yml", "main")
	if err == nil {
		t.Error("expected error when GitLab API URL not configured")
	}
}

func TestResolveIncludesWithResolver(t *testing.T) {
	// Create a test server for remote includes
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/shared.yml" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`
shared_variables:
  SHARED_VAR: "shared_value"

shared_job:
  stage: shared
  script:
    - echo "shared job"
`))
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	// Create test configuration with remote include
	yaml := `
stages:
  - build
  - shared

include:
  - remote: ` + server.URL + `/shared.yml

main_job:
  stage: build
  script:
    - echo "main job"
`

	config, err := Parse([]byte(yaml))
	if err != nil {
		t.Fatalf("parsing config with remote include: %v", err)
	}

	resolver := NewIncludeResolver("", "")
	err = ResolveIncludesWithResolver(config, "/tmp", resolver)
	if err != nil {
		t.Fatalf("resolving includes failed: %v", err)
	}

	// Check that shared job was merged
	if _, exists := config.Jobs["shared_job"]; !exists {
		t.Error("expected shared_job to be merged from remote include")
	}

	// Check that main job still exists
	if _, exists := config.Jobs["main_job"]; !exists {
		t.Error("expected main_job to be preserved")
	}

	// Should have 2 jobs total
	if len(config.Jobs) != 2 {
		t.Errorf("expected 2 jobs after include resolution, got %d", len(config.Jobs))
	}
}

func TestIncludeResolver_MergeIncludedData(t *testing.T) {
	// Create base config
	config := &GitLabConfig{
		Stages: []string{"build"},
		Variables: map[string]interface{}{
			"BASE_VAR": "base_value",
		},
		Jobs: map[string]*JobConfig{
			"base_job": {
				Stage: "build",
				Script: []string{"echo base"},
			},
		},
	}

	// Create included data
	includedData := []byte(`
stages:
  - build
  - test

variables:
  INCLUDED_VAR: "included_value"

included_job:
  stage: test
  script:
    - echo "included"
`)

	resolver := NewIncludeResolver("", "")
	err := resolver.mergeIncludedData(config, includedData, "/tmp")
	if err != nil {
		t.Fatalf("mergeIncludedData failed: %v", err)
	}

	// Check that included job was added
	if _, exists := config.Jobs["included_job"]; !exists {
		t.Error("expected included_job to be merged")
	}

	// Check that base job still exists
	if _, exists := config.Jobs["base_job"]; !exists {
		t.Error("expected base_job to be preserved")
	}

	// Check variables (base variables should be preserved, included variables should be added only if base variables is nil)
	if config.Variables["BASE_VAR"] != "base_value" {
		t.Error("expected BASE_VAR to be preserved")
	}

	// Since base config already has variables, included variables shouldn't override
	if config.Variables["INCLUDED_VAR"] != nil {
		t.Error("included variables should not override existing variables")
	}
}

// Helper function to check if string contains all substrings
func containsAll(s string, substrs ...string) bool {
	for _, substr := range substrs {
		if !containsSubstring(s, substr) {
			return false
		}
	}
	return true
}

func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && findSubstring(s, substr) != -1
}

func findSubstring(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
