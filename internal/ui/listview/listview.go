package listview

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/paginator"
	bubbletea "github.com/charmbracelet/bubbletea"
	"github.com/dloss/kubira/internal/resources"
	"github.com/dloss/kubira/internal/ui/detailview"
	"github.com/dloss/kubira/internal/ui/style"
	"github.com/dloss/kubira/internal/ui/viewstate"
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
	model.Paginator.Type = paginator.Arabic
	return &View{resource: resource, registry: registry, list: model}
}

func (v *View) Init() bubbletea.Cmd {
	return nil
}

func (v *View) Update(msg bubbletea.Msg) viewstate.Update {
	if key, ok := msg.(bubbletea.KeyMsg); ok {
		switch key.String() {
		case "esc":
			if v.list.SettingFilter() || v.list.IsFiltered() {
				v.list.ResetFilter()
				return viewstate.Update{Action: viewstate.None, Next: v}
			}
		case "enter", "l", "right":
			if selected, ok := v.list.SelectedItem().(item); ok {
				return viewstate.Update{
					Action: viewstate.Push,
					Next:   detailview.New(selected.data, v.resource),
				}
			}
		}
	}

	updated, cmd := v.list.Update(msg)
	v.list = updated
	return viewstate.Update{Action: viewstate.None, Next: v, Cmd: cmd}
}

func (v *View) View() string {
	return v.list.View()
}

func (v *View) Breadcrumb() string {
	return v.resource.Name()
}

func (v *View) Footer() string {
	parts := []string{v.paginationStatus()}
	if v.list.Paginator.TotalPages > 1 {
		parts = append(parts, "pgup prev-page  pgdn next-page")
	}
	parts = append(parts, "L logs", "/ filter", "esc clear", "? help", "q quit")
	return strings.Join(parts, "  ")
}

func (v *View) SetSize(width, height int) {
	if width == 0 || height == 0 {
		return
	}
	v.list.SetSize(width, height)
}

func (v *View) SuppressGlobalKeys() bool {
	return v.list.SettingFilter()
}

func (v *View) paginationStatus() string {
	totalVisible := len(v.list.VisibleItems())
	if totalVisible == 0 {
		return "Showing 0 of 0"
	}

	start, end := v.list.Paginator.GetSliceBounds(totalVisible)

	if v.list.IsFiltered() {
		return fmt.Sprintf(
			"Showing %d-%d of %d filtered (%d total)",
			start+1,
			end,
			totalVisible,
			len(v.list.Items()),
		)
	}

	return fmt.Sprintf("Showing %d-%d of %d", start+1, end, totalVisible)
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
