package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/dominionthedev/mushmellow/internal/config"
	"github.com/dominionthedev/mushmellow/internal/engine"
	"github.com/dominionthedev/mushmellow/internal/ui"
)

func init() {
	rootCmd.AddCommand(newDoctorCmd())
}

func newDoctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Check your mushmellow configuration for issues",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(ui.BuildHeader("Mushmellow Doctor"))
			
			cfg, err := config.LoadDefault()
			if err != nil {
				fmt.Printf("%s Failed to load config: %v\n", ui.Icons.Error, err)
				os.Exit(1)
			}

			fmt.Printf("%s Config file found\n", ui.Icons.Success)

			if err := cfg.Validate(); err != nil {
				fmt.Printf("%s Config validation failed: %v\n", ui.Icons.Error, err)
			} else {
				fmt.Printf("%s Config schema is valid\n", ui.Icons.Success)
			}

			for name, m := range cfg.Mushmellows {
				resolver := engine.NewResolver(m)
				_, err := resolver.Resolve()
				if err != nil {
					fmt.Printf("%s Mushmellow '%s': %v\n", ui.Icons.Error, name, err)
				} else {
					fmt.Printf("%s Mushmellow '%s' dependency graph is healthy\n", ui.Icons.Success, name)
				}
			}

			fmt.Println("\nAll systems soft.")
		},
	}
}
