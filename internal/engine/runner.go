package engine

import (
	"context"
	"fmt"

	"github.com/dominionthedev/mushmellow/internal/config"
	"github.com/dominionthedev/mushmellow/internal/executor"
	"github.com/dominionthedev/mushmellow/internal/ui"
)

// Runner orchestrates the execution of a mushmellow
type Runner struct {
	cfg      *config.Config
	executor *executor.ShellExecutor
}

// NewRunner creates a new Runner
func NewRunner(cfg *config.Config) *Runner {
	return &Runner{
		cfg:      cfg,
		executor: executor.NewShellExecutor(),
	}
}

// Run executes a named mushmellow
func (r *Runner) Run(ctx context.Context, name string) error {
	m, ok := r.cfg.Mushmellows[name]
	if !ok {
		return fmt.Errorf("mushmellow '%s' not found", name)
	}

	fmt.Println(ui.BuildHeader(r.cfg.Name))
	fmt.Printf("Workflow: %s\n\n", m.Description)

	for _, puff := range m.Puffs {
		fmt.Println(ui.BuildRun(puff.ID))

		result := r.executor.ExecutePuff(ctx, puff)
		if !result.Success {
			fmt.Println(ui.BuildError(puff.ID, result.ErrorMessage))
			return fmt.Errorf("puff '%s' failed", puff.ID)
		}

		fmt.Println(ui.BuildSuccess(puff.ID, result.Duration))
	}

	return nil
}
