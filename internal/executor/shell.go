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
	// Handle timeout
	if puff.Timeout != "" {
		duration, err := time.ParseDuration(puff.Timeout)
		if err == nil {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, duration)
			defer cancel()
		}
	}

	cmd := exec.CommandContext(ctx, "sh", "-c", puff.Run)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if puff.Dir != "" {
		cmd.Dir = puff.Dir
	}

	// Env inheritance + merging
	env := os.Environ()
	if len(puff.Env) > 0 {
		for k, v := range puff.Env {
			env = append(env, fmt.Sprintf("%s=%s", k, v))
		}
	}
	cmd.Env = env

	if err := cmd.Run(); err != nil {
		errorMessage := err.Error()
		if ctx.Err() == context.DeadlineExceeded {
			errorMessage = fmt.Sprintf("timed out after %s", puff.Timeout)
		}
		return Result{
			ID:           puff.ID,
			Success:      false,
			Duration:     time.Since(start),
			ErrorMessage: errorMessage,
		}
	}

	return Result{
		ID:       puff.ID,
		Success:  true,
		Duration: time.Since(start),
	}
}
