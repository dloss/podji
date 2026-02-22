package logview

import (
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	bubbletea "github.com/charmbracelet/bubbletea"
	"github.com/dloss/podji/internal/resources"
	"github.com/dloss/podji/internal/ui/style"
	"github.com/dloss/podji/internal/ui/viewstate"
)

type View struct {
	item      resources.ResourceItem
	resource  resources.ResourceType
	container string
	lines     []string
	viewport  viewport.Model
	follow    bool
	wrap      bool
	previous  bool
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
		lines:     lines,
		viewport:  vp,
		follow:    true,
		wrap:      true,
	}
	v.refreshContent()
	return v
}

func (v *View) Init() bubbletea.Cmd { return nil }

func (v *View) Update(msg bubbletea.Msg) viewstate.Update {
	switch msg := msg.(type) {
	case bubbletea.KeyMsg:
		switch msg.String() {
		case "f":
			v.follow = !v.follow
		case "w":
			v.wrap = !v.wrap
			v.refreshContent()
		case "t":
			v.previous = !v.previous
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
	line1 := style.FormatBindings(indicators)

	// Line 2: actions.
	actions := []style.Binding{
		style.B("t", "mode"), style.B("f", "pause/resume"), style.B("w", "wrap"),
		style.B("/", "search"),
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
}

func wrapLines(lines []string, width int) string {
	if width <= 0 {
		return strings.Join(lines, "\n")
	}
	wrapped := make([]string, 0, len(lines))
	for _, line := range lines {
		wrapped = append(wrapped, wrapLine(line, width)...)
	}
	return strings.Join(wrapped, "\n")
}

func wrapLine(line string, width int) []string {
	if line == "" || width <= 0 {
		return []string{line}
	}

	runes := []rune(line)
	if len(runes) <= width {
		return []string{line}
	}

	out := make([]string, 0, (len(runes)/width)+1)
	for start := 0; start < len(runes); start += width {
		end := start + width
		if end > len(runes) {
			end = len(runes)
		}
		out = append(out, string(runes[start:end]))
	}
	return out
}
