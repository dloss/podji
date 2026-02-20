package listview

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

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

	titleStyle := d.Styles.NormalTitle
	switch {
	case m.FilterState() == list.Filtering && m.FilterValue() == "":
		titleStyle = d.Styles.DimmedTitle
	case index == m.Index() && m.FilterState() != list.Filtering:
		titleStyle = d.Styles.SelectedTitle
	}

	row := renderRowWithNameMatch(it, m.MatchesForItem(index), d.Styles.FilterMatch)
	textWidth := m.Width() - titleStyle.GetPaddingLeft() - titleStyle.GetPaddingRight()
	row = ansi.Truncate(row, textWidth, "…")
	fmt.Fprint(w, titleStyle.Render(row)) //nolint:errcheck
}

func renderRowWithNameMatch(it item, matches []int, matchStyle lipgloss.Style) string {
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
				unmatched := lipgloss.NewStyle().Inline(true)
				matched := unmatched.Inherit(matchStyle)
				cellValue = lipgloss.StyleRunes(cellValue, nameMatches, matched, unmatched)
			}
		}

		if idx > 0 && it.status != "" && it.row[idx] == it.status {
			cellValue = statusStyle(cellValue)
		}
		cells = append(cells, cellValue)
	}

	return strings.Join(cells, " ")
}
