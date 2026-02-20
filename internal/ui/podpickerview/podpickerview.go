package podpickerview

import (
	"strings"

	"github.com/charmbracelet/bubbles/list"
	bubbletea "github.com/charmbracelet/bubbletea"
	"github.com/dloss/podji/internal/resources"
	"github.com/dloss/podji/internal/ui/logview"
	"github.com/dloss/podji/internal/ui/style"
	"github.com/dloss/podji/internal/ui/viewstate"
)

type podItem struct {
	data resources.ResourceItem
}

func (p podItem) Title() string {
	return p.data.Name + "  " + style.Status(p.data.Status) + "  " + p.data.Age
}
func (p podItem) Description() string { return "enter -> logs" }
func (p podItem) FilterValue() string { return p.data.Name + " " + p.data.Status }

type View struct {
	workload resources.ResourceItem
	resource *resources.WorkloadPods
	list     list.Model
}

func New(workload resources.ResourceItem) *View {
	resource := resources.NewWorkloadPods(workload)
	items := resource.Items()
	listItems := make([]list.Item, 0, len(items))
	for _, item := range items {
		listItems = append(listItems, podItem{data: item})
	}

	delegate := list.NewDefaultDelegate()
	delegate.SetHeight(1)
	delegate.ShowDescription = false
	delegate.Styles.FilterMatch = delegate.Styles.FilterMatch.Underline(false)
	model := list.New(listItems, delegate, 0, 0)
	model.SetShowHelp(false)
	model.SetShowStatusBar(false)
	model.DisableQuitKeybindings()
	model.SetFilteringEnabled(true)

	if len(items) > 0 {
		// Newest-first in this mock: select first row as the default quick path.
		model.Select(0)
	}

	return &View{
		workload: workload,
		resource: resource,
		list:     model,
	}
}

func (v *View) Init() bubbletea.Cmd { return nil }

func (v *View) Update(msg bubbletea.Msg) viewstate.Update {
	if key, ok := msg.(bubbletea.KeyMsg); ok {
		switch key.String() {
		case "esc":
			if v.list.SettingFilter() || v.list.IsFiltered() {
				v.list.ResetFilter()
				return viewstate.Update{Action: viewstate.None, Next: v}
			}
		case "enter", "l", "L", "right":
			if selected, ok := v.list.SelectedItem().(podItem); ok {
				return viewstate.Update{Action: viewstate.Push, Next: logview.New(selected.data, v.resource)}
			}
		}
	}

	updated, cmd := v.list.Update(msg)
	v.list = updated
	return viewstate.Update{Action: viewstate.None, Next: v, Cmd: cmd}
}

func (v *View) View() string {
	view := v.list.View()
	if len(v.list.VisibleItems()) == 0 {
		return view + "\n\n" + style.Muted.Render(v.resource.EmptyMessage(v.list.IsFiltered(), strings.TrimSpace(v.list.FilterValue())))
	}
	return view
}

func (v *View) Breadcrumb() string {
	return "pod picker"
}

func (v *View) Footer() string {
	return "-> logs  / filter  esc clear  backspace back"
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

func (v *View) NextBreadcrumb() string {
	if _, ok := v.list.SelectedItem().(podItem); !ok {
		return ""
	}
	return "logs"
}
