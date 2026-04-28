package executor

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/dominionthedev/mushmellow/internal/config"
)

// Result represents the outcome of a puff execution
type Result struct {
	ID           string
	Success      bool
	Duration     time.Duration
	ErrorMessage string
}

// ShellExecutor handles execution of shell commands
type ShellExecutor struct{}

// NewShellExecutor creates a new ShellExecutor
func NewShellExecutor() *ShellExecutor {
	return &ShellExecutor{}
}

// ExecutePuff runs a single puff
func (e *ShellExecutor) ExecutePuff(ctx context.Context, puff config.Puff) Result {
	start := time.Now()

	switch puff.Type {
	case "message":
		fmt.Printf("💬 %s\n", puff.Text)
		return Result{ID: puff.ID, Success: true, Duration: time.Since(start)}
	case "wait":
		duration, err := time.ParseDuration(puff.Duration)
		if err != nil {
			return Result{ID: puff.ID, Success: false, Duration: time.Since(start), ErrorMessage: fmt.Sprintf("invalid duration: %v", err)}
		}
		time.Sleep(duration)
		return Result{ID: puff.ID, Success: true, Duration: time.Since(start)}
	default:
		// Default to "run" type
		return e.runCommand(ctx, puff, start)
	}
}

func (e *ShellExecutor) runCommand(ctx context.Context, puff config.Puff, start time.Time) Result {
	cmd := exec.CommandContext(ctx, "sh", "-c", puff.Run)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if puff.Dir != "" {
		cmd.Dir = puff.Dir
	}

	// Merge env
	if len(puff.Env) > 0 {
		env := os.Environ()
		for k, v := range puff.Env {
			env = append(env, fmt.Sprintf("%s=%s", k, v))
		}
		cmd.Env = env
	}

	if err := cmd.Run(); err != nil {
		return Result{
			ID:           puff.ID,
			Success:      false,
			Duration:     time.Since(start),
			ErrorMessage: err.Error(),
		}
	}

	return Result{
		ID:       puff.ID,
		Success:  true,
		Duration: time.Since(start),
	}
}
