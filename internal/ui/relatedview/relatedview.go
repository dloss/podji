package relatedview

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/charmbracelet/bubbles/list"
	bubbletea "github.com/charmbracelet/bubbletea"
	"github.com/dloss/podji/internal/resources"
	"github.com/dloss/podji/internal/ui/eventview"
	"github.com/dloss/podji/internal/ui/filterbar"
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

// RelatedCount returns the number of related-resource categories available for
// the given item.  It is intentionally cheap (no UI objects allocated).
func RelatedCount(source resources.ResourceItem, resource resources.ResourceType, registry *resources.Registry) int {
	return len(relatedEntries(source, resource, registry))
}

func New(source resources.ResourceItem, resource resources.ResourceType, registry *resources.Registry) *View {
	items := relatedEntries(source, resource, registry)
	listItems := make([]list.Item, 0, len(items))
	for _, it := range items {
		listItems = append(listItems, it)
	}

	delegate := list.NewDefaultDelegate()
	delegate.SetHeight(1)
	delegate.SetSpacing(0)
	delegate.ShowDescription = false
	model := list.New(listItems, delegate, 0, 0)
	model.SetShowHelp(false)
	model.SetShowStatusBar(false)
	model.SetShowTitle(false)
	model.DisableQuitKeybindings()
	model.SetFilteringEnabled(true)
	filterbar.Setup(&model)

	return &View{source: source, resource: resource, registry: registry, list: model}
}

func (v *View) Init() bubbletea.Cmd { return nil }

func (v *View) Update(msg bubbletea.Msg) viewstate.Update {
	if key, ok := msg.(bubbletea.KeyMsg); ok {
		if v.list.SettingFilter() && key.String() != "esc" {
			updated, cmd := v.list.Update(msg)
			v.list = updated
			return viewstate.Update{Action: viewstate.None, Next: v, Cmd: cmd}
		}

		switch key.String() {
		case "esc":
			if v.list.SettingFilter() || v.list.IsFiltered() {
				v.list.ResetFilter()
				return viewstate.Update{Action: viewstate.None, Next: v}
			}
		case "enter", "l", "L", "right":
			if selected, ok := v.list.SelectedItem().(entry); ok && selected.open != nil {
				return viewstate.Update{Action: viewstate.Push, Next: selected.open()}
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

	dataStart := 0
	for dataStart < len(lines) && strings.TrimSpace(lines[dataStart]) == "" {
		dataStart++
	}

	header := "  RELATED"
	out := make([]string, 0, len(lines)+2)
	out = append(out, "")
	out = append(out, header)
	out = append(out, lines[dataStart:]...)

	for len(out) > len(lines) && len(out) > 0 && strings.TrimSpace(out[len(out)-1]) == "" {
		out = out[:len(out)-1]
	}

	return filterbar.Append(strings.Join(out, "\n"), v.list)
}

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
	return ""
}

func relatedEntries(source resources.ResourceItem, resource resources.ResourceType, registry *resources.Registry) []entry {
	name := strings.ToLower(resource.Name())
	entries := []entry{}

	openResource := func(r resources.ResourceType) func() viewstate.View {
		return func() viewstate.View { return newRelationList(r, registry) }
	}

	if isPodResource(resource) {
		entries = append(entries, entry{
			title:       "Events (3)",
			description: "Recent warnings and lifecycle events",
			open:        func() viewstate.View { return eventview.New(source, resource) },
		})
		entries = append(entries, entry{
			title:       "Owner (1)",
			description: "Owning workload (Deployment, StatefulSet, etc.)",
			open:        openResource(resources.NewPodOwner(source.Name)),
		})
		entries = append(entries, entry{
			title:       "Services (1)",
			description: "Services selecting this pod",
			open:        openResource(resources.NewPodServices(source.Name)),
		})
		entries = append(entries, entry{
			title:       "Config (2)",
			description: "ConfigMaps and Secrets mounted by this pod",
			open:        openResource(resources.NewPodConfig(source.Name)),
		})
		entries = append(entries, entry{
			title:       "Storage (1)",
			description: "PVCs mounted by this pod",
			open:        openResource(resources.NewPodStorage(source.Name)),
		})
		return entries
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

type relationList struct {
	resource resources.ResourceType
	registry *resources.Registry
	list     list.Model
	columns  []resources.TableColumn
}

func newRelationList(resource resources.ResourceType, registry *resources.Registry) *relationList {
	columns := relationTableColumns(resource)
	widths := relationColumnWidths(columns)
	items := resource.Items()
	listItems := make([]list.Item, 0, len(items))
	for _, res := range items {
		listItems = append(listItems, relationItem{
			data:   res,
			row:    relationTableRow(resource, res),
			status: res.Status,
			widths: widths,
		})
	}

	delegate := list.NewDefaultDelegate()
	delegate.SetHeight(1)
	delegate.SetSpacing(0)
	delegate.ShowDescription = false
	model := list.New(listItems, delegate, 0, 0)
	model.SetShowHelp(false)
	model.SetShowStatusBar(false)
	model.SetShowTitle(false)
	model.DisableQuitKeybindings()
	model.SetFilteringEnabled(true)
	filterbar.Setup(&model)
	return &relationList{resource: resource, registry: registry, list: model, columns: columns}
}

func (v *relationList) Init() bubbletea.Cmd { return nil }

func (v *relationList) Update(msg bubbletea.Msg) viewstate.Update {
	if key, ok := msg.(bubbletea.KeyMsg); ok {
		if v.list.SettingFilter() && key.String() != "esc" {
			updated, cmd := v.list.Update(msg)
			v.list = updated
			return viewstate.Update{Action: viewstate.None, Next: v, Cmd: cmd}
		}

		switch key.String() {
		case "esc":
			if v.list.SettingFilter() || v.list.IsFiltered() {
				v.list.ResetFilter()
				return viewstate.Update{Action: viewstate.None, Next: v}
			}
		case "enter", "l", "L", "right":
			if selected, ok := v.list.SelectedItem().(relationItem); ok {
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
	base := v.list.View()
	lines := strings.Split(base, "\n")
	if len(lines) < 2 {
		return base
	}

	dataStart := 0
	for dataStart < len(lines) && strings.TrimSpace(lines[dataStart]) == "" {
		dataStart++
	}

	label := resources.SingularName(relationBreadcrumbLabel(v.resource.Name()))
	childHint := resources.SingularName(v.NextBreadcrumb())
	header := "  " + relationHeaderRowWithHint(v.columns, label, childHint)
	out := make([]string, 0, len(lines)+2)
	out = append(out, "")
	out = append(out, header)
	out = append(out, lines[dataStart:]...)

	for len(out) > len(lines) && len(out) > 0 && strings.TrimSpace(out[len(out)-1]) == "" {
		out = out[:len(out)-1]
	}

	view := strings.Join(out, "\n")
	if len(v.list.VisibleItems()) == 0 {
		if provider, ok := v.resource.(resources.EmptyStateProvider); ok {
			view += "\n\n" + style.Muted.Render(provider.EmptyMessage(v.list.IsFiltered(), strings.TrimSpace(v.list.FilterValue())))
		} else {
			view += "\n\n" + style.Muted.Render("No related items.")
		}
	}
	return filterbar.Append(view, v.list)
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
	if _, ok := v.list.SelectedItem().(relationItem); !ok {
		return ""
	}
	return "logs"
}

type relationItem struct {
	data   resources.ResourceItem
	row    []string
	status string
	widths []int
}

func (i relationItem) Title() string {
	cells := make([]string, 0, len(i.row))
	for idx, value := range i.row {
		width := i.widths[idx]
		cellValue := relationPadCell(value, width)
		if idx > 0 && i.status != "" && i.row[idx] == i.status {
			cellValue = style.Status(cellValue)
		}
		cells = append(cells, cellValue)
	}
	return strings.Join(cells, " ")
}

func (i relationItem) Description() string { return "" }
func (i relationItem) FilterValue() string {
	return i.data.Name + " " + i.data.Status + " " + i.data.Ready
}

func relationTableColumns(resource resources.ResourceType) []resources.TableColumn {
	if table, ok := resource.(resources.TableResource); ok {
		return table.TableColumns()
	}
	return []resources.TableColumn{
		{Name: "NAME", Width: 48},
		{Name: "STATUS", Width: 12},
		{Name: "READY", Width: 7},
		{Name: "RESTARTS", Width: 14},
		{Name: "AGE", Width: 6},
	}
}

func relationTableRow(resource resources.ResourceType, res resources.ResourceItem) []string {
	if table, ok := resource.(resources.TableResource); ok {
		return table.TableRow(res)
	}
	return []string{res.Name, res.Status, res.Ready, res.Restarts, res.Age}
}

func relationColumnWidths(columns []resources.TableColumn) []int {
	widths := make([]int, 0, len(columns))
	for _, col := range columns {
		widths = append(widths, col.Width)
	}
	return widths
}

func relationHeaderRow(columns []resources.TableColumn, firstLabel string) string {
	return relationHeaderRowWithHint(columns, firstLabel, "")
}

func relationHeaderRowWithHint(columns []resources.TableColumn, firstLabel string, childHint string) string {
	headers := make([]string, 0, len(columns))
	for idx, col := range columns {
		name := col.Name
		if idx == 0 && strings.EqualFold(strings.TrimSpace(col.Name), "name") {
			label := strings.ToUpper(firstLabel)
			if childHint != "" {
				hint := " → " + relationTitleCase(childHint)
				visibleLen := len([]rune(label)) + len([]rune(hint))
				padding := col.Width - visibleLen
				if padding < 0 {
					padding = 0
				}
				headers = append(headers, label+style.Muted.Render(hint)+strings.Repeat(" ", padding))
				continue
			}
			name = label
		}
		headers = append(headers, relationPadCell(name, col.Width))
	}
	return strings.Join(headers, " ")
}

func relationPadCell(value string, width int) string {
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

func relationTitleCase(value string) string {
	if value == "" {
		return value
	}
	runes := []rune(value)
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}

func relationBreadcrumbLabel(resourceName string) string {
	label := strings.TrimSpace(resourceName)
	if open := strings.Index(label, "("); open > 0 {
		label = strings.TrimSpace(label[:open])
	}
	return label
}

func isPodResource(r resources.ResourceType) bool {
	switch r.(type) {
	case *resources.Pods, *resources.WorkloadPods:
		return true
	}
	return false
}
