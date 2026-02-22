package style

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// Binding represents a single key-label pair for the footer.
type Binding struct {
	Key   string
	Label string
}

// B is a shorthand constructor for Binding.
func B(key, label string) Binding {
	return Binding{Key: key, Label: label}
}

// FormatBindings renders a list of bindings with styled keys and muted labels,
// separated by double spaces.
func FormatBindings(bindings []Binding) string {
	parts := make([]string, len(bindings))
	for i, b := range bindings {
		parts[i] = FooterKey.Render(b.Key) + " " + FooterLabel.Render(b.Label)
	}
	return strings.Join(parts, "  ")
}

// FormatFooter renders bindings left-aligned with optional pagination right-aligned.
// If width is 0, no right-alignment is applied.
func FormatFooter(bindings []Binding, pagination string, width int) string {
	left := FormatBindings(bindings)
	if pagination == "" || width == 0 {
		return left
	}
	right := FooterLabel.Render(pagination)
	leftW := ansi.StringWidth(left)
	rightW := ansi.StringWidth(right)
	gap := width - leftW - rightW
	if gap < 2 {
		return left
	}
	return left + strings.Repeat(" ", gap) + right
}

// GlobalFooter renders the standard global navigation line.
func GlobalFooter(width int) string {
	line := FormatBindings([]Binding{
		{"←", "back"},
		{"home", "top"},
		{"⇧home", "default"},
		{"?", "help"},
		{"q", "quit"},
	})
	if width > 0 {
		line = ansi.Truncate(line, width-2, "…")
	}
	return lipgloss.NewStyle().Width(width).Render(line)
}
