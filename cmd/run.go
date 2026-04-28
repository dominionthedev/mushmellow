package cmd

import (
	"context"
	"os"

	"github.com/spf13/cobra"
	"github.com/dominionthedev/mushmellow/internal/config"
	"github.com/dominionthedev/mushmellow/internal/engine"
	"github.com/dominionthedev/mushmellow/internal/ci"
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

			// Load .env if exists
			if _, err := os.Stat(".env"); err == nil {
				dotenv, err := config.LoadEnv(".env")
				if err == nil {
					if cfg.Env == nil {
						cfg.Env = make(map[string]string)
					}
					for k, v := range dotenv {
						cfg.Env[k] = v
					}
				}
			}

			if err := cfg.Validate(); err != nil {
				return err
			}

			modeStr, _ := cmd.Flags().GetString("mode")
			if modeStr == "" {
				modeStr, _ = cmd.Root().PersistentFlags().GetString("mode")
			}
			mode, _ := ci.FromString(modeStr)

			dryRun, _ := cmd.Flags().GetBool("dry-run")

			runner := engine.NewRunner(cfg, mode)
			runner.SetDryRun(dryRun)
			_, err = runner.Run(context.Background(), args[0])
			return err
		},
	}

	cmd.Flags().BoolP("dry-run", "d", false, "Show what would run without executing")

	return cmd
}
