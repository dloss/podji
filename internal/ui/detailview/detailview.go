package detailview

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbletea"
	"github.com/dloss/kubira/internal/app"
	"github.com/dloss/kubira/internal/resources"
	"github.com/dloss/kubira/internal/ui/eventview"
	"github.com/dloss/kubira/internal/ui/logview"
	"github.com/dloss/kubira/internal/ui/style"
	"github.com/dloss/kubira/internal/ui/yamlview"
)

type View struct {
	item     resources.ResourceItem
	resource resources.ResourceType
}

func New(item resources.ResourceItem, resource resources.ResourceType) *View {
	return &View{item: item, resource: resource}
}

func (v *View) Init() bubbletea.Cmd { return nil }

func (v *View) Update(msg bubbletea.Msg) app.ViewUpdate {
	if key, ok := msg.(bubbletea.KeyMsg); ok {
		switch key.String() {
		case "l":
			return app.ViewUpdate{Action: app.ViewPush, Next: logview.New(v.item, v.resource)}
		case "e":
			return app.ViewUpdate{Action: app.ViewPush, Next: eventview.New(v.item, v.resource)}
		case "y":
			return app.ViewUpdate{Action: app.ViewPush, Next: yamlview.New(v.item, v.resource)}
		}
	}
	return app.ViewUpdate{Action: app.ViewNone, Next: v}
}

func (v *View) View() string {
	detail := v.resource.Detail(v.item)
	sections := []string{style.Header.Render(detail.StatusLine)}

	if len(detail.Containers) > 0 {
		sections = append(sections, "CONTAINERS")
		sections = append(sections, "NAME       IMAGE              STATE              RESTARTS  REASON")
		for _, row := range detail.Containers {
			sections = append(sections, fmt.Sprintf("%-10s %-18s %-18s %-8s %s", row.Name, row.Image, row.State, row.Restarts, row.Reason))
		}
	}

	if len(detail.Conditions) > 0 {
		sections = append(sections, "", "CONDITIONS")
		sections = append(sections, detail.Conditions...)
	}

	if len(detail.Events) > 0 {
		sections = append(sections, "", "RECENT EVENTS")
		sections = append(sections, detail.Events...)
	}

	if len(detail.Labels) > 0 {
		sections = append(sections, "", "LABELS")
		sections = append(sections, detail.Labels...)
	}

	return strings.Join(sections, "\n")
}

func (v *View) Breadcrumb() string {
	return v.item.Name
}

func (v *View) Footer() string {
	return "backspace back  l logs  e events  y yaml  ? help"
}

func (v *View) SetSize(width, height int) {
}
