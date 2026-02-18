package yamlview

import (
	"github.com/charmbracelet/bubbles/viewport"
	bubbletea "github.com/charmbracelet/bubbletea"
	"github.com/dloss/kubira/internal/resources"
	"github.com/dloss/kubira/internal/ui/viewstate"
)

type View struct {
	item     resources.ResourceItem
	resource resources.ResourceType
	viewport viewport.Model
}

func New(item resources.ResourceItem, resource resources.ResourceType) *View {
	vp := viewport.New(0, 0)
	vp.SetContent(resource.YAML(item))
	return &View{item: item, resource: resource, viewport: vp}
}

func (v *View) Init() bubbletea.Cmd { return nil }

func (v *View) Update(msg bubbletea.Msg) viewstate.Update {
	updated, cmd := v.viewport.Update(msg)
	v.viewport = updated
	return viewstate.Update{Action: viewstate.None, Next: v, Cmd: cmd}
}

func (v *View) View() string {
	return v.viewport.View()
}

func (v *View) Breadcrumb() string {
	return "yaml"
}

func (v *View) Footer() string {
	return "backspace back  esc back"
}

func (v *View) SetSize(width, height int) {
	if width == 0 || height == 0 {
		return
	}
	v.viewport.Width = width
	v.viewport.Height = height
}
