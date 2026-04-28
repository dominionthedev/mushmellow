package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(newEditCmd())
}

func newEditCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "edit",
		Short: "Open the mushmellow configuration file in your editor",
		RunE: func(cmd *cobra.Command, args []string) error {
			searchPaths := []string{
				"mushmellow.yaml",
				"mushmellow.yml",
				".mushmellow.yaml",
			}

			var configPath string
			for _, p := range searchPaths {
				if _, err := os.Stat(p); err == nil {
					configPath = p
					break
				}
			}

			if configPath == "" {
				return fmt.Errorf("mushmellow.yaml not found in current directory")
			}

			editor := os.Getenv("EDITOR")
			if editor == "" {
				editor = "vi" // Fallback
			}

			c := exec.Command(editor, configPath)
			c.Stdin = os.Stdin
			c.Stdout = os.Stdout
			c.Stderr = os.Stderr

			return c.Run()
		},
	}
}
