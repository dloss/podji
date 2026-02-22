package relatedview

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/paginator"
	bubbletea "github.com/charmbracelet/bubbletea"
	"github.com/dloss/podji/internal/resources"
	"github.com/dloss/podji/internal/ui/eventview"
	"github.com/dloss/podji/internal/ui/filterbar"
	"github.com/dloss/podji/internal/ui/logview"
	"github.com/dloss/podji/internal/ui/style"
	"github.com/dloss/podji/internal/ui/viewstate"
)

const relationColumnSeparator = "  "

type entry struct {
	name        string
	count       int
	description string
	open        func() viewstate.View
}

func (e entry) Title() string {
	if e.count > 0 {
		return fmt.Sprintf("%s (%d)", e.name, e.count)
	}
	return e.name
}
func (e entry) Description() string { return e.description }
func (e entry) FilterValue() string { return e.name }

type View struct {
	source      resources.ResourceItem
	resource    resources.ResourceType
	registry    *resources.Registry
	list        list.Model
	columns     []resources.TableColumn
	colWidths   []int
	findMode    bool
	findTargets map[int]bool
}

// RelatedCount returns the number of related-resource categories available for
// the given item.  It is intentionally cheap (no UI objects allocated).
func RelatedCount(source resources.ResourceItem, resource resources.ResourceType, registry *resources.Registry) int {
	return len(relatedEntries(source, resource, registry))
}

func New(source resources.ResourceItem, resource resources.ResourceType, registry *resources.Registry) *View {
	items := relatedEntries(source, resource, registry)
	columns := relatedTableColumns()
	rows := make([][]string, 0, len(items))
	for _, it := range items {
		rows = append(rows, []string{it.name, relatedCountCell(it.count), it.description})
	}
	widths := relationColumnWidthsForRows(columns, rows, 0, "RELATED")
	listItems := make([]list.Item, 0, len(items))
	for idx, it := range items {
		listItems = append(listItems, relatedItem{
			entry:  it,
			row:    rows[idx],
			widths: widths,
		})
	}

	v := &View{
		source:    source,
		resource:  resource,
		registry:  registry,
		columns:   columns,
		colWidths: widths,
	}
	delegate := newRelatedTableDelegate(&v.findMode, &v.findTargets)
	model := list.New(listItems, delegate, 0, 0)
	model.SetShowHelp(false)
	model.SetShowStatusBar(false)
	model.SetShowTitle(false)
	model.DisableQuitKeybindings()
	model.SetFilteringEnabled(true)
	filterbar.Setup(&model)
	model.Paginator.Type = paginator.Arabic

	v.list = model
	return v
}

func (v *View) Init() bubbletea.Cmd { return nil }

func (v *View) Update(msg bubbletea.Msg) viewstate.Update {
	if key, ok := msg.(bubbletea.KeyMsg); ok {
		if v.list.SettingFilter() && key.String() != "esc" {
			updated, cmd := v.list.Update(msg)
			v.list = updated
			return viewstate.Update{Action: viewstate.None, Next: v, Cmd: cmd}
		}

		if v.findMode {
			v.findMode = false
			v.findTargets = nil
			if key.String() == "esc" {
				return viewstate.Update{Action: viewstate.None, Next: v}
			}
			if r := relatedSingleRune(key); r != 0 {
				v.jumpToChar(r)
			}
			return viewstate.Update{Action: viewstate.None, Next: v}
		}

		switch key.String() {
		case "esc":
			if v.list.SettingFilter() || v.list.IsFiltered() {
				v.list.ResetFilter()
				return viewstate.Update{Action: viewstate.None, Next: v}
			}
		case "enter", "l", "right":
			if selected, ok := v.list.SelectedItem().(relatedItem); ok && selected.entry.open != nil {
				return viewstate.Update{Action: viewstate.Push, Next: selected.entry.open()}
			}
		case "f":
			v.findMode = true
			v.findTargets = v.computeFindTargets()
			return viewstate.Update{Action: viewstate.None, Next: v}
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

	header := "  " + relationHeaderRowWithHint(v.columns, v.colWidths, "related", v.NextBreadcrumb())
	out := make([]string, 0, len(lines)+2)
	out = append(out, "")
	out = append(out, header)
	out = append(out, lines[dataStart:]...)

	for len(out) > len(lines) && len(out) > 0 && strings.TrimSpace(out[len(out)-1]) == "" {
		out = out[:len(out)-1]
	}

	view := strings.Join(out, "\n")
	if len(v.list.VisibleItems()) == 0 {
		if v.list.IsFiltered() {
			view += "\n\n" + style.Muted.Render("No related categories match the active filter. Press esc to clear.")
		} else {
			view += "\n\n" + style.Muted.Render("No related categories found.")
		}
	}

	return filterbar.Append(view, v.list)
}

func (v *View) Breadcrumb() string { return "related" }

func (v *View) Footer() string {
	indicators := []style.Binding{}
	if v.findMode {
		indicators = append(indicators, style.B("f", "…"))
	}
	if v.list.IsFiltered() {
		indicators = append(indicators, style.B("filter", strings.TrimSpace(v.list.FilterValue())))
	}
	line1 := style.StatusFooter(indicators, v.paginationStatus(), v.list.Width())
	line2 := style.ActionFooter([]style.Binding{style.B("tab", "lens")}, v.list.Width())
	return line1 + "\n" + line2
}

func (v *View) SetSize(width, height int) {
	if width == 0 || height == 0 {
		return
	}
	v.list.SetSize(width, height)
	v.refreshItems()
}

func (v *View) refreshItems() {
	items := relatedEntries(v.source, v.resource, v.registry)
	rows := make([][]string, 0, len(items))
	for _, it := range items {
		rows = append(rows, []string{it.name, relatedCountCell(it.count), it.description})
	}
	v.colWidths = relationColumnWidthsForRows(v.columns, rows, v.list.Width()-2, "RELATED")

	listItems := make([]list.Item, 0, len(items))
	for idx, it := range items {
		listItems = append(listItems, relatedItem{
			entry:  it,
			row:    rows[idx],
			widths: v.colWidths,
		})
	}

	selected := v.list.Index()
	v.list.SetItems(listItems)
	if selected >= 0 && selected < len(listItems) {
		v.list.Select(selected)
	}
}

func (v *View) SuppressGlobalKeys() bool {
	return v.list.SettingFilter() || v.findMode
}

func (v *View) NextBreadcrumb() string {
	selected, ok := v.list.SelectedItem().(relatedItem)
	if !ok {
		return ""
	}
	return strings.ToLower(selected.entry.name)
}

func relatedEntries(source resources.ResourceItem, resource resources.ResourceType, registry *resources.Registry) []entry {
	name := strings.ToLower(resource.Name())
	entries := []entry{}

	openResource := func(r resources.ResourceType) func() viewstate.View {
		return func() viewstate.View { return newRelationList(r, registry) }
	}

	if isPodResource(resource) {
		entries = append(entries, entry{
			name:        "events",
			count:       3,
			description: "Recent warnings and lifecycle events",
			open:        func() viewstate.View { return eventview.New(source, resource) },
		})
		entries = append(entries, entry{
			name:        "owner",
			count:       1,
			description: "Owning workload (Deployment, StatefulSet, etc.)",
			open:        openResource(resources.NewPodOwner(source.Name)),
		})
		entries = append(entries, entry{
			name:        "services",
			count:       1,
			description: "Services selecting this pod",
			open:        openResource(resources.NewPodServices(source.Name)),
		})
		entries = append(entries, entry{
			name:        "config",
			count:       2,
			description: "ConfigMaps and Secrets mounted by this pod",
			open:        openResource(resources.NewPodConfig(source.Name)),
		})
		entries = append(entries, entry{
			name:        "storage",
			count:       1,
			description: "PVCs mounted by this pod",
			open:        openResource(resources.NewPodStorage(source.Name)),
		})
		return entries
	}

	if name == "workloads" {
		// Workload tweak: promote Events near top for debugging.
		entries = append(entries, entry{
			name:        "events",
			count:       12,
			description: "Recent warnings and rollout events",
			open:        func() viewstate.View { return eventview.New(source, resource) },
		})
		entries = append(entries, entry{
			name:        "pods",
			count:       len(resources.NewWorkloadPods(source).Items()),
			description: "Owned pods",
			open:        openResource(resources.NewWorkloadPods(source)),
		})
		if source.Kind == "CJ" {
			entries = append(entries, entry{
				name:        "jobs",
				count:       2,
				description: "Owned jobs",
				open:        openResource(resources.NewJobsForCronJob(source.Name)),
			})
		}
		entries = append(entries, entry{
			name:        "services",
			count:       1,
			description: "Network endpoints",
			open:        openResource(resources.NewRelatedServices(source.Name)),
		})
		entries = append(entries, entry{
			name:        "config",
			count:       2,
			description: "ConfigMaps and Secrets",
			open:        openResource(resources.NewRelatedConfig(source.Name)),
		})
		entries = append(entries, entry{
			name:        "storage",
			count:       1,
			description: "PVC and PV references",
			open:        openResource(resources.NewRelatedStorage(source.Name)),
		})
		return entries
	}

	if name == "services" {
		return []entry{
			{name: "backends", count: 2, description: "EndpointSlice observed endpoints", open: openResource(resources.NewBackends(source.Name))},
			{name: "events", count: 4, description: "Service-related events", open: func() viewstate.View { return eventview.New(source, resource) }},
		}
	}

	if name == "configmaps" || name == "secrets" {
		return []entry{
			{name: "consumers", count: 2, description: "Pods/workloads referencing this object", open: openResource(resources.NewConsumers(source.Name))},
			{name: "events", count: 3, description: "Recent events", open: func() viewstate.View { return eventview.New(source, resource) }},
		}
	}

	if name == "persistentvolumeclaims" || strings.Contains(name, "pvc") {
		return []entry{
			{name: "mounted-by", count: 1, description: "Pods mounting this claim", open: openResource(resources.NewMountedBy(source.Name))},
			{name: "events", count: 2, description: "Recent events", open: func() viewstate.View { return eventview.New(source, resource) }},
		}
	}

	return []entry{
		{name: "events", count: 3, description: "Recent events", open: func() viewstate.View { return eventview.New(source, resource) }},
	}
}

type relationList struct {
	resource    resources.ResourceType
	registry    *resources.Registry
	list        list.Model
	columns     []resources.TableColumn
	colWidths   []int
	findMode    bool
	findTargets map[int]bool
}

func newRelationList(resource resources.ResourceType, registry *resources.Registry) *relationList {
	columns := relationTableColumns(resource)
	items := resource.Items()
	rows := make([][]string, 0, len(items))
	for _, res := range items {
		rows = append(rows, relationTableRow(resource, res))
	}
	firstHeader := strings.ToUpper(resources.SingularName(relationBreadcrumbLabel(resource.Name())))
	widths := relationColumnWidthsForRows(columns, rows, 0, firstHeader)
	listItems := make([]list.Item, 0, len(items))
	for idx, res := range items {
		listItems = append(listItems, relationItem{
			data:   res,
			row:    rows[idx],
			status: res.Status,
			widths: widths,
		})
	}

	v := &relationList{resource: resource, registry: registry, columns: columns, colWidths: widths}
	delegate := newRelatedTableDelegate(&v.findMode, &v.findTargets)
	model := list.New(listItems, delegate, 0, 0)
	model.SetShowHelp(false)
	model.SetShowStatusBar(false)
	model.SetShowTitle(false)
	model.DisableQuitKeybindings()
	model.SetFilteringEnabled(true)
	filterbar.Setup(&model)
	model.Paginator.Type = paginator.Arabic
	v.list = model
	return v
}

func (v *relationList) Init() bubbletea.Cmd { return nil }

func (v *relationList) Update(msg bubbletea.Msg) viewstate.Update {
	if key, ok := msg.(bubbletea.KeyMsg); ok {
		if v.list.SettingFilter() && key.String() != "esc" {
			updated, cmd := v.list.Update(msg)
			v.list = updated
			return viewstate.Update{Action: viewstate.None, Next: v, Cmd: cmd}
		}

		if v.findMode {
			v.findMode = false
			v.findTargets = nil
			if key.String() == "esc" {
				return viewstate.Update{Action: viewstate.None, Next: v}
			}
			if r := relatedSingleRune(key); r != 0 {
				v.jumpToChar(r)
			}
			return viewstate.Update{Action: viewstate.None, Next: v}
		}

		switch key.String() {
		case "esc":
			if v.list.SettingFilter() || v.list.IsFiltered() {
				v.list.ResetFilter()
				return viewstate.Update{Action: viewstate.None, Next: v}
			}
		case "enter", "l", "right":
			if selected, ok := v.list.SelectedItem().(relationItem); ok {
				return viewstate.Update{
					Action: viewstate.Push,
					Next:   logview.New(selected.data, v.resource),
				}
			}
		case "f":
			v.findMode = true
			v.findTargets = v.computeFindTargets()
			return viewstate.Update{Action: viewstate.None, Next: v}
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
	header := "  " + relationHeaderRowWithHint(v.columns, v.colWidths, label, childHint)
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
	indicators := []style.Binding{}
	if v.findMode {
		indicators = append(indicators, style.B("f", "…"))
	}
	if v.list.IsFiltered() {
		indicators = append(indicators, style.B("filter", strings.TrimSpace(v.list.FilterValue())))
	}
	line1 := style.StatusFooter(indicators, v.paginationStatus(), v.list.Width())
	line2 := style.ActionFooter([]style.Binding{style.B("tab", "lens")}, v.list.Width())
	return line1 + "\n" + line2
}

func (v *relationList) SetSize(width, height int) {
	if width == 0 || height == 0 {
		return
	}
	v.list.SetSize(width, height)
	v.refreshItems()
}

func (v *relationList) refreshItems() {
	items := v.resource.Items()
	rows := make([][]string, 0, len(items))
	for _, res := range items {
		rows = append(rows, relationTableRow(v.resource, res))
	}
	firstHeader := strings.ToUpper(resources.SingularName(relationBreadcrumbLabel(v.resource.Name())))
	v.colWidths = relationColumnWidthsForRows(v.columns, rows, v.list.Width()-2, firstHeader)

	listItems := make([]list.Item, 0, len(items))
	for idx, res := range items {
		listItems = append(listItems, relationItem{
			data:   res,
			row:    rows[idx],
			status: res.Status,
			widths: v.colWidths,
		})
	}

	selected := v.list.Index()
	v.list.SetItems(listItems)
	if selected >= 0 && selected < len(listItems) {
		v.list.Select(selected)
	}
}

func (v *relationList) SuppressGlobalKeys() bool {
	return v.list.SettingFilter() || v.findMode
}

func (v *relationList) NextBreadcrumb() string {
	if _, ok := v.list.SelectedItem().(relationItem); !ok {
		return ""
	}
	return "logs"
}

type relatedItem struct {
	entry  entry
	row    []string
	widths []int
}

func (i relatedItem) Title() string {
	cells := make([]string, 0, len(i.row))
	for idx, value := range i.row {
		width := i.widths[idx]
		cells = append(cells, relationPadCell(value, width))
	}
	return strings.Join(cells, relationColumnSeparator)
}

func (i relatedItem) Description() string { return "" }
func (i relatedItem) FilterValue() string {
	return i.entry.FilterValue()
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
	return strings.Join(cells, relationColumnSeparator)
}

func (i relationItem) Description() string { return "" }
func (i relationItem) FilterValue() string {
	return i.data.Name
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

func relatedTableColumns() []resources.TableColumn {
	return []resources.TableColumn{
		{Name: "RELATED", Width: 18},
		{Name: "COUNT", Width: 5},
		{Name: "DESCRIPTION", Width: 58},
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

func relationColumnWidthsForRows(columns []resources.TableColumn, rows [][]string, availableWidth int, firstHeader string) []int {
	if len(columns) == 0 {
		return nil
	}

	if availableWidth <= 0 {
		return relationColumnWidths(columns)
	}

	widths := make([]int, len(columns))
	for idx, col := range columns {
		maxContent := 0
		for _, row := range rows {
			if idx >= len(row) {
				continue
			}
			cellWidth := len([]rune(strings.TrimSpace(row[idx])))
			if cellWidth > maxContent {
				maxContent = cellWidth
			}
		}

		headerName := strings.TrimSpace(col.Name)
		if idx == 0 && firstHeader != "" {
			headerName = firstHeader
		}
		headerWidth := len([]rune(headerName))
		width := headerWidth
		if maxContent > width {
			width = maxContent
		}
		if width < 1 {
			width = 1
		}
		widths[idx] = width
	}

	availableContent := availableWidth - ((len(columns) - 1) * len(relationColumnSeparator))
	if availableContent < len(columns) {
		availableContent = len(columns)
	}
	sum := 0
	for _, width := range widths {
		sum += width
	}
	if sum <= availableContent {
		return widths
	}

	minWidths := make([]int, len(widths))
	for idx := range minWidths {
		if idx == 0 {
			minWidths[idx] = 6
		} else {
			minWidths[idx] = 3
		}
		if minWidths[idx] > widths[idx] {
			minWidths[idx] = widths[idx]
		}
	}

	over := sum - availableContent
	for over > 0 {
		progress := false
		for idx := len(widths) - 1; idx >= 1 && over > 0; idx-- {
			if widths[idx] > minWidths[idx] {
				widths[idx]--
				over--
				progress = true
			}
		}
		if over > 0 && widths[0] > minWidths[0] {
			widths[0]--
			over--
			progress = true
		}
		if !progress {
			break
		}
	}

	for over > 0 {
		progress := false
		for idx := len(widths) - 1; idx >= 0 && over > 0; idx-- {
			if widths[idx] > 1 {
				widths[idx]--
				over--
				progress = true
			}
		}
		if !progress {
			break
		}
	}

	return widths
}

func relationHeaderRow(columns []resources.TableColumn, firstLabel string) string {
	return relationHeaderRowWithHint(columns, nil, firstLabel, "")
}

func relationHeaderRowWithHint(columns []resources.TableColumn, widths []int, firstLabel string, childHint string) string {
	headers := make([]string, 0, len(columns))
	for idx, col := range columns {
		width := col.Width
		if idx < len(widths) && widths[idx] > 0 {
			width = widths[idx]
		}
		name := col.Name
		if idx == 0 && strings.EqualFold(strings.TrimSpace(col.Name), "name") {
			label := strings.ToUpper(firstLabel)
			if childHint != "" {
				hint := " → " + relationTitleCase(childHint)
				visibleLen := len([]rune(label)) + len([]rune(hint))
				if visibleLen <= width {
					padding := width - visibleLen
					headers = append(headers, label+style.Muted.Render(hint)+strings.Repeat(" ", padding))
					continue
				}
			}
			name = label
		}
		headers = append(headers, relationPadCell(name, width))
	}
	return strings.Join(headers, relationColumnSeparator)
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

func relatedSingleRune(key bubbletea.KeyMsg) rune {
	if key.Type == bubbletea.KeyRunes && len(key.Runes) == 1 {
		return key.Runes[0]
	}
	return 0
}

func (v *View) jumpToChar(r rune) {
	target := unicode.ToLower(r)
	visible := v.list.VisibleItems()
	for i, li := range visible {
		if it, ok := li.(relatedItem); ok {
			name := strings.TrimSpace(it.entry.name)
			if len(name) > 0 && unicode.ToLower([]rune(name)[0]) == target {
				v.list.Select(i)
				return
			}
		}
	}
}

func (v *View) computeFindTargets() map[int]bool {
	targets := make(map[int]bool)
	seen := make(map[rune]bool)
	for i, li := range v.list.VisibleItems() {
		if it, ok := li.(relatedItem); ok {
			name := strings.TrimSpace(it.entry.name)
			if len(name) > 0 {
				ch := unicode.ToLower([]rune(name)[0])
				if !seen[ch] {
					seen[ch] = true
					targets[i] = true
				}
			}
		}
	}
	return targets
}

func (v *View) paginationStatus() string {
	totalVisible := len(v.list.VisibleItems())
	if totalVisible == 0 {
		if v.list.IsFiltered() {
			return fmt.Sprintf("Showing 0 of 0 filtered (%d total)", len(v.list.Items()))
		}
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

func (v *relationList) jumpToChar(r rune) {
	target := unicode.ToLower(r)
	visible := v.list.VisibleItems()
	for i, li := range visible {
		if it, ok := li.(relationItem); ok {
			name := strings.TrimSpace(it.data.Name)
			if len(name) > 0 && unicode.ToLower([]rune(name)[0]) == target {
				v.list.Select(i)
				return
			}
		}
	}
}

func (v *relationList) computeFindTargets() map[int]bool {
	targets := make(map[int]bool)
	seen := make(map[rune]bool)
	for i, li := range v.list.VisibleItems() {
		if it, ok := li.(relationItem); ok {
			name := strings.TrimSpace(it.data.Name)
			if len(name) > 0 {
				ch := unicode.ToLower([]rune(name)[0])
				if !seen[ch] {
					seen[ch] = true
					targets[i] = true
				}
			}
		}
	}
	return targets
}

func (v *relationList) paginationStatus() string {
	totalVisible := len(v.list.VisibleItems())
	if totalVisible == 0 {
		if v.list.IsFiltered() {
			return fmt.Sprintf("Showing 0 of 0 filtered (%d total)", len(v.list.Items()))
		}
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

func relatedCountCell(count int) string {
	if count <= 0 {
		return "-"
	}
	return fmt.Sprintf("%d", count)
}

func isPodResource(r resources.ResourceType) bool {
	switch r.(type) {
	case *resources.Pods, *resources.WorkloadPods:
		return true
	}
	return false
}
