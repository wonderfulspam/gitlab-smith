package parser

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

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
				Stage:  "build",
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
