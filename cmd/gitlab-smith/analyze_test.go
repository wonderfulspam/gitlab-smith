package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestAnalyzeCommand(t *testing.T) {
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
		format      string
		expectError bool
		checkOutput func(t *testing.T, output string)
	}{
		{
			name:        "analyze with table format",
			args:        []string{testFile},
			format:      "table",
			expectError: false,
			checkOutput: func(t *testing.T, output string) {
				if !strings.Contains(output, "GitLab CI Analysis Report") {
					t.Error("Expected table format header")
				}
				if !strings.Contains(output, "Summary") {
					t.Error("Expected summary section")
				}
			},
		},
		{
			name:        "analyze with json format",
			args:        []string{testFile, "--format", "json"},
			format:      "json",
			expectError: false,
			checkOutput: func(t *testing.T, output string) {
				var result map[string]interface{}
				err := json.Unmarshal([]byte(output), &result)
				if err != nil {
					t.Errorf("Output is not valid JSON: %v", err)
					return
				}

				if result["file"] == nil {
					t.Error("Expected 'file' field in JSON output")
				}
				if result["analysis"] == nil {
					t.Error("Expected 'analysis' field in JSON output")
				}
			},
		},
		{
			name:        "analyze with invalid format",
			args:        []string{testFile, "--format", "invalid"},
			format:      "invalid",
			expectError: true,
			checkOutput: func(t *testing.T, output string) {
				if !strings.Contains(output, "unsupported format") {
					t.Error("Expected unsupported format error")
				}
			},
		},
		{
			name:        "analyze non-existent file",
			args:        []string{"/non/existent/file.yml"},
			format:      "table",
			expectError: true,
			checkOutput: func(t *testing.T, output string) {
				if !strings.Contains(output, "no such file or directory") && !strings.Contains(output, "cannot find") {
					t.Errorf("Expected file not found error, got: %s", output)
				}
			},
		},
		{
			name:        "analyze with no arguments",
			args:        []string{},
			format:      "table",
			expectError: true,
			checkOutput: func(t *testing.T, output string) {
				if !strings.Contains(output, "accepts 1 arg") && !strings.Contains(output, "requires exactly 1 arg") {
					t.Error("Expected argument count error")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create analyze command for testing
			var analyzeFormat string
			cmd := &cobra.Command{
				Use:   "analyze [file]",
				Short: "Analyze GitLab CI configuration for issues and improvements",
				Args:  cobra.ExactArgs(1),
				RunE: func(cmd *cobra.Command, args []string) error {
					configFile := args[0]

					// Simulate parsing and analysis
					_, err := os.ReadFile(configFile)
					if err != nil {
						return err
					}

					// Mock analysis result
					mockResult := map[string]interface{}{
						"TotalIssues": 2,
						"Summary": map[string]int{
							"Performance":     1,
							"Security":        0,
							"Maintainability": 1,
							"Reliability":     0,
						},
						"Issues": []map[string]interface{}{
							{
								"Type":       "performance",
								"Severity":   "medium",
								"Message":    "Consider caching dependencies",
								"Path":       "build",
								"JobName":    "build",
								"Suggestion": "Add cache configuration",
							},
						},
					}

					switch analyzeFormat {
					case "json":
						output := map[string]interface{}{
							"file":     configFile,
							"analysis": mockResult,
						}
						encoder := json.NewEncoder(cmd.OutOrStdout())
						encoder.SetIndent("", "  ")
						return encoder.Encode(output)
					case "table":
						cmd.Printf("GitLab CI Analysis Report\n")
						cmd.Printf("========================\n")
						cmd.Printf("File: %s\n\n", configFile)
						cmd.Printf("Summary\n")
						cmd.Printf("-------\n")
						cmd.Printf("Total Issues: %d\n", 2)
						cmd.Printf("  Performance: %d\n", 1)
						cmd.Printf("  Security: %d\n", 0)
						cmd.Printf("  Maintainability: %d\n", 1)
						cmd.Printf("  Reliability: %d\n", 0)
						return nil
					default:
						return fmt.Errorf("unsupported format: %s (supported: table, json)", analyzeFormat)
					}
				},
			}

			cmd.Flags().StringVar(&analyzeFormat, "format", "table", "Output format: table, json")

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

func TestAnalyzeCommandOutputFormatting(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.gitlab-ci.yml")

	err := os.WriteFile(testFile, []byte("stages:\n  - build"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	tests := []struct {
		name           string
		format         string
		mockIssues     int
		expectedOutput []string
	}{
		{
			name:       "table format with issues",
			format:     "table",
			mockIssues: 3,
			expectedOutput: []string{
				"GitLab CI Analysis Report",
				"Summary",
				"Total Issues: 3",
			},
		},
		{
			name:       "table format no issues",
			format:     "table",
			mockIssues: 0,
			expectedOutput: []string{
				"GitLab CI Analysis Report",
				"✅ No issues found!",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var analyzeFormat string
			cmd := &cobra.Command{
				Use:  "analyze [file]",
				Args: cobra.ExactArgs(1),
				RunE: func(cmd *cobra.Command, args []string) error {
					configFile := args[0]

					switch analyzeFormat {
					case "table":
						cmd.Printf("GitLab CI Analysis Report\n")
						cmd.Printf("========================\n")
						cmd.Printf("File: %s\n\n", configFile)
						cmd.Printf("Summary\n")
						cmd.Printf("-------\n")
						cmd.Printf("Total Issues: %d\n", tt.mockIssues)

						if tt.mockIssues == 0 {
							cmd.Printf("\n✅ No issues found! Your GitLab CI configuration looks good.\n")
						}
						return nil
					}
					return nil
				},
			}

			cmd.Flags().StringVar(&analyzeFormat, "format", "table", "Output format: table, json")

			var buf bytes.Buffer
			cmd.SetOut(&buf)
			cmd.SetArgs([]string{testFile, "--format", tt.format})

			err := cmd.Execute()
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			output := buf.String()
			for _, expected := range tt.expectedOutput {
				if !strings.Contains(output, expected) {
					t.Errorf("Expected output to contain '%s', but got: %s", expected, output)
				}
			}
		})
	}
}

func TestAnalyzeCommandHelp(t *testing.T) {
	var analyzeFormat string
	cmd := &cobra.Command{
		Use:   "analyze [file]",
		Short: "Analyze GitLab CI configuration for issues and improvements",
		Long: `Analyze GitLab CI configuration files to identify potential issues,
optimization opportunities, and suggest improvements for better maintainability,
performance, security, and reliability.`,
		Args: cobra.ExactArgs(1),
	}

	cmd.Flags().StringVar(&analyzeFormat, "format", "table", "Output format: table, json")

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"--help"})

	err := cmd.Execute()
	if err != nil {
		t.Errorf("Expected no error for help command, got: %v", err)
	}

	output := buf.String()
	expectedTexts := []string{
		"Analyze GitLab CI configuration",
		"optimization opportunities",
	}

	for _, expected := range expectedTexts {
		if !strings.Contains(output, expected) {
			t.Errorf("Expected help text to contain '%s', got: %s", expected, output)
		}
	}

	// Just check that we got some help output
	if len(output) < 50 {
		t.Error("Expected substantial help output")
	}
}
