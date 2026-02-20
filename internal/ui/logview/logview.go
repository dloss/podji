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
	vp.SetContent(strings.Join(lines, "\n"))
	return &View{
		item:      item,
		resource:  resource,
		container: container,
		viewport:  vp,
		follow:    true,
		wrap:      true,
	}
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
	return style.Header.Render("  Logs") + "\n" + v.viewport.View()
}

func (v *View) Breadcrumb() string {
	return "logs"
}

func (v *View) Footer() string {
	mode := "current"
	if v.previous {
		mode = "previous"
	}
	return "t mode:" + mode + "  f follow  w wrap  / search  space pause  esc back"
}

func (v *View) SetSize(width, height int) {
	if width == 0 || height == 0 {
		return
	}
	v.viewport.Width = width
	v.viewport.Height = height
}
