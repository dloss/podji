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
	name := padCell(i.data.Name, 48)
	status := statusStyle(padCell(i.data.Status, 12))
	ready := padCell(i.data.Ready, 7)
	restarts := padCell(i.data.Restarts, 14)
	age := padCell(i.data.Age, 6)
	return name + " " + status + " " + ready + " " + restarts + " " + age
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

	delegate := list.NewDefaultDelegate()
	delegate.SetHeight(1)
	delegate.SetSpacing(0)
	delegate.ShowDescription = false

	model := list.New(listItems, delegate, 0, 0)
	model.Title = strings.ToUpper(resource.Name())
	model.SetShowHelp(false)
	model.SetShowStatusBar(false)
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
	base := v.list.View()
	lines := strings.Split(base, "\n")
	if len(lines) < 2 {
		return base
	}

	insertAt := 1
	if len(lines) > 1 && lines[1] == "" {
		insertAt = 2
	}

	header := "  " + headerRow()
	out := make([]string, 0, len(lines)+1)
	out = append(out, lines[:insertAt]...)
	out = append(out, header)
	out = append(out, lines[insertAt:]...)

	// Keep dense layout: remove the first empty spacer line after the header.
	for i := insertAt + 1; i < len(out); i++ {
		if strings.TrimSpace(out[i]) == "" {
			out = append(out[:i], out[i+1:]...)
			break
		}
	}

	return strings.Join(out, "\n")
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

func headerRow() string {
	return padCell("NAME", 48) + " " + padCell("STATUS", 12) + " " + padCell("READY", 7) + " " + padCell("RESTARTS", 14) + " " + padCell("AGE", 6)
}

func padCell(value string, width int) string {
	runes := []rune(strings.TrimSpace(value))
	if len(runes) > width {
		if width <= 1 {
			return "…"
		}
		value = string(runes[:width-1]) + "…"
	} else {
		value = string(runes)
	}

	padding := width - len([]rune(value))
	if padding > 0 {
		return value + strings.Repeat(" ", padding)
	}
	return value
}
