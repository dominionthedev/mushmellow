package engine

import (
	"context"
	"fmt"

	"github.com/dominionthedev/mushmellow/internal/config"
	"github.com/dominionthedev/mushmellow/internal/executor"
	"github.com/dominionthedev/mushmellow/internal/ui"
	"github.com/dominionthedev/mushmellow/internal/ci"
)

// Runner orchestrates the execution of a mushmellow
type Runner struct {
	cfg      *config.Config
	executor *executor.ShellExecutor
	mode     ci.Mode
	dryRun   bool
}

// NewRunner creates a new Runner
func NewRunner(cfg *config.Config, mode ci.Mode) *Runner {
	return &Runner{
		cfg:      cfg,
		executor: executor.NewShellExecutor(),
		mode:     mode,
	}
}

// SetDryRun enables or disables dry-run mode
func (r *Runner) SetDryRun(dry bool) {
	r.dryRun = dry
}

// Summary represents the outcome of a mushmellow execution
type Summary struct {
	Name    string
	Results []executor.Result
}

// Run executes a named mushmellow
func (r *Runner) Run(ctx context.Context, name string) (*Summary, error) {
	m, ok := r.cfg.Mushmellows[name]
	if !ok {
		return nil, fmt.Errorf("mushmellow '%s' not found", name)
	}

	// Resolve execution order based on dependencies
	resolver := NewResolver(m)
	batches, err := resolver.Resolve()
	if err != nil {
		return nil, fmt.Errorf("failed to resolve dependencies: %w", err)
	}

	if r.mode == ci.SoftMode {
		fmt.Println(ui.BuildHeader(r.cfg.Name))
		fmt.Println(ui.BuildWorkflowInfo(m.Description))
	}

	summary := &Summary{Name: name}

	for _, batch := range batches {
		for _, puff := range batch {
			// Merge environment variables: Global -> Mushmellow -> Puff
			mergedEnv := make(map[string]string)
			for k, v := range r.cfg.Env {
				mergedEnv[k] = v
			}
			for k, v := range m.Env {
				mergedEnv[k] = v
			}
			for k, v := range puff.Env {
				mergedEnv[k] = v
			}
			puff.Env = mergedEnv

			if r.mode == ci.SoftMode {
				if puff.Type == "message" {
					fmt.Println(ui.BuildMessage(puff.Text))
				} else {
					fmt.Println(ui.BuildRun(puff.ID))
				}
			} else if r.mode == ci.CIMode {
				fmt.Printf("%s Executing puff: %s\n", ui.Icons.Bullet, puff.ID)
			}

			if r.dryRun {
				if r.mode == ci.SoftMode {
					fmt.Printf("    (dry-run: %s)\n", puff.Run)
				}
				continue
			}

			result := r.executor.ExecutePuff(ctx, puff)
			summary.Results = append(summary.Results, result)

			if !result.Success {
				if r.mode == ci.SoftMode {
					fmt.Println(ui.BuildError(puff.ID, result.ErrorMessage))
				} else {
					fmt.Printf("%s puff '%s' failed: %s\n", ui.Icons.Error, puff.ID, result.ErrorMessage)
				}
				return summary, fmt.Errorf("puff '%s' failed", puff.ID)
			}

			if r.mode == ci.SoftMode {
				fmt.Println(ui.BuildSuccess(puff.ID, result.Duration))
			} else if r.mode == ci.CIMode {
				fmt.Printf("%s Finished puff: %s (%s)\n", ui.Icons.Success, puff.ID, result.Duration)
			}
		}
	}

	return summary, nil
	}
