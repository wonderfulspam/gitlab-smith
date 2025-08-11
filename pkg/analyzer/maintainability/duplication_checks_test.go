package maintainability

import (
	"strings"
	"testing"

	"github.com/wonderfulspam/gitlab-smith/pkg/analyzer/types"
	"github.com/wonderfulspam/gitlab-smith/pkg/parser"
)

func TestCheckDuplicatedCode(t *testing.T) {
	t.Run("Duplicated scripts", func(t *testing.T) {
		config := &parser.GitLabConfig{
			Jobs: map[string]*parser.JobConfig{
				"test1": {
					Stage:  "test",
					Script: []string{"npm test", "npm run coverage"},
				},
				"test2": {
					Stage:  "test",
					Script: []string{"npm test", "npm run coverage"},
				},
				"build": {
					Stage:  "build",
					Script: []string{"npm run build"},
				},
			},
		}

		issues := CheckDuplicatedCode(config)

		if len(issues) != 1 {
			t.Errorf("Expected 1 issue, got %d", len(issues))
		}

		issue := issues[0]
		if issue.Type != types.IssueTypeMaintainability {
			t.Errorf("Expected maintainability issue, got %s", issue.Type)
		}

		if !strings.Contains(issue.Message, "test1") || !strings.Contains(issue.Message, "test2") {
			t.Errorf("Expected message to contain both job names, got '%s'", issue.Message)
		}
	})
}

func TestCheckDuplicatedBeforeScripts(t *testing.T) {
	t.Run("Similar before_script blocks", func(t *testing.T) {
		config := &parser.GitLabConfig{
			Jobs: map[string]*parser.JobConfig{
				"build": {
					Stage: "build",
					BeforeScript: []string{
						"echo 'Starting build'",
						"apt-get update",
						"apt-get install -y git",
						"npm ci",
					},
				},
				"test": {
					Stage: "test",
					BeforeScript: []string{
						"echo 'Starting test'",
						"apt-get update",
						"apt-get install -y git",
						"npm ci",
					},
				},
			},
		}

		issues := CheckDuplicatedBeforeScripts(config)

		if len(issues) == 0 {
			t.Errorf("Expected at least 1 issue for similar before_scripts, got %d", len(issues))
		}

		foundSimilar := false
		for _, issue := range issues {
			if strings.Contains(issue.Message, "Similar before_script blocks") {
				foundSimilar = true
				break
			}
		}

		if !foundSimilar {
			t.Error("Expected similar before_script blocks issue")
		}
	})

	t.Run("Empty before_scripts", func(t *testing.T) {
		config := &parser.GitLabConfig{
			Jobs: map[string]*parser.JobConfig{
				"job1": {
					Stage:        "test",
					BeforeScript: []string{},
				},
				"job2": {
					Stage:        "test",
					BeforeScript: nil,
				},
			},
		}

		issues := CheckDuplicatedBeforeScripts(config)

		if len(issues) != 0 {
			t.Errorf("Expected 0 issues for empty before_scripts, got %d", len(issues))
		}
	})
}

func TestCheckDuplicatedCacheConfig(t *testing.T) {
	t.Run("Duplicate cache configurations", func(t *testing.T) {
		config := &parser.GitLabConfig{
			Jobs: map[string]*parser.JobConfig{
				"build": {
					Stage: "build",
					Cache: &parser.Cache{
						Key:   "$CI_COMMIT_REF_SLUG",
						Paths: []string{"node_modules/", ".npm/"},
					},
				},
				"test": {
					Stage: "test",
					Cache: &parser.Cache{
						Key:   "$CI_COMMIT_REF_SLUG",
						Paths: []string{"node_modules/", ".npm/"},
					},
				},
				"deploy": {
					Stage: "deploy",
					Cache: &parser.Cache{
						Key:   "$CI_COMMIT_REF_SLUG",
						Paths: []string{"node_modules/", ".npm/"},
					},
				},
			},
		}

		issues := CheckDuplicatedCacheConfig(config)

		if len(issues) != 1 {
			t.Errorf("Expected 1 issue for duplicate cache, got %d", len(issues))
		}

		if !strings.Contains(issues[0].Message, "Duplicate cache configuration") {
			t.Errorf("Expected duplicate cache message, got: %s", issues[0].Message)
		}
	})

	t.Run("No cache configurations", func(t *testing.T) {
		config := &parser.GitLabConfig{
			Jobs: map[string]*parser.JobConfig{
				"build": {
					Stage:  "build",
					Script: []string{"echo build"},
				},
				"test": {
					Stage:  "test",
					Script: []string{"echo test"},
				},
			},
		}

		issues := CheckDuplicatedCacheConfig(config)

		if len(issues) != 0 {
			t.Errorf("Expected 0 issues for no cache, got %d", len(issues))
		}
	})
}

func TestCheckDuplicatedImageConfig(t *testing.T) {
	t.Run("Duplicate image configurations", func(t *testing.T) {
		config := &parser.GitLabConfig{
			Jobs: map[string]*parser.JobConfig{
				"build": {
					Stage: "build",
					Image: "node:16",
				},
				"test": {
					Stage: "test",
					Image: "node:16",
				},
				"lint": {
					Stage: "test",
					Image: "node:16",
				},
			},
		}

		issues := CheckDuplicatedImageConfig(config)

		if len(issues) != 1 {
			t.Errorf("Expected 1 issue for duplicate image, got %d", len(issues))
		}

		if !strings.Contains(issues[0].Message, "Duplicate image configuration") {
			t.Errorf("Expected duplicate image message, got: %s", issues[0].Message)
		}
	})

	t.Run("Variable expansion prevents false negatives in duplication detection", func(t *testing.T) {
		config := &parser.GitLabConfig{
			Variables: map[string]interface{}{
				"PYTHON_IMAGE": "python:3.11",
			},
			Jobs: map[string]*parser.JobConfig{
				"test1": {Image: "${PYTHON_IMAGE}"},
				"test2": {Image: "${PYTHON_IMAGE}"},
				"test3": {Image: "${PYTHON_IMAGE}"},
			},
		}

		issues := CheckDuplicatedImageConfig(config)

		if len(issues) != 1 {
			t.Errorf("Expected 1 issue for duplicate expanded images, got %d", len(issues))
		}

		if len(issues) > 0 && !strings.Contains(issues[0].Message, "python:3.11") {
			t.Errorf("Expected expanded image name in message, got: %s", issues[0].Message)
		}
	})

	t.Run("Mixed variable and literal images are properly compared", func(t *testing.T) {
		config := &parser.GitLabConfig{
			Variables: map[string]interface{}{
				"NODE_IMAGE": "node:18",
			},
			Jobs: map[string]*parser.JobConfig{
				"job1": {Image: "${NODE_IMAGE}"},
				"job2": {Image: "node:18"}, // Same as expanded variable
				"job3": {Image: "${NODE_IMAGE}"},
			},
		}

		issues := CheckDuplicatedImageConfig(config)

		if len(issues) != 1 {
			t.Errorf("Expected 1 issue for mixed variable/literal duplicates, got %d", len(issues))
		}

		if len(issues) > 0 && !strings.Contains(issues[0].Message, "node:18") {
			t.Errorf("Expected resolved image name in message, got: %s", issues[0].Message)
		}
	})
}

func TestCheckDuplicatedSetup(t *testing.T) {
	t.Run("Duplicate setup commands", func(t *testing.T) {
		config := &parser.GitLabConfig{
			Jobs: map[string]*parser.JobConfig{
				"build": {
					Stage: "build",
					Script: []string{
						"npm ci --cache .npm",
						"npm run build",
					},
				},
				"test": {
					Stage: "test",
					Script: []string{
						"npm ci --cache .npm",
						"npm test",
					},
				},
			},
		}

		issues := CheckDuplicatedSetup(config)

		if len(issues) == 0 {
			t.Errorf("Expected at least 1 issue for duplicate setup, got %d", len(issues))
		}

		foundDuplicateSetup := false
		for _, issue := range issues {
			if strings.Contains(issue.Message, "Duplicate setup configuration") {
				foundDuplicateSetup = true
				break
			}
		}

		if !foundDuplicateSetup {
			t.Error("Expected duplicate setup configuration issue")
		}
	})
}

func TestNormalizeSetupCommand(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "apt-get install command",
			input:    "apt-get install -y git",
			expected: "apt-get-install",
		},
		{
			name:     "npm ci command",
			input:    "npm ci --cache .npm",
			expected: "npm-ci",
		},
		{
			name:     "npm cache clean command",
			input:    "npm cache clean --force",
			expected: "npm-cache-clean",
		},
		{
			name:     "docker login command",
			input:    "docker login -u user -p pass",
			expected: "docker-login",
		},
		{
			name:     "version check command",
			input:    "node --version",
			expected: "version-check",
		},
		{
			name:     "unmatched command",
			input:    "echo 'Hello world'",
			expected: "",
		},
		{
			name:     "kubectl with curl",
			input:    "curl -LO kubectl && chmod +x kubectl",
			expected: "kubectl-install",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeSetupCommand(tt.input)
			if result != tt.expected {
				t.Errorf("normalizeSetupCommand(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestIsSetupCommand(t *testing.T) {
	tests := []struct {
		name     string
		command  string
		expected bool
	}{
		{"npm ci", "npm ci", true},
		{"npm cache", "npm cache clean", true},
		{"pip install", "pip install requirements.txt", true},
		{"bundle install", "bundle install", true},
		{"composer install", "composer install", true},
		{"apt-get update", "apt-get update", true},
		{"apt-get install", "apt-get install -y git", true},
		{"apk add", "apk add git", true},
		{"yum install", "yum install git", true},
		{"echo command", "echo 'setup message'", true}, // echo is in setup keywords
		{"version check", "node --version", true},
		{"curl command", "curl -o file.txt", true},
		{"kubectl command", "kubectl apply", true},
		{"build command", "npm run build", false},
		{"test command", "npm test", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isSetupCommand(tt.command)
			if result != tt.expected {
				t.Errorf("isSetupCommand(%q) = %v, expected %v", tt.command, result, tt.expected)
			}
		})
	}
}

func TestContainsSetupCommands(t *testing.T) {
	t.Run("with setup commands", func(t *testing.T) {
		script := []string{
			"echo 'Starting'",
			"npm ci",
			"npm run build",
		}

		result := containsSetupCommands(script)
		if !result {
			t.Error("Expected true for script with setup commands")
		}
	})

	t.Run("without setup commands", func(t *testing.T) {
		script := []string{
			"echo 'Starting'",
			"npm run build",
			"npm test",
		}

		result := containsSetupCommands(script)
		if result {
			t.Error("Expected false for script without setup commands")
		}
	})

	t.Run("empty script", func(t *testing.T) {
		script := []string{}

		result := containsSetupCommands(script)
		if result {
			t.Error("Expected false for empty script")
		}
	})
}

func TestCreateSetupFingerprint(t *testing.T) {
	t.Run("consistent fingerprint", func(t *testing.T) {
		job1 := &parser.JobConfig{
			Script: []string{"npm ci", "echo 'setup done'"},
		}
		job2 := &parser.JobConfig{
			Script: []string{"npm ci --cache .npm", "echo 'setup complete'"},
		}

		fingerprint1 := createSetupFingerprint(job1)
		fingerprint2 := createSetupFingerprint(job2)

		// Should be the same because they have the same setup command pattern
		if fingerprint1 != fingerprint2 {
			t.Errorf("Expected same fingerprint for similar setup scripts, got %s vs %s", fingerprint1, fingerprint2)
		}
	})

	t.Run("different fingerprint", func(t *testing.T) {
		job1 := &parser.JobConfig{
			Script: []string{"npm ci"},
		}
		job2 := &parser.JobConfig{
			Script: []string{"yarn install"},
		}

		fingerprint1 := createSetupFingerprint(job1)
		fingerprint2 := createSetupFingerprint(job2)

		if fingerprint1 == fingerprint2 {
			t.Error("Expected different fingerprints for different setup commands")
		}
	})

	t.Run("empty script", func(t *testing.T) {
		job := &parser.JobConfig{
			Script: []string{},
		}
		fingerprint := createSetupFingerprint(job)

		// Empty script might result in empty fingerprint - that's valid
		t.Logf("Fingerprint for empty script: '%s'", fingerprint)
	})
}
