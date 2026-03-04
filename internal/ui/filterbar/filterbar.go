package filterbar

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/lipgloss"
	"github.com/dloss/podji/internal/ui/style"
)

// Setup configures a list.Model to hide the default top filter bar and
// restyle the filter input for status row rendering with a muted "/ " prompt.
func Setup(model *list.Model) {
	model.SetShowFilter(false)
	model.KeyMap.Filter = key.NewBinding(
		key.WithKeys("&"),
		key.WithHelp("&", "filter"),
	)
	model.FilterInput.Prompt = "& "
	model.FilterInput.PromptStyle = style.FilterPrompt
	model.FilterInput.TextStyle = lipgloss.NewStyle()
	model.Styles.FilterPrompt = style.FilterPrompt
}

// FilterInputView returns the filter input view when the list is in filter mode.
func FilterInputView(l list.Model) string {
	if !l.SettingFilter() {
		return ""
	}
	return l.FilterInput.View()
}
