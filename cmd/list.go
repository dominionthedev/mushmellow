package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/dominionthedev/mushmellow/internal/config"
	"github.com/dominionthedev/mushmellow/internal/ui"
)

func init() {
	rootCmd.AddCommand(newListCmd())
}

func newListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List available mushmellows",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.LoadDefault()
			if err != nil {
				return err
			}

			fmt.Println(ui.BuildHeader(cfg.Name))
			fmt.Println("Available workflows:")
			fmt.Println()

			for name, m := range cfg.Mushmellows {
				fmt.Printf("  %s - %s (%d puffs)\n", name, m.Description, len(m.Puffs))
			}

			return nil
		},
	}
}
