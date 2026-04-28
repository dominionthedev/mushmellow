package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/dominionthedev/mushmellow/internal/ui"
	"gopkg.in/yaml.v3"
)

func init() {
	rootCmd.AddCommand(newNewCmd())
}

func newNewCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "new <workflow-name>",
		Short: "Scaffold a new mushmellow workflow",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			fmt.Println(ui.BuildHeader("Mushmellow Scaffold"))

			filename := "mushmellow.yaml"
			if _, err := os.Stat(filename); os.IsNotExist(err) {
				// Create initial config if doesn't exist
				fmt.Printf("%s Creating initial %s...\n", ui.Icons.Info, filename)
				initial := map[string]interface{}{
					"version": 1,
					"name":    "My Project",
					"mushmellows": map[string]interface{}{
						name: map[string]interface{}{
							"description": "Scaffolded workflow",
							"puffs": []map[string]interface{}{
								{
									"id":   "hello",
									"type": "message",
									"text": "Hello from " + name + "!",
								},
							},
						},
					},
				}
				data, _ := yaml.Marshal(initial)
				return os.WriteFile(filename, data, 0644)
			}

			fmt.Printf("%s Config file already exists. Updating %s...\n", ui.Icons.Info, filename)
			return fmt.Errorf("adding to existing file not yet implemented in v0.1.1-preview")
		},
	}
}
