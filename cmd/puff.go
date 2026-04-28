package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/dominionthedev/mushmellow/internal/ui"
)

func init() {
	rootCmd.AddCommand(newPuffCmd())
}

func newPuffCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "puff",
		Short: "Manage puffs in your workflows",
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "list <workflow>",
		Short: "List puffs in a workflow",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(ui.BuildHeader("Puff Manager"))
			fmt.Printf("%s Listing puffs for %s is coming soon...\n", ui.Icons.Info, args[0])
		},
	})

	return cmd
}
