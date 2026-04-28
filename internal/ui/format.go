package ui

import (
	"fmt"
	"time"
)

// BuildHeader returns a styled header for a mushmellow
func BuildHeader(name string) string {
	return Styles.MushmellowHeader.Render(fmt.Sprintf("🍡 %s", name))
}

// BuildRun returns a styled running indicator for a puff
func BuildRun(id string) string {
	return fmt.Sprintf("%s %s", Styles.Run.Render("•"), Styles.Name.Render(id))
}

// BuildSuccess returns a styled success indicator for a puff
func BuildSuccess(id string, duration time.Duration) string {
	durStr := Styles.Duration.Render(fmt.Sprintf("(%s)", duration.Round(time.Millisecond)))
	return fmt.Sprintf("%s %s %s", Styles.Passed.Render("✓"), Styles.Name.Render(id), durStr)
}

// BuildError returns a styled error indicator for a puff
func BuildError(id string, err string) string {
	return fmt.Sprintf("%s %s: %s", Styles.Failed.Render("✗"), Styles.Error.Render(id), err)
}

// BuildMessage returns a styled message for a puff
func BuildMessage(text string) string {
	return Styles.PuffHeader.Render(fmt.Sprintf("💬 %s", text))
}
