package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "gitlab-smith",
	Short: "GitLab CI/CD configuration refactoring and validation tool",
	Long: `GitLabSmith analyzes and validates GitLab CI/CD configuration changes,
providing semantic diffing and optimization suggestions.`,
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
