package resourcebrowser

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

func newBrowserDelegate(findMode *bool, findTargets *map[int]bool) browserDelegate {
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
	return browserDelegate{DefaultDelegate: delegate, findMode: findMode, findTargets: findTargets}
}

type browserDelegate struct {
	list.DefaultDelegate
	findMode    *bool
	findTargets *map[int]bool
}

func (d browserDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	it, ok := listItem.(browserItem)
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

	findTarget := d.findMode != nil && *d.findMode && d.findTargets != nil && (*d.findTargets)[index]
	row := renderBrowserRow(it, matches, matchStyle, matchBase, findTarget)
	textWidth := m.Width() - titleStyle.GetPaddingLeft() - titleStyle.GetPaddingRight()
	row = ansi.Truncate(row, textWidth, "…")
	fmt.Fprint(w, titleStyle.Render(row)) //nolint:errcheck
}

func renderBrowserRow(
	it browserItem,
	matches []int,
	matchStyle lipgloss.Style,
	unmatchedStyle lipgloss.Style,
	findTarget bool,
) string {
	cells := make([]string, 0, len(it.row))
	for idx, value := range it.row {
		cellValue := padCell(value, it.widths[idx])
		if idx == 0 && len(matches) > 0 {
			nameMatches := make([]int, 0, len(matches))
			for _, pos := range matches {
				if pos >= 0 && pos < len([]rune(it.entry.kind)) {
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
	return strings.Join(cells, columnSeparator)
}

func underlineFirstChar(value string) string {
	runes := []rune(value)
	if len(runes) == 0 {
		return value
	}
	return "\x1b[4;1;97m" + string(runes[0]) + "\x1b[22;24;39m" + string(runes[1:])
}
