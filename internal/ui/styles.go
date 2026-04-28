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
	WorkflowInfo     lipgloss.Style
	Action           lipgloss.Style
	PuffIcon         lipgloss.Style
}

// DefaultStyles returns the default styling for Mushmellow
func DefaultStyles() MushStyles {
	s := MushStyles{}

	// Vibrant Spectrum
	pink := lipgloss.Color("212")
	purple := lipgloss.Color("141")
	cyan := lipgloss.Color("51")
	green := lipgloss.Color("48")
	yellow := lipgloss.Color("226")
	orange := lipgloss.Color("208")
	red := lipgloss.Color("196")
	blue := lipgloss.Color("33")

	s.MushmellowHeader = lipgloss.NewStyle().
		Bold(true).
		Foreground(purple).
		Border(lipgloss.ThickBorder()).
		BorderForeground(pink).
		Padding(0, 4).
		MarginBottom(1)

	s.PuffHeader = lipgloss.NewStyle().
		PaddingLeft(2).
		Foreground(cyan).
		Bold(true)

	s.WorkflowInfo = lipgloss.NewStyle().
		Bold(true).
		Foreground(yellow).
		PaddingLeft(2).
		MarginBottom(1)

	s.Success = lipgloss.NewStyle().
		Foreground(green).
		Bold(true).
		Underline(true)

	s.Error = lipgloss.NewStyle().
		Foreground(red).
		Bold(true)

	s.Delayed = lipgloss.NewStyle().
		Foreground(orange)

	s.Run = lipgloss.NewStyle().
		Foreground(blue).
		PaddingLeft(2)

	s.PuffIcon = lipgloss.NewStyle().
		Foreground(pink)

	s.Action = lipgloss.NewStyle().
		Foreground(cyan).
		Bold(true)

	s.Duration = lipgloss.NewStyle().
		Foreground(lipgloss.Color("244"))

	s.Name = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("255"))

	return s
}

var Styles = DefaultStyles()
