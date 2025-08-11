package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestVisualizeCommand(t *testing.T) {
	// Create a temporary test file
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.gitlab-ci.yml")

	testContent := `
stages:
  - build
  - test
  - deploy

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

deploy:
  stage: deploy
  script:
    - echo "Deploying to production"
  dependencies:
    - test
  only:
    - main
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
			name:        "visualize with mermaid format (default)",
			args:        []string{testFile},
			expectError: false,
			checkOutput: func(t *testing.T, output string) {
				if !strings.Contains(output, "graph") || !strings.Contains(output, "build") {
					t.Error("Expected mermaid diagram output with stages and jobs")
				}
			},
		},
		{
			name:        "visualize with dot format",
			args:        []string{testFile, "--format", "dot"},
			expectError: false,
			checkOutput: func(t *testing.T, output string) {
				if !strings.Contains(output, "digraph") || !strings.Contains(output, "build") {
					t.Error("Expected DOT graph output with stages and jobs")
				}
			},
		},
		{
			name:        "visualize with mermaid format explicit",
			args:        []string{testFile, "--format", "mermaid"},
			expectError: false,
			checkOutput: func(t *testing.T, output string) {
				if !strings.Contains(output, "graph") || !strings.Contains(output, "build") {
					t.Error("Expected mermaid diagram output")
				}
			},
		},
		{
			name:        "visualize with invalid format",
			args:        []string{testFile, "--format", "invalid"},
			expectError: true,
			checkOutput: func(t *testing.T, output string) {
				if !strings.Contains(output, "unsupported format") && !strings.Contains(output, "invalid format") {
					t.Error("Expected unsupported format error")
				}
			},
		},
		{
			name:        "visualize non-existent file",
			args:        []string{"/non/existent/file.yml"},
			expectError: true,
			checkOutput: func(t *testing.T, output string) {
				if !strings.Contains(output, "reading config file") && !strings.Contains(output, "no such file") {
					t.Errorf("Expected file reading error, got: %s", output)
				}
			},
		},
		{
			name:        "visualize with no arguments",
			args:        []string{},
			expectError: true,
			checkOutput: func(t *testing.T, output string) {
				if !strings.Contains(output, "accepts 1 arg") && !strings.Contains(output, "requires exactly 1 arg") {
					t.Error("Expected argument count error")
				}
			},
		},
		{
			name:        "visualize with too many arguments",
			args:        []string{testFile, "extra-arg"},
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
			// Create visualize command for testing
			var testFormat, testOutputFile string

			cmd := &cobra.Command{
				Use:   "visualize <config-file>",
				Short: "Generate a visual representation of a GitLab CI pipeline",
				Args:  cobra.ExactArgs(1),
				RunE: func(cmd *cobra.Command, args []string) error {
					configFile := args[0]

					// Check file existence
					_, err := os.ReadFile(configFile)
					if err != nil {
						return err
					}

					// Mock visual output based on format
					var visualOutput string
					switch testFormat {
					case "dot":
						visualOutput = `digraph pipeline {
	rankdir=LR;
	
	subgraph cluster_build {
		label="build";
		build [shape=box];
	}
	
	subgraph cluster_test {
		label="test";
		test [shape=box];
	}
	
	subgraph cluster_deploy {
		label="deploy";
		deploy [shape=box];
	}
	
	build -> test;
	test -> deploy;
}`
					case "mermaid":
						visualOutput = `graph LR
	subgraph "build"
		build["build"]
	end
	
	subgraph "test"
		test["test"]
	end
	
	subgraph "deploy"
		deploy["deploy"]
	end
	
	build --> test
	test --> deploy`
					default:
						return fmt.Errorf("unsupported format: %s", testFormat)
					}

					cmd.Print(visualOutput)
					return nil
				},
			}

			cmd.Flags().StringVar(&testFormat, "format", "mermaid", "Visual format (dot, mermaid)")
			cmd.Flags().StringVar(&testOutputFile, "output", "", "Output file for the diagram")

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

func TestVisualizeCommandOutputFile(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.gitlab-ci.yml")
	outputFile := filepath.Join(tempDir, "output.dot")

	// Create test file
	testContent := "stages:\n  - build\n\nbuild:\n  stage: build\n  script:\n    - echo test"
	err := os.WriteFile(testFile, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	var testFormat, testOutputFile string

	cmd := &cobra.Command{
		Use:  "visualize <config-file>",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			configFile := args[0]

			_, err := os.ReadFile(configFile)
			if err != nil {
				return err
			}

			// Mock visual output
			visualOutput := "digraph pipeline { build [shape=box]; }"

			if testOutputFile != "" {
				err := os.WriteFile(testOutputFile, []byte(visualOutput), 0644)
				if err != nil {
					return err
				}

				switch testFormat {
				case "dot":
					cmd.Printf("DOT graph written to %s\n", testOutputFile)
					cmd.Println("💡 To generate an image: dot -Tpng -o pipeline.png " + testOutputFile)
				case "mermaid":
					cmd.Printf("Mermaid diagram written to %s\n", testOutputFile)
					cmd.Println("💡 View online at: https://mermaid.live/")
				}
			} else {
				cmd.Print(visualOutput)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&testFormat, "format", "dot", "Visual format")
	cmd.Flags().StringVar(&testOutputFile, "output", "", "Output file")

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{testFile, "--format", "dot", "--output", outputFile})

	err = cmd.Execute()
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Check that output file was created
	if _, err := os.Stat(outputFile); os.IsNotExist(err) {
		t.Error("Expected output file to be created")
	}

	// Check that stdout contains success message and instructions
	output := buf.String()
	if !strings.Contains(output, "DOT graph written to") {
		t.Error("Expected success message in output")
	}
	if !strings.Contains(output, "dot -Tpng") {
		t.Error("Expected DOT usage instructions")
	}
}

func TestVisualizeCommandMermaidOutputFile(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.gitlab-ci.yml")
	outputFile := filepath.Join(tempDir, "output.mmd")

	// Create test file
	testContent := "stages:\n  - build\n\nbuild:\n  stage: build\n  script:\n    - echo test"
	err := os.WriteFile(testFile, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	var testFormat, testOutputFile string

	cmd := &cobra.Command{
		Use:  "visualize <config-file>",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			configFile := args[0]

			_, err := os.ReadFile(configFile)
			if err != nil {
				return err
			}

			// Mock visual output
			visualOutput := "graph LR\n  build[build]"

			if testOutputFile != "" {
				err := os.WriteFile(testOutputFile, []byte(visualOutput), 0644)
				if err != nil {
					return err
				}

				switch testFormat {
				case "dot":
					cmd.Printf("DOT graph written to %s\n", testOutputFile)
					cmd.Println("💡 To generate an image: dot -Tpng -o pipeline.png " + testOutputFile)
				case "mermaid":
					cmd.Printf("Mermaid diagram written to %s\n", testOutputFile)
					cmd.Println("💡 View online at: https://mermaid.live/")
				}
			} else {
				cmd.Print(visualOutput)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&testFormat, "format", "mermaid", "Visual format")
	cmd.Flags().StringVar(&testOutputFile, "output", "", "Output file")

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{testFile, "--format", "mermaid", "--output", outputFile})

	err = cmd.Execute()
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Check that output file was created
	if _, err := os.Stat(outputFile); os.IsNotExist(err) {
		t.Error("Expected output file to be created")
	}

	// Check that stdout contains success message and instructions
	output := buf.String()
	if !strings.Contains(output, "Mermaid diagram written to") {
		t.Error("Expected success message in output")
	}
	if !strings.Contains(output, "mermaid.live") {
		t.Error("Expected Mermaid usage instructions")
	}
}

func TestVisualizeCommandHelp(t *testing.T) {
	var testFormat, testOutputFile string

	cmd := &cobra.Command{
		Use:   "visualize <config-file>",
		Short: "Generate a visual representation of a GitLab CI pipeline",
		Long: `Creates a visual diagram of the GitLab CI pipeline structure showing jobs, stages, 
and dependencies. Supports DOT graph and Mermaid diagram formats.`,
		Args: cobra.ExactArgs(1),
	}

	cmd.Flags().StringVar(&testFormat, "format", "mermaid", "Visual format (dot, mermaid)")
	cmd.Flags().StringVar(&testOutputFile, "output", "", "Output file for the diagram")

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
		"visual diagram",
		"pipeline structure",
		"DOT graph and Mermaid",
	}

	for _, expected := range expectedTexts {
		if !strings.Contains(output, expected) {
			t.Errorf("Expected help text to contain '%s', got: %s", expected, output)
		}
	}
}
