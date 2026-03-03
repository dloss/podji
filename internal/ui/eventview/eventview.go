package eventview

import (
	"context"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	bubbletea "github.com/charmbracelet/bubbletea"
	"github.com/dloss/podji/internal/resources"
	"github.com/dloss/podji/internal/ui/style"
	"github.com/dloss/podji/internal/ui/viewstate"
)

type View struct {
	item     resources.ResourceItem
	resource resources.ResourceType
	viewport viewport.Model
}

func New(item resources.ResourceItem, resource resources.ResourceType) *View {
	lines := readEvents(resource, item)
	vp := viewport.New(0, 0)
	vp.SetContent(strings.Join(lines, "\n"))
	return &View{item: item, resource: resource, viewport: vp}
}

func readEvents(resource resources.ResourceType, item resources.ResourceItem) []string {
	if reader, ok := resource.(resources.EventOptionsReader); ok {
		ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		defer cancel()
		lines, err := reader.EventsWithOptions(ctx, item, resources.EventOptions{Limit: 200})
		if err == nil && len(lines) > 0 {
			return lines
		}
	}
	return resource.Events(item)
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
	return "events"
}

func (v *View) Footer() string {
	line1 := ""
	line2 := style.ActionFooter(nil, v.viewport.Width)
	return line1 + "\n" + line2
}

func (v *View) SetSize(width, height int) {
	if width == 0 || height == 0 {
		return
	}
	v.viewport.Width = width
	v.viewport.Height = height
}
