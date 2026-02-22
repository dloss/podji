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
// filter mode. If the view already has trailing blank padding, the final
// padding line is replaced to keep the same line budget.
func Append(view string, l list.Model) string {
	if !l.SettingFilter() {
		return view
	}
	bar := l.FilterInput.View()
	lines := strings.Split(view, "\n")
	// Only consume trailing padding. Interior blank lines are content and must
	// remain in place (for example, between table and empty-state message).
	if len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "" {
		lines[len(lines)-1] = bar
		return strings.Join(lines, "\n")
	}
	// Fallback: just append.
	return view + "\n" + bar
}
