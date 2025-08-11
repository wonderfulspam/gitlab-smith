package varexpand

import (
	"testing"

	"github.com/wonderfulspam/gitlab-smith/pkg/parser"
)

func TestExpander_ExpandString(t *testing.T) {
	tests := []struct {
		name     string
		config   *parser.GitLabConfig
		input    string
		jobVars  map[string]interface{}
		expected string
	}{
		{
			name: "simple global variable",
			config: &parser.GitLabConfig{
				Variables: map[string]interface{}{
					"NODE_IMAGE": "node:22",
				},
			},
			input:    "${NODE_IMAGE}",
			jobVars:  nil,
			expected: "node:22",
		},
		{
			name: "variable with latest tag",
			config: &parser.GitLabConfig{
				Variables: map[string]interface{}{
					"BASE_IMAGE": "ubuntu:latest",
				},
			},
			input:    "$BASE_IMAGE",
			jobVars:  nil,
			expected: "ubuntu:latest",
		},
		{
			name: "job variable overrides global",
			config: &parser.GitLabConfig{
				Variables: map[string]interface{}{
					"VERSION": "18",
				},
			},
			input: "node:${VERSION}",
			jobVars: map[string]interface{}{
				"VERSION": "20",
			},
			expected: "node:20",
		},
		{
			name: "predefined GitLab CI variable",
			config: &parser.GitLabConfig{
				Variables: map[string]interface{}{},
			},
			input:    "${CI_REGISTRY_IMAGE}:latest",
			jobVars:  nil,
			expected: "registry.gitlab.com/group/project:latest",
		},
		{
			name: "mixed variables and text",
			config: &parser.GitLabConfig{
				Variables: map[string]interface{}{
					"REGISTRY": "my.registry.com",
					"PROJECT":  "my-app",
				},
			},
			input:    "${REGISTRY}/${PROJECT}:v1.0",
			jobVars:  nil,
			expected: "my.registry.com/my-app:v1.0",
		},
		{
			name: "non-string variables converted",
			config: &parser.GitLabConfig{
				Variables: map[string]interface{}{
					"VERSION": 22,
					"DEBUG":   true,
				},
			},
			input:    "node:${VERSION}-${DEBUG}",
			jobVars:  nil,
			expected: "node:22-true",
		},
		{
			name: "unresolved variable left as-is",
			config: &parser.GitLabConfig{
				Variables: map[string]interface{}{},
			},
			input:    "${UNKNOWN_VAR}:latest",
			jobVars:  nil,
			expected: "${UNKNOWN_VAR}:latest",
		},
		{
			name: "no variables in string",
			config: &parser.GitLabConfig{
				Variables: map[string]interface{}{},
			},
			input:    "alpine:3.18",
			jobVars:  nil,
			expected: "alpine:3.18",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expander := New(tt.config)
			result := expander.ExpandString(tt.input, tt.jobVars)
			if result != tt.expected {
				t.Errorf("ExpandString() = %q, expected %q", result, tt.expected)
			}
		})
	}
}

func TestExpander_HasUnresolvedVariables(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "no variables",
			input:    "node:22",
			expected: false,
		},
		{
			name:     "has unresolved variable",
			input:    "${UNKNOWN}:latest",
			expected: true,
		},
		{
			name:     "has resolved variable - no dollar sign",
			input:    "node:22",
			expected: false,
		},
		{
			name:     "multiple unresolved variables",
			input:    "${VAR1}/${VAR2}",
			expected: true,
		},
		{
			name:     "escaped dollar sign",
			input:    "echo $PATH",
			expected: true, // Still contains $ even if it's meant to be shell variable
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expander := New(&parser.GitLabConfig{})
			result := expander.HasUnresolvedVariables(tt.input)
			if result != tt.expected {
				t.Errorf("HasUnresolvedVariables() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestExpander_IntegrationWithRealWorldExamples(t *testing.T) {
	// Test cases that demonstrate the real improvements we made
	config := &parser.GitLabConfig{
		Variables: map[string]interface{}{
			"NODE_IMAGE": "node:22",        // Should pass image tag check
			"BAD_IMAGE":  "node:latest",    // Should fail image tag check
			"UNTAGGED":   "alpine",         // Should fail image tag check
			"CACHE_KEY":  "node-modules",   // For cache duplication
			"CACHE_PATH": "./node_modules", // For cache duplication
		},
		Jobs: make(map[string]*parser.JobConfig),
	}

	expander := New(config)

	tests := []struct {
		name     string
		input    string
		jobVars  map[string]interface{}
		expected string
		passes   string // What check this would pass/fail
	}{
		{
			name:     "properly tagged image should expand and pass security check",
			input:    "${NODE_IMAGE}",
			jobVars:  nil,
			expected: "node:22",
			passes:   "security image tag check",
		},
		{
			name:     "latest tagged image should expand and fail security check",
			input:    "${BAD_IMAGE}",
			jobVars:  nil,
			expected: "node:latest",
			passes:   "security image tag check (should flag :latest)",
		},
		{
			name:     "untagged image should expand and fail security check",
			input:    "${UNTAGGED}",
			jobVars:  nil,
			expected: "alpine",
			passes:   "security image tag check (should flag missing tag)",
		},
		{
			name:     "cache path with variables should be properly compared for duplication",
			input:    "${CACHE_PATH}",
			jobVars:  nil,
			expected: "./node_modules",
			passes:   "maintainability cache duplication check",
		},
		{
			name:     "job variable overrides global for accurate analysis",
			input:    "${NODE_IMAGE}",
			jobVars:  map[string]interface{}{"NODE_IMAGE": "node:18"},
			expected: "node:18",
			passes:   "job-level variable precedence",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := expander.ExpandString(tt.input, tt.jobVars)
			if result != tt.expected {
				t.Errorf("ExpandString() = %q, expected %q (for %s)", result, tt.expected, tt.passes)
			}

			// Verify that fully resolved variables don't have unresolved variables
			if !expander.HasUnresolvedVariables(result) && result == tt.expected {
				t.Logf("✓ Variable fully resolved for %s: %s -> %s", tt.passes, tt.input, result)
			}
		})
	}
}
