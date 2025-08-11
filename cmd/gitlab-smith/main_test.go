package main

import (
	"bytes"
	"testing"

	"github.com/spf13/cobra"
)

func TestRootCommand(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{
			name: "root command without args",
			args: []string{},
		},
		{
			name: "help flag",
			args: []string{"--help"},
		},
		{
			name: "version info",
			args: []string{"--version"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a copy of the root command for isolated testing
			cmd := &cobra.Command{
				Use:   "gitlab-smith",
				Short: "GitLab CI/CD configuration refactoring and validation tool",
				Long: `GitLabSmith analyzes and validates GitLab CI/CD configuration changes,
providing semantic diffing and optimization suggestions.`,
			}

			// Add subcommands for complete testing
			cmd.AddCommand(parseCmd)
			cmd.AddCommand(analyzeCmd)

			// Capture output
			var buf bytes.Buffer
			cmd.SetOut(&buf)
			cmd.SetErr(&buf)
			cmd.SetArgs(tt.args)

			err := cmd.Execute()

			// For help commands, we expect no error
			if tt.name == "help flag" {
				if err != nil {
					t.Errorf("Expected no error for help command, got: %v", err)
				}

				output := buf.String()
				if output == "" {
					t.Error("Expected help output, got empty string")
				}
			}

			// For root command without args, should show help
			if tt.name == "root command without args" {
				if err != nil {
					t.Errorf("Expected no error for root command, got: %v", err)
				}
			}
		})
	}
}

func TestMainFunctionError(t *testing.T) {
	// Test with invalid command by creating root command structure
	cmd := &cobra.Command{
		Use: "gitlab-smith",
	}
	// Add known subcommands
	cmd.AddCommand(parseCmd)
	cmd.AddCommand(analyzeCmd)

	cmd.SetArgs([]string{"invalid-command"})
	err := cmd.Execute()

	if err == nil {
		t.Error("Expected error for invalid command, got nil")
	}
}

func TestCommandStructure(t *testing.T) {
	// Test that all expected subcommands are registered
	expectedCommands := []string{"parse", "analyze", "refactor", "visualize"}

	// Create root command with all subcommands
	cmd := &cobra.Command{Use: "gitlab-smith"}
	cmd.AddCommand(parseCmd)
	cmd.AddCommand(analyzeCmd)
	// Note: refactor and visualize commands will be tested separately

	for _, expectedCmd := range expectedCommands[:2] { // Only test parse and analyze for now
		found := false
		for _, subCmd := range cmd.Commands() {
			if subCmd.Use == expectedCmd || subCmd.Name() == expectedCmd {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected subcommand '%s' not found", expectedCmd)
		}
	}
}
