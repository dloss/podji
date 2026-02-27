package logview

import (
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/charmbracelet/bubbles/viewport"
	bubbletea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/dloss/podji/internal/resources"
	"github.com/dloss/podji/internal/ui/style"
	"github.com/dloss/podji/internal/ui/viewstate"
)

var sinceWindows = []string{"1m", "5m", "15m", "1h", "all"}

type View struct {
	item      resources.ResourceItem
	resource  resources.ResourceType
	container string
	allLines  []string
	lines     []string
	viewport  viewport.Model
	follow    bool
	wrap      bool
	previous  bool
	sinceIdx  int

	searchActive bool
	searchQuery  string
	matchLines   []int
	matchIndex   int
}

func New(item resources.ResourceItem, resource resources.ResourceType) *View {
	return NewWithContainer(item, resource, "")
}

func NewWithContainer(item resources.ResourceItem, resource resources.ResourceType, container string) *View {
	lines := resource.Logs(item)
	vp := viewport.New(0, 0)
	v := &View{
		item:      item,
		resource:  resource,
		container: container,
		allLines:  lines,
		lines:     lines,
		viewport:  vp,
		follow:    true,
		wrap:      true,
		sinceIdx:  1, // default to 5m
	}
	v.refreshContent()
	return v
}

func (v *View) Init() bubbletea.Cmd { return nil }

func (v *View) Update(msg bubbletea.Msg) viewstate.Update {
	switch msg := msg.(type) {
	case bubbletea.KeyMsg:
		if v.searchActive {
			switch msg.String() {
			case "enter":
				v.searchActive = false
				v.recomputeMatches()
				if len(v.matchLines) > 0 {
					v.matchIndex = 0
					v.viewport.SetYOffset(v.matchLines[v.matchIndex])
				}
			case "esc":
				v.searchActive = false
				v.searchQuery = ""
				v.matchLines = nil
				v.matchIndex = 0
			case "backspace", "ctrl+h":
				r := []rune(v.searchQuery)
				if len(r) > 0 {
					v.searchQuery = string(r[:len(r)-1])
				}
			default:
				if msg.Type == bubbletea.KeyRunes && len(msg.Runes) == 1 && msg.Runes[0] >= 32 {
					v.searchQuery += string(msg.Runes[0])
				}
			}
			return viewstate.Update{Action: viewstate.None, Next: v}
		}

		switch msg.String() {
		case "f":
			v.follow = !v.follow
		case "w":
			v.wrap = !v.wrap
			v.refreshContent()
		case "t":
			v.previous = !v.previous
		case "/":
			v.searchActive = true
			v.searchQuery = ""
			v.matchLines = nil
			v.matchIndex = 0
		case "n":
			if len(v.matchLines) > 0 {
				v.matchIndex = (v.matchIndex + 1) % len(v.matchLines)
				v.viewport.SetYOffset(v.matchLines[v.matchIndex])
			}
		case "N":
			if len(v.matchLines) > 0 {
				v.matchIndex = (v.matchIndex - 1 + len(v.matchLines)) % len(v.matchLines)
				v.viewport.SetYOffset(v.matchLines[v.matchIndex])
			}
		case "]":
			v.sinceIdx = (v.sinceIdx + 1) % len(sinceWindows)
			v.refreshWindow()
			v.refreshContent()
		case "[":
			v.sinceIdx = (v.sinceIdx - 1 + len(sinceWindows)) % len(sinceWindows)
			v.refreshWindow()
			v.refreshContent()
		case "c":
			// When logs were opened for a specific container, pop back to the
			// container list so the user can choose a different one.
			if v.container != "" {
				return viewstate.Update{Action: viewstate.Pop}
			}
		case "pgdown", "pgdn", " ":
			v.viewport.LineDown(pageStep(v.viewport.Height))
		case "pgup":
			v.viewport.LineUp(pageStep(v.viewport.Height))
		case "up", "k":
			v.viewport.LineUp(1)
		case "down", "j":
			v.viewport.LineDown(1)
		}
	}

	updated, cmd := v.viewport.Update(msg)
	v.viewport = updated
	return viewstate.Update{Action: viewstate.None, Next: v, Cmd: cmd}
}

func (v *View) View() string {
	return "\n" + v.viewport.View()
}

func (v *View) Breadcrumb() string {
	return "logs"
}

func (v *View) Footer() string {
	// Line 1: status indicators (non-default only).
	var indicators []style.Binding
	if v.previous {
		indicators = append(indicators, style.B("mode", "previous"))
	}
	if !v.follow {
		indicators = append(indicators, style.B("follow", "off"))
	}
	if !v.wrap {
		indicators = append(indicators, style.B("wrap", "off"))
	}
	if sinceWindows[v.sinceIdx] != "5m" {
		indicators = append(indicators, style.B("since", sinceWindows[v.sinceIdx]))
	}
	if v.searchActive {
		indicators = append(indicators, style.B("/", v.searchQuery+"█"))
	} else if len(v.matchLines) > 0 {
		indicators = append(indicators, style.B("match", matchSummary(v.matchIndex, len(v.matchLines))))
	}
	line1 := style.FormatBindings(indicators)

	// Line 2: actions.
	actions := []style.Binding{
		style.B("t", "mode"), style.B("f", "pause/resume"), style.B("w", "wrap"),
		style.B("/", "search"), style.B("[ ]", "since"), style.B("c", "container"),
		style.B("pgup/pgdn", "page"),
	}
	line2 := style.ActionFooter(actions, v.viewport.Width)
	return line1 + "\n" + line2
}

func (v *View) SetSize(width, height int) {
	if width == 0 || height == 0 {
		return
	}
	v.viewport.Width = width
	v.viewport.Height = height
	v.refreshWindow()
	v.refreshContent()
}

func (v *View) refreshContent() {
	content := strings.Join(v.lines, "\n")
	if v.wrap && v.viewport.Width > 0 {
		content = wrapLines(v.lines, v.viewport.Width)
	}

	atBottom := v.viewport.AtBottom()
	yOffset := v.viewport.YOffset
	v.viewport.SetContent(content)
	if atBottom {
		v.viewport.GotoBottom()
		return
	}
	v.viewport.SetYOffset(yOffset)
	v.recomputeMatches()
}

func (v *View) refreshWindow() {
	v.lines = applySinceWindow(v.allLines, sinceWindows[v.sinceIdx])
}

func applySinceWindow(lines []string, window string) []string {
	if window == "all" {
		out := make([]string, len(lines))
		copy(out, lines)
		return out
	}
	// Stub data doesn't track full timestamps per queryable window. Keep the
	// newest subset for smaller windows so cycling still provides useful context.
	if len(lines) == 0 {
		return nil
	}
	keep := len(lines)
	switch window {
	case "1m":
		keep = minInt(2, len(lines))
	case "5m":
		keep = minInt(4, len(lines))
	case "15m":
		keep = minInt(6, len(lines))
	case "1h":
		keep = minInt(8, len(lines))
	}
	start := len(lines) - keep
	out := make([]string, keep)
	copy(out, lines[start:])
	return out
}

func (v *View) recomputeMatches() {
	if strings.TrimSpace(v.searchQuery) == "" {
		v.matchLines = nil
		v.matchIndex = 0
		return
	}
	query := strings.ToLower(v.searchQuery)
	lines := v.lines
	if v.wrap && v.viewport.Width > 0 {
		lines = wrappedLines(v.lines, v.viewport.Width)
	}
	matches := make([]int, 0, len(lines))
	for i, line := range lines {
		if strings.Contains(strings.ToLower(ansi.Strip(line)), query) {
			matches = append(matches, i)
		}
	}
	v.matchLines = matches
	if len(v.matchLines) == 0 {
		v.matchIndex = 0
		return
	}
	if v.matchIndex >= len(v.matchLines) {
		v.matchIndex = len(v.matchLines) - 1
	}
}

func wrappedLines(lines []string, width int) []string {
	if width <= 0 {
		out := make([]string, len(lines))
		copy(out, lines)
		return out
	}
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		out = append(out, wrapLine(line, width)...)
	}
	return out
}

func matchSummary(index, total int) string {
	if total <= 0 {
		return "0/0"
	}
	return strconv.Itoa(index+1) + "/" + strconv.Itoa(total)
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func pageStep(height int) int {
	if height <= 1 {
		return 1
	}
	return height - 1
}

func wrapLines(lines []string, width int) string {
	if width <= 0 {
		return strings.Join(lines, "\n")
	}
	wrapped := wrappedLines(lines, width)
	return strings.Join(wrapped, "\n")
}

func wrapLine(line string, width int) []string {
	if line == "" || width <= 0 {
		return []string{line}
	}

	if printableRuneWidth(line) <= width {
		return []string{line}
	}

	out := make([]string, 0, (printableRuneWidth(line)/width)+1)
	var segment strings.Builder
	currentWidth := 0
	activeSGR := ""

	for i := 0; i < len(line); {
		seq, n, ok := ansiEscapeAt(line, i)
		if ok {
			segment.WriteString(seq)
			i += n
			if isSGRSequence(seq) {
				if seq == "\x1b[0m" || seq == "\x1b[m" {
					activeSGR = ""
				} else {
					activeSGR = seq
				}
			}
			continue
		}

		r, n := utf8.DecodeRuneInString(line[i:])
		if r == utf8.RuneError && n == 0 {
			break
		}
		rw := printableRuneWidth(string(r))
		if currentWidth+rw > width && segment.Len() > 0 {
			out = append(out, segment.String())
			segment.Reset()
			if activeSGR != "" {
				segment.WriteString(activeSGR)
			}
			currentWidth = 0
		}
		segment.WriteRune(r)
		currentWidth += rw
		i += n
	}
	if segment.Len() > 0 {
		out = append(out, segment.String())
	}
	return out
}

func ansiEscapeAt(s string, i int) (string, int, bool) {
	if i+1 >= len(s) || s[i] != 0x1b || s[i+1] != '[' {
		return "", 0, false
	}
	j := i + 2
	for ; j < len(s); j++ {
		// CSI final bytes are in 0x40..0x7E.
		if s[j] >= 0x40 && s[j] <= 0x7E {
			j++
			return s[i:j], j - i, true
		}
	}
	return "", 0, false
}

func isSGRSequence(seq string) bool {
	return strings.HasSuffix(seq, "m")
}

// printableRuneWidth returns the visible rune width, ignoring ANSI escapes.
func printableRuneWidth(s string) int {
	return len([]rune(ansi.Strip(s)))
}
