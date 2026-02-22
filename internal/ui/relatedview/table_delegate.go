package relatedview

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/dloss/podji/internal/ui/style"
)

func newRelatedTableDelegate(findMode *bool, findTargets *map[int]bool) relatedTableDelegate {
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
	return relatedTableDelegate{DefaultDelegate: delegate, findMode: findMode, findTargets: findTargets}
}

type relatedTableDelegate struct {
	list.DefaultDelegate
	findMode    *bool
	findTargets *map[int]bool
}

func (d relatedTableDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	if m.Width() <= 0 {
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

	findTarget := d.findMode != nil && *d.findMode && d.findTargets != nil && (*d.findTargets)[index]

	var row string
	switch it := listItem.(type) {
	case relatedItem:
		row = renderRelatedRow(it, matches, matchStyle, matchBase, findTarget)
	case relationItem:
		row = renderRelationRow(it, matches, matchStyle, matchBase, findTarget)
	default:
		return
	}

	textWidth := m.Width() - titleStyle.GetPaddingLeft() - titleStyle.GetPaddingRight()
	row = ansi.Truncate(row, textWidth, "…")
	fmt.Fprint(w, titleStyle.Render(row)) //nolint:errcheck
}

func renderRelatedRow(
	it relatedItem,
	matches []int,
	matchStyle lipgloss.Style,
	unmatchedStyle lipgloss.Style,
	findTarget bool,
) string {
	cells := make([]string, 0, len(it.row))
	for idx, value := range it.row {
		width := it.widths[idx]
		cellValue := relationPadCell(value, width)
		if idx == 0 && len(matches) > 0 {
			nameMatches := make([]int, 0, len(matches))
			for _, pos := range matches {
				if pos >= 0 && pos < len([]rune(it.entry.name)) {
					nameMatches = append(nameMatches, pos)
				}
			}
			if len(nameMatches) > 0 {
				cellValue = lipgloss.StyleRunes(cellValue, nameMatches, matchStyle, unmatchedStyle)
			}
		}
		if idx == 0 && findTarget {
			cellValue = underlineFirstChar(cellValue)
		}
		cells = append(cells, cellValue)
	}
	return strings.Join(cells, relationColumnSeparator)
}

func renderRelationRow(
	it relationItem,
	matches []int,
	matchStyle lipgloss.Style,
	unmatchedStyle lipgloss.Style,
	findTarget bool,
) string {
	cells := make([]string, 0, len(it.row))
	for idx, value := range it.row {
		width := it.widths[idx]
		cellValue := relationPadCell(value, width)
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
		if idx == 0 && findTarget {
			cellValue = underlineFirstChar(cellValue)
		}
		if idx > 0 && it.status != "" && it.row[idx] == it.status {
			cellValue = style.Status(cellValue)
		}
		cells = append(cells, cellValue)
	}
	return strings.Join(cells, relationColumnSeparator)
}

func underlineFirstChar(value string) string {
	runes := []rune(value)
	if len(runes) == 0 {
		return value
	}
	return "\x1b[4;1;97m" + string(runes[0]) + "\x1b[22;24;39m" + string(runes[1:])
}
