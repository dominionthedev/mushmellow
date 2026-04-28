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
				fmt.Printf("❌ Failed to load config: %v\n", err)
				os.Exit(1)
			}

			fmt.Println("✅ Config file found")

			if err := cfg.Validate(); err != nil {
				fmt.Printf("❌ Config validation failed: %v\n", err)
			} else {
				fmt.Println("✅ Config schema is valid")
			}

			for name, m := range cfg.Mushmellows {
				resolver := engine.NewResolver(m)
				_, err := resolver.Resolve()
				if err != nil {
					fmt.Printf("❌ Mushmellow '%s': %v\n", name, err)
				} else {
					fmt.Printf("✅ Mushmellow '%s' dependency graph is healthy\n", name)
				}
			}

			fmt.Println("\nAll systems soft.")
		},
	}
}
