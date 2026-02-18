package listview

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbletea"
	"github.com/dloss/kubira/internal/app"
	"github.com/dloss/kubira/internal/resources"
	"github.com/dloss/kubira/internal/ui/detailview"
	"github.com/dloss/kubira/internal/ui/style"
)

type item struct {
	data resources.ResourceItem
}

func (i item) Title() string {
	return i.data.Name
}

func (i item) Description() string {
	parts := []string{}
	if i.data.Status != "" {
		parts = append(parts, i.data.Status)
	}
	if i.data.Ready != "" {
		parts = append(parts, "ready "+i.data.Ready)
	}
	if i.data.Restarts != "" {
		parts = append(parts, "restarts "+i.data.Restarts)
	}
	if i.data.Age != "" {
		parts = append(parts, "age "+i.data.Age)
	}
	return strings.Join(parts, "  ")
}

func (i item) FilterValue() string {
	return i.data.Name + " " + i.data.Status + " " + i.data.Ready
}

type View struct {
	resource resources.ResourceType
	registry *resources.Registry
	list     list.Model
}

func New(resource resources.ResourceType, registry *resources.Registry) *View {
	items := resource.Items()
	listItems := make([]list.Item, 0, len(items))
	for _, res := range items {
		listItems = append(listItems, item{data: res})
	}

	model := list.New(listItems, list.NewDefaultDelegate(), 0, 0)
	model.Title = strings.ToUpper(resource.Name())
	model.SetShowHelp(false)
	model.DisableQuitKeybindings()
	model.SetFilteringEnabled(true)
	return &View{resource: resource, registry: registry, list: model}
}

func (v *View) Init() bubbletea.Cmd {
	return nil
}

func (v *View) Update(msg bubbletea.Msg) app.ViewUpdate {
	if key, ok := msg.(bubbletea.KeyMsg); ok {
		switch key.String() {
		case "enter", "l", "right":
			if selected, ok := v.list.SelectedItem().(item); ok {
				return app.ViewUpdate{
					Action: app.ViewPush,
					Next:   detailview.New(selected.data, v.resource),
				}
			}
		}
	}

	updated, cmd := v.list.Update(msg)
	v.list = updated
	return app.ViewUpdate{Action: app.ViewNone, Next: v, Cmd: cmd}
}

func (v *View) View() string {
	return v.list.View()
}

func (v *View) Breadcrumb() string {
	return v.resource.Name()
}

func (v *View) Footer() string {
	return fmt.Sprintf("j/k navigate  enter detail  l logs  / filter  ? help  q quit  %c", v.resource.Key())
}

func (v *View) SetSize(width, height int) {
	if width == 0 || height == 0 {
		return
	}
	v.list.SetSize(width, height)
}

func statusStyle(status string) string {
	switch status {
	case "CrashLoop", "Error", "Failed":
		return style.Error.Render(status)
	case "Pending", "Warning":
		return style.Warning.Render(status)
	default:
		return style.Healthy.Render(status)
	}
}
