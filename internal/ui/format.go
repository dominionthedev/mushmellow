package ui

import (
	"fmt"
	"time"
)

// BuildHeader returns a styled header for a mushmellow
func BuildHeader(name string) string {
	return Styles.MushmellowHeader.Render(fmt.Sprintf("%s %s", Icons.Mushmellow, name))
}

// BuildWorkflowInfo returns a styled info line for a workflow
func BuildWorkflowInfo(description string) string {
	return Styles.WorkflowInfo.Render(fmt.Sprintf("%s Workflow: %s", Icons.Info, description))
}

// BuildRun returns a styled running indicator for a puff
func BuildRun(id string) string {
	return fmt.Sprintf("%s %s %s", Styles.PuffIcon.Render(Icons.Bullet), Styles.Action.Render("Puffing:"), Styles.Name.Render(id))
}

// BuildSuccess returns a styled success indicator for a puff
func BuildSuccess(id string, duration time.Duration) string {
	durStr := Styles.Duration.Render(fmt.Sprintf("(%s)", duration.Round(time.Millisecond)))
	return fmt.Sprintf("%s %s %s", Styles.Passed.Render(Icons.Check), Styles.Success.Render(id), durStr)
}

// BuildError returns a styled error indicator for a puff
func BuildError(id string, err string) string {
	return fmt.Sprintf("%s %s: %s", Styles.Failed.Render(Icons.Cross), Styles.Error.Render(id), err)
}

// BuildMessage returns a styled message for a puff
func BuildMessage(text string) string {
	return Styles.PuffHeader.Render(fmt.Sprintf("%s %s", Icons.Message, text))
}
