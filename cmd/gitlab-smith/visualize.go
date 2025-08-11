package main

import (
	"fmt"
	"os"

	"github.com/wonderfulspam/gitlab-smith/pkg/parser"
	"github.com/wonderfulspam/gitlab-smith/pkg/renderer"
	"github.com/spf13/cobra"
)

var visualizeCmd = &cobra.Command{
	Use:   "visualize <config-file>",
	Short: "Generate a visual representation of a GitLab CI pipeline",
	Long: `Creates a visual diagram of the GitLab CI pipeline structure showing jobs, stages, 
and dependencies. Supports DOT graph and Mermaid diagram formats.`,
	Args: cobra.ExactArgs(1),
	RunE: runVisualize,
}

var (
	visualFormat     string
	visualOutputFile string
)

func init() {
	visualizeCmd.Flags().StringVar(&visualFormat, "format", "mermaid", "Visual format (dot, mermaid)")
	visualizeCmd.Flags().StringVar(&visualOutputFile, "output", "", "Output file for the diagram (default: stdout)")

	rootCmd.AddCommand(visualizeCmd)
}

func runVisualize(cmd *cobra.Command, args []string) error {
	configFile := args[0]

	// Parse configuration
	data, err := os.ReadFile(configFile)
	if err != nil {
		return fmt.Errorf("reading config file '%s': %w", configFile, err)
	}

	config, err := parser.Parse(data)
	if err != nil {
		return fmt.Errorf("parsing GitLab CI config '%s': %w", configFile, err)
	}

	// Generate visual representation
	renderer := renderer.New(nil)
	visualOutput, err := renderer.RenderVisualPipeline(config, visualFormat)
	if err != nil {
		return fmt.Errorf("generating visual representation: %w", err)
	}

	// Write output
	if visualOutputFile != "" {
		err = os.WriteFile(visualOutputFile, []byte(visualOutput), 0644)
		if err != nil {
			return fmt.Errorf("writing output file: %w", err)
		}

		// Provide helpful instructions based on format
		switch visualFormat {
		case "dot":
			fmt.Printf("DOT graph written to %s\n", visualOutputFile)
			fmt.Println("💡 To generate an image: dot -Tpng -o pipeline.png " + visualOutputFile)
		case "mermaid":
			fmt.Printf("Mermaid diagram written to %s\n", visualOutputFile)
			fmt.Println("💡 View online at: https://mermaid.live/")
		}
	} else {
		fmt.Print(visualOutput)
	}

	return nil
}
