package listview

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

func newTableDelegate() tableDelegate {
	delegate := list.NewDefaultDelegate()
	delegate.SetHeight(1)
	delegate.SetSpacing(0)
	delegate.ShowDescription = false
	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.
		Bold(true).
		Foreground(lipgloss.Color("15")).
		Background(lipgloss.Color("236")).
		BorderLeft(true).
		BorderStyle(lipgloss.Border{Left: "▌"})
	return tableDelegate{DefaultDelegate: delegate}
}

// tableDelegate keeps Bubble's default list behavior but scopes filter-match
// highlighting to the first (name) column in table rows.
type tableDelegate struct {
	list.DefaultDelegate
}

func (d tableDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	it, ok := listItem.(item)
	if !ok || m.Width() <= 0 {
		return
	}

	isSelected := index == m.Index()
	isFiltered := m.FilterState() == list.Filtering || m.FilterState() == list.FilterApplied
	titleStyle := d.Styles.NormalTitle
	switch {
	case m.FilterState() == list.Filtering && m.FilterValue() == "":
		titleStyle = d.Styles.DimmedTitle
	case isSelected && m.FilterState() != list.Filtering:
		titleStyle = d.Styles.SelectedTitle
	}

	var matches []int
	if isFiltered {
		matches = m.MatchesForItem(index)
	}
	matchBase := d.Styles.NormalTitle.Inline(true)
	if isSelected {
		matchBase = d.Styles.SelectedTitle.Inline(true)
	}
	matchStyle := matchBase.Inherit(d.Styles.FilterMatch)

	row := renderRowWithNameMatch(it, isSelected, matches, matchStyle, matchBase)
	textWidth := m.Width() - titleStyle.GetPaddingLeft() - titleStyle.GetPaddingRight()
	row = ansi.Truncate(row, textWidth, "…")
	fmt.Fprint(w, titleStyle.Render(row)) //nolint:errcheck
}

func renderRowWithNameMatch(
	it item,
	isSelected bool,
	matches []int,
	matchStyle lipgloss.Style,
	unmatchedStyle lipgloss.Style,
) string {
	cells := make([]string, 0, len(it.row))

	for idx, value := range it.row {
		width := it.widths[idx]
		cellValue := padCell(value, width)

		if idx == 0 && len(matches) > 0 {
			nameMatches := make([]int, 0, len(matches))
			for _, pos := range matches {
				if pos >= 0 && pos < len([]rune(it.data.Name)) {
					nameMatches = append(nameMatches, pos)
				}
			}
			if len(nameMatches) > 0 {
				cellValue = lipgloss.StyleRunes(cellValue, nameMatches, matchStyle, unmatchedStyle)
			}
		}

		if idx > 0 && it.status != "" && it.row[idx] == it.status {
			cellValue = inlineStatusStyle(cellValue, isSelected)
		}

		cells = append(cells, cellValue)
	}

	return strings.Join(cells, " ")
}

// inlineStatusStyle applies status coloring. When preserveBg is true it uses
// raw ANSI foreground+bold sequences so the selected-row background is not
// cleared by a full SGR reset.
func inlineStatusStyle(value string, preserveBg bool) string {
	if !preserveBg {
		return statusStyle(value)
	}
	code := statusANSI(value)
	if code == "" {
		return value
	}
	// Use SGR foreground + bold, then restore with just a foreground reset
	// and bold-off so the background set by the row style is preserved.
	return code + value + "\x1b[22;39m"
}

// statusANSI returns raw ANSI SGR sequences for the status foreground color
// (with bold). Returns "" for unrecognised statuses.
func statusANSI(value string) string {
	trimmed := strings.ToLower(strings.TrimSpace(value))
	switch {
	case strings.Contains(trimmed, "crashloop"),
		strings.Contains(trimmed, "error"),
		strings.Contains(trimmed, "fail"),
		strings.Contains(trimmed, "oom"),
		strings.Contains(trimmed, "backoff"):
		return "\x1b[1;31m" // bold red
	case strings.Contains(trimmed, "pending"),
		strings.Contains(trimmed, "warning"),
		strings.Contains(trimmed, "degraded"),
		strings.Contains(trimmed, "progress"),
		strings.Contains(trimmed, "terminat"),
		strings.Contains(trimmed, "unknown"):
		return "\x1b[1;33m" // bold yellow
	case strings.Contains(trimmed, "suspend"):
		return "\x1b[38;5;241m" // muted (no bold)
	default:
		return "\x1b[92m" // bright green
	}
}
