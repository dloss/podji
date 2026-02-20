package detailview

import (
	"strings"

	"github.com/charmbracelet/bubbles/list"
	bubbletea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/dloss/podji/internal/resources"
	"github.com/dloss/podji/internal/ui/logview"
	"github.com/dloss/podji/internal/ui/style"
	"github.com/dloss/podji/internal/ui/viewstate"
)

type containerItem struct {
	name string
}

func (c containerItem) Title() string       { return c.name }
func (c containerItem) Description() string { return "container logs" }
func (c containerItem) FilterValue() string { return c.name }

type ContainerPicker struct {
	item     resources.ResourceItem
	resource resources.ResourceType
	list     list.Model
}

func NewContainerPicker(item resources.ResourceItem, resource resources.ResourceType) *ContainerPicker {
	containers := resource.Detail(item).Containers
	items := make([]list.Item, 0, len(containers))
	for _, container := range containers {
		items = append(items, containerItem{name: container.Name})
	}

	delegate := list.NewDefaultDelegate()
	delegate.SetHeight(1)
	delegate.ShowDescription = false
	model := list.New(items, delegate, 0, 0)
	model.SetShowHelp(false)
	model.SetShowStatusBar(false)
	model.DisableQuitKeybindings()
	model.SetFilteringEnabled(true)
	model.Title = "Containers"
	model.Styles.TitleBar = lipgloss.NewStyle().Padding(0, 0, 0, 2)
	model.Styles.Title = style.Header

	return &ContainerPicker{item: item, resource: resource, list: model}
}

func (v *ContainerPicker) Init() bubbletea.Cmd { return nil }

func (v *ContainerPicker) Update(msg bubbletea.Msg) viewstate.Update {
	if key, ok := msg.(bubbletea.KeyMsg); ok {
		switch key.String() {
		case "esc":
			if v.list.SettingFilter() || v.list.IsFiltered() {
				v.list.ResetFilter()
				return viewstate.Update{Action: viewstate.None, Next: v}
			}
		case "enter", "l", "right":
			if selected, ok := v.list.SelectedItem().(containerItem); ok {
				return viewstate.Update{Action: viewstate.Push, Next: logview.NewWithContainer(v.item, v.resource, selected.name)}
			}
		}
	}

	updated, cmd := v.list.Update(msg)
	v.list = updated
	return viewstate.Update{Action: viewstate.None, Next: v, Cmd: cmd}
}

func (v *ContainerPicker) View() string { return v.list.View() }

func (v *ContainerPicker) Breadcrumb() string { return "containers" }

func (v *ContainerPicker) SelectedBreadcrumb() string {
	selected, ok := v.list.SelectedItem().(containerItem)
	if !ok || selected.name == "" {
		return "containers"
	}
	return "containers: " + selected.name
}

func (v *ContainerPicker) Footer() string {
	return "-> logs  / filter  esc clear  backspace back"
}

func (v *ContainerPicker) SetSize(width, height int) {
	if width == 0 || height == 0 {
		return
	}
	v.list.SetSize(width, height)
}

func (v *ContainerPicker) SuppressGlobalKeys() bool {
	return v.list.SettingFilter()
}

func (v *ContainerPicker) NextBreadcrumb() string {
	if _, ok := v.list.SelectedItem().(containerItem); !ok {
		return ""
	}
	return "logs"
}

func ContainerLabel(item resources.ResourceItem, resource resources.ResourceType) string {
	containers := resource.Detail(item).Containers
	if len(containers) == 0 {
		return ""
	}
	names := make([]string, 0, len(containers))
	for _, container := range containers {
		names = append(names, container.Name)
	}
	return strings.Join(names, ", ")
}
