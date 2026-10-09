package tui

import "github.com/charmbracelet/lipgloss"

var (
	accent = lipgloss.AdaptiveColor{Light: "4", Dark: "14"}
	green  = lipgloss.AdaptiveColor{Light: "2", Dark: "10"}
	muted  = lipgloss.AdaptiveColor{Light: "8", Dark: "8"}

	styleActive = lipgloss.NewStyle().Foreground(accent).Bold(true)
	styleDone   = lipgloss.NewStyle().Foreground(green)
	styleBar    = lipgloss.NewStyle().Foreground(muted)
	styleInput  = lipgloss.NewStyle().Foreground(accent)
	styleHint   = lipgloss.NewStyle().Foreground(muted).Italic(true)
	styleChoice = lipgloss.NewStyle().Foreground(muted)
	stylePick   = lipgloss.NewStyle().Foreground(accent).Bold(true)
	styleBack   = lipgloss.NewStyle().Foreground(muted)
	styleNext   = lipgloss.NewStyle().Foreground(accent).Bold(true)
)
