package term

import "github.com/charmbracelet/lipgloss"

// Drawer ink. These styles never set a margin, padding, border, or width:
// those add cells, and the desk counts on the line it already fitted.
var (
	stylePick = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("15")).
			Background(lipgloss.Color("5"))
	styleTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("13"))
	stylePrompt = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("14"))
	styleMuted = lipgloss.NewStyle().
			Foreground(lipgloss.Color("8"))
	styleInk = lipgloss.NewStyle().
			Foreground(lipgloss.Color("7"))
	styleArmed = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("13"))
	styleErr = lipgloss.NewStyle().
			Foreground(lipgloss.Color("9"))
)

func paint(text string, on bool) string {
	if text == "" {
		return ""
	}
	if on {
		return stylePick.Render(text)
	}
	return styleInk.Render(text)
}

func dress(text string, selected, armed bool) string {
	if text == "" {
		return ""
	}
	if selected {
		return stylePick.Render(text)
	}
	if armed {
		return styleArmed.Render(text)
	}
	return styleInk.Render(text)
}
