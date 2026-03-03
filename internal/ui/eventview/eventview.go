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
	cancel   context.CancelFunc
	request  int
}

func New(item resources.ResourceItem, resource resources.ResourceType) *View {
	vp := viewport.New(0, 0)
	vp.SetContent("Loading events...")
	return &View{item: item, resource: resource, viewport: vp}
}

type eventReloadResultMsg struct {
	requestID int
	lines     []string
	err       error
}

func (v *View) Init() bubbletea.Cmd { return v.reloadEventsCmd() }

func readEvents(ctx context.Context, resource resources.ResourceType, item resources.ResourceItem) ([]string, error) {
	if reader, ok := resource.(resources.EventOptionsReader); ok {
		lines, err := reader.EventsWithOptions(ctx, item, resources.EventOptions{Limit: 200})
		if err == nil && len(lines) > 0 {
			return lines, nil
		}
		return nil, err
	}
	return resource.Events(item), nil
}

func (v *View) Update(msg bubbletea.Msg) viewstate.Update {
	switch msg := msg.(type) {
	case eventReloadResultMsg:
		if msg.requestID != v.request {
			return viewstate.Update{Action: viewstate.None, Next: v}
		}
		if msg.err != nil {
			v.viewport.SetContent(strings.Join(v.resource.Events(v.item), "\n"))
			return viewstate.Update{Action: viewstate.None, Next: v}
		}
		lines := msg.lines
		if len(lines) == 0 {
			lines = []string{"No recent events."}
		}
		v.viewport.SetContent(strings.Join(lines, "\n"))
		return viewstate.Update{Action: viewstate.None, Next: v}
	}
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

func (v *View) reloadEventsCmd() bubbletea.Cmd {
	if v.cancel != nil {
		v.cancel()
		v.cancel = nil
	}
	v.request++
	requestID := v.request
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	v.cancel = cancel
	return func() bubbletea.Msg {
		lines, err := readEvents(ctx, v.resource, v.item)
		return eventReloadResultMsg{requestID: requestID, lines: lines, err: err}
	}
}

func (v *View) Dispose() {
	if v.cancel != nil {
		v.cancel()
		v.cancel = nil
	}
}
