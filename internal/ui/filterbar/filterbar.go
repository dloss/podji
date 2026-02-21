package filterbar

import (
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/lipgloss"
	"github.com/dloss/podji/internal/ui/style"
)

// Setup configures a list.Model to hide the default top filter bar and
// restyle the filter input for bottom rendering with a muted "/ " prompt.
func Setup(model *list.Model) {
	model.SetShowFilter(false)
	model.FilterInput.Prompt = "/ "
	model.FilterInput.PromptStyle = style.FilterPrompt
	model.FilterInput.TextStyle = lipgloss.NewStyle()
	model.Styles.FilterPrompt = style.FilterPrompt
}

// Append adds the filter bar at the bottom of the view when the list is in
// filter mode, replacing the last blank line to stay within the same line
// budget (the list pads its output with trailing blank lines).
func Append(view string, l list.Model) string {
	if !l.SettingFilter() {
		return view
	}
	bar := l.FilterInput.View()
	lines := strings.Split(view, "\n")
	// Replace the last blank line with the filter bar.
	for i := len(lines) - 1; i >= 0; i-- {
		if strings.TrimSpace(lines[i]) == "" {
			lines[i] = bar
			return strings.Join(lines, "\n")
		}
	}
	// Fallback: just append.
	return view + "\n" + bar
}
