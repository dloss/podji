package logview

import (
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	bubbletea "github.com/charmbracelet/bubbletea"
	"github.com/dloss/kubira/internal/resources"
	"github.com/dloss/kubira/internal/ui/viewstate"
)

type View struct {
	item     resources.ResourceItem
	resource resources.ResourceType
	viewport viewport.Model
	follow   bool
	wrap     bool
	stamp    bool
}

func New(item resources.ResourceItem, resource resources.ResourceType) *View {
	lines := resource.Logs(item)
	vp := viewport.New(0, 0)
	vp.SetContent(strings.Join(lines, "\n"))
	return &View{
		item:     item,
		resource: resource,
		viewport: vp,
		follow:   true,
		wrap:     true,
		stamp:    true,
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
			v.stamp = !v.stamp
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
	return v.viewport.View()
}

func (v *View) Breadcrumb() string {
	return "logs"
}

func (v *View) Footer() string {
	return "f follow  w wrap  t timestamps  c container  / search  space pause  esc back"
}

func (v *View) SetSize(width, height int) {
	if width == 0 || height == 0 {
		return
	}
	v.viewport.Width = width
	v.viewport.Height = height
}
