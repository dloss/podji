package logview

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/charmbracelet/bubbles/viewport"
	bubbletea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/dloss/podji/internal/resources"
	"github.com/dloss/podji/internal/ui/style"
	"github.com/dloss/podji/internal/ui/viewstate"
)

var sinceWindows = []string{"1m", "5m", "15m", "1h", "all"}

type logReloadResultMsg struct {
	requestID int
	lines     []string
	err       error
}

type logStreamAppendMsg struct {
	requestID int
	line      string
}

type logStreamDoneMsg struct {
	requestID int
	err       error
}

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
	filterActive bool
	filterQuery  string
	filterValue  string
	requestID    int
	cancel       context.CancelFunc
	streamCh     <-chan bubbletea.Msg
	streamErr    string

	// ContainerViewFactory, when set, is called to produce a container-picker
	// view for the pod. Pressing c opens that picker so the user can switch
	// containers without leaving the log view stack.
	ContainerViewFactory func(item resources.ResourceItem, res resources.ResourceType) viewstate.View
}

func New(item resources.ResourceItem, resource resources.ResourceType) *View {
	return NewWithContainer(item, resource, "")
}

func NewWithContainer(item resources.ResourceItem, resource resources.ResourceType, container string) *View {
	vp := viewport.New(0, 0)
	v := &View{
		item:      item,
		resource:  resource,
		container: container,
		viewport:  vp,
		follow:    true,
		wrap:      true,
		sinceIdx:  1, // default to 5m
	}
	v.reloadLogs()
	v.refreshContent()
	return v
}

func (v *View) Init() bubbletea.Cmd {
	if v.follow {
		return v.reloadLogsCmd()
	}
	return nil
}

func (v *View) Update(msg bubbletea.Msg) viewstate.Update {
	switch msg := msg.(type) {
	case logReloadResultMsg:
		if msg.requestID != v.requestID {
			return viewstate.Update{Action: viewstate.None, Next: v}
		}
		if msg.err == nil && len(msg.lines) > 0 {
			v.streamErr = ""
			v.allLines = msg.lines
			v.refreshWindow()
			v.refreshContent()
		}
		return viewstate.Update{Action: viewstate.None, Next: v}
	case logStreamAppendMsg:
		if msg.requestID != v.requestID {
			return viewstate.Update{Action: viewstate.None, Next: v}
		}
		v.allLines = append(v.allLines, msg.line)
		v.refreshWindow()
		v.refreshContent()
		return viewstate.Update{Action: viewstate.None, Next: v, Cmd: v.nextStreamMsgCmd()}
	case logStreamDoneMsg:
		if msg.requestID != v.requestID {
			return viewstate.Update{Action: viewstate.None, Next: v}
		}
		if !isCanceledErr(msg.err) && msg.err != nil {
			v.streamErr = shortErr(msg.err, 32)
			return viewstate.Update{Action: viewstate.None, Next: v}
		}
		return viewstate.Update{Action: viewstate.None, Next: v}
	case bubbletea.KeyMsg:
		if v.filterActive {
			switch msg.String() {
			case "enter":
				v.filterActive = false
				v.filterValue = strings.TrimSpace(v.filterQuery)
				v.refreshWindow()
				v.refreshContent()
			case "esc":
				v.filterActive = false
				v.filterQuery = v.filterValue
			case "backspace", "ctrl+h":
				r := []rune(v.filterQuery)
				if len(r) > 0 {
					v.filterQuery = string(r[:len(r)-1])
				}
			default:
				if msg.Type == bubbletea.KeyRunes && len(msg.Runes) > 0 {
					for _, r := range msg.Runes {
						if r >= 32 {
							v.filterQuery += string(r)
						}
					}
				}
			}
			return viewstate.Update{Action: viewstate.None, Next: v}
		}

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
				if msg.Type == bubbletea.KeyRunes && len(msg.Runes) > 0 {
					for _, r := range msg.Runes {
						if r >= 32 {
							v.searchQuery += string(r)
						}
					}
				}
			}
			return viewstate.Update{Action: viewstate.None, Next: v}
		}

		switch msg.String() {
		case "esc":
			if strings.TrimSpace(v.filterValue) != "" {
				v.filterValue = ""
				v.filterQuery = ""
				v.refreshWindow()
				v.refreshContent()
				return viewstate.Update{Action: viewstate.None, Next: v}
			}
		case "f":
			v.follow = !v.follow
			return viewstate.Update{Action: viewstate.None, Next: v, Cmd: v.reloadLogsCmd()}
		case "w":
			v.wrap = !v.wrap
			v.refreshContent()
		case "p":
			v.previous = !v.previous
			return viewstate.Update{Action: viewstate.None, Next: v, Cmd: v.reloadLogsCmd()}
		case "&":
			v.filterActive = true
			v.filterQuery = v.filterValue
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
		case "b":
			if len(v.matchLines) > 0 {
				v.matchIndex = (v.matchIndex - 1 + len(v.matchLines)) % len(v.matchLines)
				v.viewport.SetYOffset(v.matchLines[v.matchIndex])
			}
		case ".":
			v.sinceIdx = (v.sinceIdx + 1) % len(sinceWindows)
			return viewstate.Update{Action: viewstate.None, Next: v, Cmd: v.reloadLogsCmd()}
		case ",":
			v.sinceIdx = (v.sinceIdx - 1 + len(sinceWindows)) % len(sinceWindows)
			return viewstate.Update{Action: viewstate.None, Next: v, Cmd: v.reloadLogsCmd()}
		case "c":
			if v.container != "" {
				// Came here via the container picker — pop back so the user can
				// choose a different container.
				return viewstate.Update{Action: viewstate.Pop}
			}
			if v.ContainerViewFactory != nil {
				// Opened directly (e.g. workloads 'o' shortcut) but the pod has
				// multiple containers — push the container picker now.
				return viewstate.Update{Action: viewstate.Push, Next: v.ContainerViewFactory(v.item, v.resource)}
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
	if v.filterActive {
		filterLabel := style.FooterKey.Render("filter")
		filterVal := style.FooterKey.Render("& " + v.filterQuery + "▌")
		line1 := filterLabel + "  " + filterVal
		if v.viewport.Width > 0 {
			line1 = ansi.Truncate(line1, v.viewport.Width-2, "…")
		}
		line2 := style.FormatBindings([]style.Binding{
			style.B("enter", "confirm"),
			style.B("esc", "cancel"),
		})
		if v.viewport.Width > 0 {
			line2 = ansi.Truncate(line2, v.viewport.Width-2, "…")
		}
		return line1 + "\n" + line2
	}
	if v.searchActive {
		searchLabel := style.FooterKey.Render("search")
		searchVal := style.FooterKey.Render("/ " + v.searchQuery + "▌")
		line1 := searchLabel + "  " + searchVal
		if v.viewport.Width > 0 {
			line1 = ansi.Truncate(line1, v.viewport.Width-2, "…")
		}
		line2 := style.FormatBindings([]style.Binding{
			style.B("enter", "confirm"),
			style.B("esc", "cancel"),
		})
		if v.viewport.Width > 0 {
			line2 = ansi.Truncate(line2, v.viewport.Width-2, "…")
		}
		return line1 + "\n" + line2
	}

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
	if v.filterValue != "" {
		indicators = append(indicators, style.B("filter", v.filterValue))
	}
	if len(v.matchLines) > 0 && !v.searchActive {
		indicators = append(indicators, style.B("match", matchSummary(v.matchIndex, len(v.matchLines))))
	}
	if v.streamErr != "" {
		indicators = append(indicators, style.B("stream", v.streamErr))
	}
	line1 := style.FormatBindings(indicators)

	actions := []style.Binding{
		style.B("p", "mode"), style.B("f", "pause/resume"), style.B("w", "wrap"),
		style.B("/", "search"), style.B("&", "filter"),
	}
	if v.container != "" || v.ContainerViewFactory != nil {
		actions = append(actions, style.B("c", "container"))
	}
	if len(v.matchLines) > 0 {
		actions = append(actions, style.B("n/b", "next/prev"))
	}
	actions = append(actions, style.B(", .", "since"))
	actions = append(actions, style.B("pgup/pgdn", "page"))
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

func (v *View) SuppressGlobalKeys() bool {
	return v.searchActive || v.filterActive || strings.TrimSpace(v.filterValue) != "" || len(v.matchLines) > 0
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
	v.lines = applyFilter(applySinceWindow(v.allLines, sinceWindows[v.sinceIdx]), v.filterValue)
}

func (v *View) reloadLogs() {
	// Keep constructor path synchronous to render immediate content.
	opts := resources.LogOptions{
		Tail:      tailForWindow(sinceWindows[v.sinceIdx]),
		Follow:    v.follow,
		Previous:  v.previous,
		Container: v.container,
	}
	if reader, ok := v.resource.(resources.LogOptionsReader); ok {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		lines, err := reader.LogsWithOptions(ctx, v.item, opts)
		if err == nil && len(lines) > 0 {
			v.allLines = lines
			return
		}
	}
	v.allLines = v.resource.Logs(v.item)
}

func (v *View) reloadLogsCmd() bubbletea.Cmd {
	v.cancelReload()
	v.streamErr = ""
	opts := resources.LogOptions{
		Tail:      tailForWindow(sinceWindows[v.sinceIdx]),
		Follow:    v.follow,
		Previous:  v.previous,
		Container: v.container,
	}
	if streamer, ok := v.resource.(resources.LogStreamReader); ok && opts.Follow {
		v.requestID++
		requestID := v.requestID
		ctx, cancel := context.WithCancel(context.Background())
		v.cancel = cancel
		streamCh := make(chan bubbletea.Msg, 256)
		v.streamCh = streamCh
		go func() {
			err := streamer.LogsStream(ctx, v.item, opts, func(line string) {
				select {
				case streamCh <- logStreamAppendMsg{requestID: requestID, line: line}:
				case <-ctx.Done():
				}
			})
			select {
			case streamCh <- logStreamDoneMsg{requestID: requestID, err: err}:
			default:
			}
			close(streamCh)
		}()
		return v.nextStreamMsgCmd()
	}
	if reader, ok := v.resource.(resources.LogOptionsReader); ok {
		v.requestID++
		requestID := v.requestID
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		v.cancel = cancel
		return func() bubbletea.Msg {
			lines, err := reader.LogsWithOptions(ctx, v.item, opts)
			return logReloadResultMsg{requestID: requestID, lines: lines, err: err}
		}
	}
	v.allLines = v.resource.Logs(v.item)
	v.refreshWindow()
	v.refreshContent()
	return nil
}

func (v *View) Dispose() {
	v.cancelReload()
}

func (v *View) cancelReload() {
	if v.cancel != nil {
		v.cancel()
		v.cancel = nil
	}
	v.streamCh = nil
}

func (v *View) nextStreamMsgCmd() bubbletea.Cmd {
	if v.streamCh == nil {
		return nil
	}
	ch := v.streamCh
	return func() bubbletea.Msg {
		msg, ok := <-ch
		if !ok {
			return nil
		}
		return msg
	}
}

func isCanceledErr(err error) bool {
	return errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
}

func shortErr(err error, max int) string {
	if err == nil {
		return ""
	}
	s := strings.TrimSpace(err.Error())
	if max <= 0 || len([]rune(s)) <= max {
		return s
	}
	r := []rune(s)
	return string(r[:max-1]) + "…"
}

func tailForWindow(window string) int {
	switch window {
	case "1m":
		return 50
	case "5m":
		return 200
	case "15m":
		return 500
	case "1h":
		return 1000
	case "all":
		return 2000
	default:
		return 200
	}
}

func applySinceWindow(lines []string, window string) []string {
	if len(lines) == 0 {
		return nil
	}
	// Windowing is handled at fetch time via tailForWindow(). Avoid additional
	// client-side slicing here so live logs are not reduced to tiny subsets.
	out := make([]string, len(lines))
	copy(out, lines)
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

func applyFilter(lines []string, query string) []string {
	query = strings.TrimSpace(strings.ToLower(query))
	if query == "" {
		out := make([]string, len(lines))
		copy(out, lines)
		return out
	}
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		if strings.Contains(strings.ToLower(ansi.Strip(line)), query) {
			out = append(out, line)
		}
	}
	return out
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
