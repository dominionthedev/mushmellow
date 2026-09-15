package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/dominionthedev/mushmellow/internal/graph"
	"github.com/dominionthedev/mushmellow/internal/scheduler"
	"github.com/dominionthedev/mushmellow/internal/state"
	"github.com/dominionthedev/mushmellow/internal/workspace"
)

var runCmd = &cobra.Command{
	Use:   "run <puff>",
	Short: "run a puff and its dependency closure",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		puffName := args[0]

		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		rootPath, err := workspace.FindWorkspaceRoot(cwd)
		if err != nil {
			return err
		}
		root, err := workspace.ParseFile(rootPath)
		if err != nil {
			return err
		}

		g, err := graph.Build(root)
		if err != nil {
			return fmt.Errorf("building graph: %w", err)
		}

		d := scheduler.New(root, g)
		result, runErr := d.Run(puffName)
		if result == nil {
			return runErr
		}

		printReport(result)

		if runErr != nil {
			return runErr
		}
		if final := result.Puffs[puffName]; final != nil &&
			final.Status != state.Success && final.Status != state.Recovered {
			os.Exit(1)
		}
		return nil
	},
}

func printReport(r *state.Run) {
	fmt.Printf("\nrun %s (invoked: %s)\n", r.RunID, r.InvokedPuff)
	fmt.Printf("%-28s %-10s %s\n", "PUFF", "STATUS", "NOTES")
	for name, ps := range r.Puffs {
		notes := ""
		switch ps.Status {
		case state.Blocked:
			notes = fmt.Sprintf("blocked by %s", ps.Reason)
		case state.Skipped:
			notes = fmt.Sprintf("skipped: %s", ps.Reason)
		case state.Cancelled:
			notes = ps.Reason
		}
		if ps.SelfHandler != nil && ps.SelfHandler.Fired {
			notes += fmt.Sprintf(" self_handler(recovered=%v)", ps.SelfHandler.Recovered)
		}
		if len(ps.Attempts) > 1 {
			notes += fmt.Sprintf(" attempts=%d", len(ps.Attempts))
		}
		fmt.Printf("%-28s %-10s %s\n", name, ps.Status, notes)
	}
	fmt.Printf("\nstate: .mushmellow/runs/%s/state.json\n", r.RunID)
}

func init() {
	rootCmd.AddCommand(runCmd)
}
