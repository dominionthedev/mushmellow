package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/dominionthedev/mushmellow/internal/ui"
)

var rootCmd = &cobra.Command{
	Use:   "mushmellow",
	Short: ui.Icons.Mushmellow + " Mushmellow — Soft workflows. Hard execution.",
	Long: `Mushmellow is a lightweight, stylish workflow runtime for defining 
and executing structured developer flows called "mushmellows", composed 
of dependency-aware units called "puffs".

It replaces bash glue, rigid Makefiles, and fragmented scripts with 
something readable, portable, expressive, and aesthetic.`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "%s %v\n", ui.Icons.Error, err)
		os.Exit(1)
	}
}

func init() {
	// Global flags
	rootCmd.PersistentFlags().StringP("mode", "m", "soft", "Execution mode: soft, ci, quiet")
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "verbose output")
}
