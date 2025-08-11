package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/emt/gitlab-smith/pkg/parser"
	"github.com/spf13/cobra"
)

var parseCmd = &cobra.Command{
	Use:   "parse <file>",
	Short: "Parse and display a GitLab CI configuration file",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		filename := args[0]

		data, err := os.ReadFile(filename)
		if err != nil {
			return fmt.Errorf("reading file: %w", err)
		}

		config, err := parser.Parse(data)
		if err != nil {
			return fmt.Errorf("parsing GitLab CI config: %w", err)
		}

		output, err := json.MarshalIndent(config, "", "  ")
		if err != nil {
			return fmt.Errorf("marshaling output: %w", err)
		}

		fmt.Println(string(output))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(parseCmd)
}
