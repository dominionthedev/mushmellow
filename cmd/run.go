package cmd

import (
	"context"

	"github.com/spf13/cobra"
	"github.com/dominionthedev/mushmellow/internal/config"
	"github.com/dominionthedev/mushmellow/internal/engine"
)

func init() {
	rootCmd.AddCommand(newRunCmd())
}

func newRunCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run <name>",
		Short: "Run a mushmellow workflow",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.LoadDefault()
			if err != nil {
				return err
			}

			if err := cfg.Validate(); err != nil {
				return err
			}

			runner := engine.NewRunner(cfg)
			return runner.Run(context.Background(), args[0])
		},
	}

	return cmd
}
