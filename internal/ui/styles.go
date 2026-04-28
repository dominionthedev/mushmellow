package ui

import (
	"github.com/charmbracelet/lipgloss"
)

// MushStyles holds the Lipgloss styling for Mushmellow
type MushStyles struct {
	MushmellowHeader lipgloss.Style
	PuffHeader       lipgloss.Style
	Success          lipgloss.Style
	Error            lipgloss.Style
	Delayed          lipgloss.Style
	Run              lipgloss.Style
	Failed           lipgloss.Style
	Passed           lipgloss.Style
	Duration         lipgloss.Style
	ID               lipgloss.Style
	Name             lipgloss.Style
	Divider          lipgloss.Style
}

// DefaultStyles returns the default styling for Mushmellow
func DefaultStyles() MushStyles {
	s := MushStyles{}

	s.MushmellowHeader = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("129")).
		Padding(1, 0)

	s.PuffHeader = lipgloss.NewStyle().
		PaddingLeft(2).
		Foreground(lipgloss.Color("87"))

	s.Success = lipgloss.NewStyle().
		Foreground(lipgloss.Color("107")).
		Bold(true)

	s.Error = lipgloss.NewStyle().
		Foreground(lipgloss.Color("197")).
		Bold(true)

	s.Delayed = lipgloss.NewStyle().
		Foreground(lipgloss.Color("214"))

	s.Run = lipgloss.NewStyle().PaddingLeft(2)
	s.Failed = lipgloss.NewStyle().PaddingLeft(2)
	s.Passed = lipgloss.NewStyle().PaddingLeft(2)

	s.Duration = lipgloss.NewStyle().
		Foreground(lipgloss.Color("240"))

	s.ID = lipgloss.NewStyle().
		Foreground(lipgloss.Color("246"))

	s.Name = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("255"))

	s.Divider = lipgloss.NewStyle().
		Foreground(lipgloss.Color("235"))

	return s
}

var Styles = DefaultStyles()
