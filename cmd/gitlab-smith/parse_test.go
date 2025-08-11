package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/wonderfulspam/gitlab-smith/pkg/parser"
)

func TestParseCommand(t *testing.T) {
	// Create a temporary test file
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.gitlab-ci.yml")

	testContent := `
stages:
  - build
  - test

variables:
  NODE_VERSION: "16"

build:
  stage: build
  script:
    - echo "Building application"
  artifacts:
    paths:
      - dist/

test:
  stage: test
  script:
    - echo "Running tests"
  dependencies:
    - build
`

	err := os.WriteFile(testFile, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	tests := []struct {
		name        string
		args        []string
		expectError bool
		checkOutput func(t *testing.T, output string)
	}{
		{
			name:        "valid GitLab CI file",
			args:        []string{testFile},
			expectError: false,
			checkOutput: func(t *testing.T, output string) {
				t.Logf("Parse output: %s", output)

				// Verify JSON output
				var result map[string]interface{}
				err := json.Unmarshal([]byte(output), &result)
				if err != nil {
					t.Errorf("Output is not valid JSON: %v\nOutput was: %s", err, output)
					return
				}

				// Check for expected structure - jobs should be populated
				if result["jobs"] == nil {
					t.Error("Expected 'jobs' in parsed output")
				}
			},
		},
		{
			name:        "non-existent file",
			args:        []string{"/non/existent/file.yml"},
			expectError: true,
			checkOutput: func(t *testing.T, output string) {
				if !strings.Contains(output, "no such file or directory") && !strings.Contains(output, "cannot find") {
					t.Errorf("Expected file not found error message, got: %s", output)
				}
			},
		},
		{
			name:        "no arguments",
			args:        []string{},
			expectError: true,
			checkOutput: func(t *testing.T, output string) {
				if !strings.Contains(output, "accepts 1 arg") && !strings.Contains(output, "requires exactly 1 arg") {
					t.Error("Expected argument count error message")
				}
			},
		},
		{
			name:        "too many arguments",
			args:        []string{testFile, "extra-arg"},
			expectError: true,
			checkOutput: func(t *testing.T, output string) {
				if !strings.Contains(output, "accepts 1 arg") && !strings.Contains(output, "requires exactly 1 arg") {
					t.Error("Expected argument count error message")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a fresh command instance for each test
			cmd := &cobra.Command{
				Use:   "parse <file>",
				Short: "Parse and display a GitLab CI configuration file",
				Args:  cobra.ExactArgs(1),
				RunE:  parseCmd.RunE,
			}

			var buf bytes.Buffer
			cmd.SetOut(&buf)
			cmd.SetErr(&buf)
			cmd.SetArgs(tt.args)

			err := cmd.Execute()
			output := buf.String()

			if tt.expectError && err == nil {
				t.Errorf("Expected error, but got none. Output: %s", output)
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error, but got: %v. Output: %s", err, output)
			}

			if tt.checkOutput != nil {
				tt.checkOutput(t, output)
			}
		})
	}
}

func TestParseCommandInvalidYAML(t *testing.T) {
	// Create a temporary test file with invalid YAML
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "invalid.gitlab-ci.yml")

	invalidContent := `
stages:
  - build
  - test
invalid_yaml: [unclosed_bracket
`

	err := os.WriteFile(testFile, []byte(invalidContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	cmd := &cobra.Command{
		Use:   "parse <file>",
		Short: "Parse and display a GitLab CI configuration file",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			filename := args[0]

			data, err := os.ReadFile(filename)
			if err != nil {
				return err
			}

			// Try to parse with actual parser to get real parsing error
			_, err = parser.Parse(data)
			return err
		},
	}

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{testFile})

	err = cmd.Execute()
	if err == nil {
		t.Error("Expected parsing error for invalid YAML, got none")
	}

	output := buf.String()
	if output != "" {
		t.Logf("Error output: %s", output)
	}
}

func TestParseCommandHelp(t *testing.T) {
	cmd := &cobra.Command{
		Use:   "parse <file>",
		Short: "Parse and display a GitLab CI configuration file",
		Args:  cobra.ExactArgs(1),
	}

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"--help"})

	err := cmd.Execute()
	if err != nil {
		t.Errorf("Expected no error for help command, got: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Parse and display a GitLab CI configuration file") {
		t.Error("Expected help text not found in output")
	}
}
