package style

import (
	"strings"

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

// FormatKeys renders a cluster of bright keys with no labels, separated by single spaces.
// Used for navigation shortcut clusters like "W P D S C K O".
func FormatKeys(keys []string) string {
	parts := make([]string, len(keys))
	for i, k := range keys {
		parts[i] = FooterKey.Render(k)
	}
	return strings.Join(parts, " ")
}

// NavKeys is the standard navigation shortcut cluster ordered by lens grouping.
var NavKeys = []string{"W", "P", "D", "S", "C", "K", "O"}

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

// StatusFooter renders status indicators left-aligned with pagination right-aligned.
// Indicators are only shown when non-default. Returns empty string if no indicators and no pagination.
func StatusFooter(indicators []Binding, pagination string, width int) string {
	return FormatFooter(indicators, pagination, width)
}

// ActionFooter renders view-specific action bindings, then nav keys, then "? help".
func ActionFooter(actions []Binding, width int) string {
	parts := []string{}
	if len(actions) > 0 {
		parts = append(parts, FormatBindings(actions))
	}
	parts = append(parts, FormatKeys(NavKeys))
	parts = append(parts, FormatBindings([]Binding{B("?", "help")}))
	line := strings.Join(parts, "  ")
	if width > 0 {
		line = ansi.Truncate(line, width-2, "…")
	}
	return line
}
