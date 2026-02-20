package relatedview

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	bubbletea "github.com/charmbracelet/bubbletea"
	"github.com/dloss/podji/internal/resources"
	"github.com/dloss/podji/internal/ui/eventview"
	"github.com/dloss/podji/internal/ui/logview"
	"github.com/dloss/podji/internal/ui/style"
	"github.com/dloss/podji/internal/ui/viewstate"
)

type entry struct {
	title       string
	description string
	open        func() viewstate.View
}

func (e entry) Title() string       { return e.title }
func (e entry) Description() string { return e.description }
func (e entry) FilterValue() string { return e.title + " " + e.description }

type View struct {
	source   resources.ResourceItem
	resource resources.ResourceType
	registry *resources.Registry
	list     list.Model
}

func New(source resources.ResourceItem, resource resources.ResourceType, registry *resources.Registry) *View {
	items := relatedEntries(source, resource, registry)
	listItems := make([]list.Item, 0, len(items))
	for _, it := range items {
		listItems = append(listItems, it)
	}

	delegate := list.NewDefaultDelegate()
	delegate.SetHeight(1)
	delegate.ShowDescription = true
	delegate.Styles.FilterMatch = delegate.Styles.FilterMatch.Underline(false)
	model := list.New(listItems, delegate, 0, 0)
	model.SetShowHelp(false)
	model.SetShowStatusBar(false)
	model.DisableQuitKeybindings()
	model.SetFilteringEnabled(true)

	return &View{source: source, resource: resource, registry: registry, list: model}
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
		case "enter", "l", "right":
			if selected, ok := v.list.SelectedItem().(entry); ok && selected.open != nil {
				return viewstate.Update{Action: viewstate.Push, Next: selected.open()}
			}
		}
	}

	updated, cmd := v.list.Update(msg)
	v.list = updated
	return viewstate.Update{Action: viewstate.None, Next: v, Cmd: cmd}
}

func (v *View) View() string { return v.list.View() }

func (v *View) Breadcrumb() string { return "related" }

func (v *View) Footer() string {
	return "-> open  / filter  esc clear  backspace back"
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
	selected, ok := v.list.SelectedItem().(entry)
	if !ok {
		return ""
	}
	return selected.title
}

func relatedEntries(source resources.ResourceItem, resource resources.ResourceType, registry *resources.Registry) []entry {
	name := strings.ToLower(resource.Name())
	entries := []entry{}

	openResource := func(r resources.ResourceType) func() viewstate.View {
		return func() viewstate.View { return newRelationList(r, registry) }
	}

	if name == "workloads" {
		// Workload tweak: promote Events near top for debugging.
		entries = append(entries, entry{
			title:       "Events (12)",
			description: "Recent warnings and rollout events",
			open:        func() viewstate.View { return eventview.New(source, resource) },
		})
		entries = append(entries, entry{
			title:       fmt.Sprintf("Pods (%d)", len(resources.NewWorkloadPods(source).Items())),
			description: "Owned pods",
			open:        openResource(resources.NewWorkloadPods(source)),
		})
		if source.Kind == "CJ" {
			entries = append(entries, entry{
				title:       "Jobs (2)",
				description: "Owned jobs",
				open:        openResource(resources.NewJobsForCronJob(source.Name)),
			})
		}
		entries = append(entries, entry{
			title:       "Services (1)",
			description: "Network endpoints",
			open:        openResource(resources.NewRelatedServices(source.Name)),
		})
		entries = append(entries, entry{
			title:       "Config (2)",
			description: "ConfigMaps and Secrets",
			open:        openResource(resources.NewRelatedConfig(source.Name)),
		})
		entries = append(entries, entry{
			title:       "Storage (1)",
			description: "PVC and PV references",
			open:        openResource(resources.NewRelatedStorage(source.Name)),
		})
		return entries
	}

	if name == "services" {
		return []entry{
			{title: "Backends (2)", description: "EndpointSlice observed endpoints", open: openResource(resources.NewBackends(source.Name))},
			{title: "Events (4)", description: "Service-related events", open: func() viewstate.View { return eventview.New(source, resource) }},
		}
	}

	if name == "configmaps" || name == "secrets" {
		return []entry{
			{title: "Consumers (2)", description: "Pods/workloads referencing this object", open: openResource(resources.NewConsumers(source.Name))},
			{title: "Events (3)", description: "Recent events", open: func() viewstate.View { return eventview.New(source, resource) }},
		}
	}

	if name == "persistentvolumeclaims" || strings.Contains(name, "pvc") {
		return []entry{
			{title: "Mounted-by (1)", description: "Pods mounting this claim", open: openResource(resources.NewMountedBy(source.Name))},
			{title: "Events (2)", description: "Recent events", open: func() viewstate.View { return eventview.New(source, resource) }},
		}
	}

	return []entry{
		{title: "Events (3)", description: "Recent events", open: func() viewstate.View { return eventview.New(source, resource) }},
	}
}

type resourceItem struct {
	data resources.ResourceItem
}

func (i resourceItem) Title() string {
	return i.data.Name + "  " + i.data.Status + "  " + i.data.Age
}
func (i resourceItem) Description() string { return "enter -> open" }
func (i resourceItem) FilterValue() string {
	return i.data.Name + " " + i.data.Status + " " + i.data.Ready
}

type relationList struct {
	resource resources.ResourceType
	registry *resources.Registry
	list     list.Model
}

func newRelationList(resource resources.ResourceType, registry *resources.Registry) *relationList {
	items := resource.Items()
	listItems := make([]list.Item, 0, len(items))
	for _, item := range items {
		listItems = append(listItems, resourceItem{data: item})
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
	return &relationList{resource: resource, registry: registry, list: model}
}

func (v *relationList) Init() bubbletea.Cmd { return nil }

func (v *relationList) Update(msg bubbletea.Msg) viewstate.Update {
	if key, ok := msg.(bubbletea.KeyMsg); ok {
		switch key.String() {
		case "esc":
			if v.list.SettingFilter() || v.list.IsFiltered() {
				v.list.ResetFilter()
				return viewstate.Update{Action: viewstate.None, Next: v}
			}
		case "enter", "l", "right":
			if selected, ok := v.list.SelectedItem().(resourceItem); ok {
				return viewstate.Update{
					Action: viewstate.Push,
					Next:   logview.New(selected.data, v.resource),
				}
			}
		}
	}

	updated, cmd := v.list.Update(msg)
	v.list = updated
	return viewstate.Update{Action: viewstate.None, Next: v, Cmd: cmd}
}

func (v *relationList) View() string {
	view := v.list.View()
	if len(v.list.VisibleItems()) == 0 {
		if provider, ok := v.resource.(resources.EmptyStateProvider); ok {
			return view + "\n\n" + style.Muted.Render(provider.EmptyMessage(v.list.IsFiltered(), strings.TrimSpace(v.list.FilterValue())))
		}
		return view + "\n\n" + style.Muted.Render("No related items.")
	}
	return view
}

func (v *relationList) Breadcrumb() string { return v.resource.Name() }

func (v *relationList) Footer() string {
	return "-> logs  / filter  esc clear  backspace back"
}

func (v *relationList) SetSize(width, height int) {
	if width == 0 || height == 0 {
		return
	}
	v.list.SetSize(width, height)
}

func (v *relationList) SuppressGlobalKeys() bool {
	return v.list.SettingFilter()
}

func (v *relationList) NextBreadcrumb() string {
	if _, ok := v.list.SelectedItem().(resourceItem); !ok {
		return ""
	}
	return "logs"
}
