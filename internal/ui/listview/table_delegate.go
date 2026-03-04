package listview

import (
	"fmt"
	"io"
	"regexp"
	"strings"
	"unicode"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

var podGeneratedSegmentRE = regexp.MustCompile(`^[a-z0-9]{4,6}$`)

const podSuffixMutedANSI = "\x1b[38;5;247m"

func newTableDelegate(findMode *bool, findTargets *map[int]bool) tableDelegate {
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
	return tableDelegate{DefaultDelegate: delegate, findMode: findMode, findTargets: findTargets}
}

// tableDelegate keeps Bubble's default list behavior while scoping find
// markers to the configured match column (usually NAME).
type tableDelegate struct {
	list.DefaultDelegate
	findMode    *bool
	findTargets *map[int]bool
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

	findTarget := d.findMode != nil && *d.findMode && d.findTargets != nil && (*d.findTargets)[index]
	row := renderRowWithNameMatch(it, isSelected, matches, matchStyle, matchBase, findTarget)
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
	findTarget bool,
) string {
	cells := make([]string, 0, len(it.row))
	matchColumn := it.matchColumn
	if matchColumn < 0 || matchColumn >= len(it.row) {
		matchColumn = 0
	}
	columnMatches := splitMatchesByColumn(it.row, matches)

	for idx, value := range it.row {
		width := it.widths[idx]
		cellValue := padCell(value, width)
		if idx == matchColumn {
			cellValue = dimPodGeneratedSuffix(cellValue, it.data.Name, it.dimPodName, isSelected, len(matches) > 0)
		}

		if localMatches := visibleMatchesForCell(cellValue, columnMatches[idx]); len(localMatches) > 0 {
			cellValue = lipgloss.StyleRunes(cellValue, localMatches, matchStyle, unmatchedStyle)
		}

		if idx == matchColumn && findTarget {
			cellValue = underlineFirstChar(cellValue)
		}

		if idx > 0 && it.status != "" && it.row[idx] == it.status {
			cellValue = inlineStatusStyle(cellValue, isSelected)
		}

		cells = append(cells, cellValue)
	}

	return strings.Join(cells, columnSeparator)
}

func splitMatchesByColumn(row []string, matches []int) [][]int {
	if len(row) == 0 {
		return nil
	}
	perColumn := make([][]int, len(row))
	if len(matches) == 0 {
		return perColumn
	}
	offset := 0
	for idx, value := range row {
		cellLen := len([]rune(value))
		for _, pos := range matches {
			if pos >= offset && pos < offset+cellLen {
				perColumn[idx] = append(perColumn[idx], pos-offset)
			}
		}
		offset += cellLen
		if idx < len(row)-1 {
			offset++ // spaces inserted by strings.Join(row, " ")
		}
	}
	return perColumn
}

func visibleMatchesForCell(cellValue string, matches []int) []int {
	if len(matches) == 0 {
		return nil
	}
	max := len([]rune(cellValue))
	out := make([]int, 0, len(matches))
	for _, pos := range matches {
		if pos >= 0 && pos < max {
			out = append(out, pos)
		}
	}
	return out
}

// underlineFirstChar applies underline + bright foreground to the first
// visible character of value using raw ANSI sequences so it composes with
// existing row styling.
func underlineFirstChar(value string) string {
	runes := []rune(value)
	if len(runes) == 0 {
		return value
	}
	// \x1b[4m = underline on, \x1b[1m = bold, \x1b[97m = bright white fg
	// \x1b[22;24;39m = bold off, underline off, default fg
	return "\x1b[4;1;97m" + string(runes[0]) + "\x1b[22;24;39m" + string(runes[1:])
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

func dimPodGeneratedSuffix(cellValue, podName string, enabled, isSelected, hasMatches bool) string {
	if !enabled || isSelected || hasMatches {
		return cellValue
	}
	start, end, ok := podGeneratedSuffixRange(podName)
	if !ok {
		return cellValue
	}
	runes := []rune(cellValue)
	if start < 0 || start >= len(runes) || end <= start {
		return cellValue
	}
	if end > len(runes) {
		end = len(runes)
	}
	return string(runes[:start]) + podSuffixMutedANSI + string(runes[start:end]) + "\x1b[39m" + string(runes[end:])
}

// podGeneratedSuffixRange returns the [start,end) rune range for generated pod
// suffixes like "-7c6c8d5f7d-x8p2k" (optionally followed by ordinal suffixes
// like "-01", which stay undimmed).
func podGeneratedSuffixRange(name string) (start, end int, ok bool) {
	runes := []rune(name)
	if len(runes) == 0 {
		return 0, 0, false
	}
	segments, starts, ends := splitDashSegments(name)
	if len(segments) < 3 {
		return 0, 0, false
	}

	for i := 1; i+1 < len(segments); i++ {
		if !isHexHashSegment(segments[i]) {
			continue
		}
		if !isGeneratedPodSegment(segments[i+1]) {
			continue
		}
		// Keep ordinal tails visible: foo-<hash>-<gen>-01.
		if i+2 < len(segments) && !isOrdinalSegment(segments[i+2]) {
			continue
		}
		return starts[i] - 1, ends[i+1], true
	}
	return 0, 0, false
}

func splitDashSegments(s string) (segments []string, starts, ends []int) {
	inSegment := false
	segStart := 0
	runes := []rune(s)
	for i, r := range runes {
		if r == '-' {
			if inSegment {
				segments = append(segments, string(runes[segStart:i]))
				starts = append(starts, segStart)
				ends = append(ends, i)
				inSegment = false
			}
			continue
		}
		if !inSegment {
			inSegment = true
			segStart = i
		}
	}
	if inSegment {
		segments = append(segments, string(runes[segStart:]))
		starts = append(starts, segStart)
		ends = append(ends, len(runes))
	}
	return segments, starts, ends
}

func isHexHashSegment(segment string) bool {
	if len(segment) < 8 || len(segment) > 12 {
		return false
	}
	for _, r := range segment {
		if !unicode.IsDigit(r) && (r < 'a' || r > 'f') {
			return false
		}
	}
	return true
}

func isGeneratedPodSegment(segment string) bool {
	return podGeneratedSegmentRE.MatchString(segment)
}

func isOrdinalSegment(segment string) bool {
	if len(segment) == 0 || len(segment) > 3 {
		return false
	}
	for _, r := range segment {
		if !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}
