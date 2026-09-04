package cmd

import (
	"fmt"
	"os"
	"sort"

	"github.com/spf13/cobra"

	"github.com/dominionthedev/mushmellow/internal/graph"
	"github.com/dominionthedev/mushmellow/internal/workspace"
)

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "parse the workspace and build the graph, reporting the expanded node set",
	RunE: func(cmd *cobra.Command, args []string) error {
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}

		rootPath, err := workspace.FindWorkspaceRoot(cwd)
		if err != nil {
			return err
		}
		fmt.Printf("workspace root: %s\n\n", rootPath)

		root, err := workspace.ParseFile(rootPath)
		if err != nil {
			return err
		}

		members, err := workspace.DiscoverMembers(root.Dir)
		if err != nil {
			return err
		}
		if len(members) > 0 {
			fmt.Println("members:")
			names := make([]string, 0, len(members))
			for name := range members {
				names = append(names, name)
			}
			sort.Strings(names)
			for _, name := range names {
				puffNames := make([]string, 0, len(members[name].Puffs))
				for pn := range members[name].Puffs {
					puffNames = append(puffNames, pn)
				}
				sort.Strings(puffNames)
				fmt.Printf("  %s (%d puffs: %v)\n", name, len(puffNames), puffNames)
			}
			fmt.Println()
		}

		g, err := graph.Build(root)
		if err != nil {
			return fmt.Errorf("building graph: %w", err)
		}

		fmt.Printf("on_failure: %s\n", root.EffectiveOnFailure())
		base := root.BaseProfile()
		fmt.Printf("base profile %q: pool=%d\n\n", base.Name, base.Pool)

		fmt.Printf("graph: %d node(s)\n", len(g.Nodes))
		names := make([]string, 0, len(g.Nodes))
		for name := range g.Nodes {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			n := g.Nodes[name]
			deps := make([]string, 0, len(n.Edges))
			for _, e := range n.Edges {
				deps = append(deps, fmt.Sprintf("%s(on:%s)", e.Puff, e.EffectiveCondition()))
			}
			profile := n.Puff.Profile
			if profile == "" {
				profile = "(base)"
			}
			fmt.Printf("  - %-28s profile=%-12s deps=%v\n", name, profile, deps)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(validateCmd)
}
