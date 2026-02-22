package listview

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/paginator"
	bubbletea "github.com/charmbracelet/bubbletea"
	"github.com/dloss/podji/internal/resources"
	"github.com/dloss/podji/internal/ui/describeview"
	"github.com/dloss/podji/internal/ui/detailview"
	"github.com/dloss/podji/internal/ui/eventview"
	"github.com/dloss/podji/internal/ui/filterbar"
	"github.com/dloss/podji/internal/ui/logview"
	"github.com/dloss/podji/internal/ui/podpickerview"
	"github.com/dloss/podji/internal/ui/relatedview"
	"github.com/dloss/podji/internal/ui/style"
	"github.com/dloss/podji/internal/ui/viewstate"
	"github.com/dloss/podji/internal/ui/yamlview"
)

const columnSeparator = "  "

type item struct {
	data   resources.ResourceItem
	row    []string
	status string
	widths []int
}

func (i item) Title() string {
	cells := make([]string, 0, len(i.row))
	for idx, value := range i.row {
		width := i.widths[idx]
		cellValue := padCell(value, width)
		if idx > 0 && i.status != "" && i.row[idx] == i.status {
			cellValue = statusStyle(cellValue)
		}
		cells = append(cells, cellValue)
	}
	return strings.Join(cells, columnSeparator)
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
	return i.data.Name
}

type View struct {
	resource    resources.ResourceType
	registry    *resources.Registry
	list        list.Model
	columns     []resources.TableColumn
	colWidths   []int
	sortLabel   string
	findMode    bool
	findTargets map[int]bool
}

func New(resource resources.ResourceType, registry *resources.Registry) *View {
	columns := tableColumns(resource)
	items := resource.Items()
	rows := make([][]string, 0, len(items))
	for _, res := range items {
		rows = append(rows, tableRow(resource, res))
	}
	widths := columnWidthsForRows(columns, rows, 0)
	listItems := make([]list.Item, 0, len(items))
	for idx, res := range items {
		row := rows[idx]
		listItems = append(listItems, item{
			data:   res,
			row:    row,
			status: res.Status,
			widths: widths,
		})
	}

	v := &View{
		resource:  resource,
		registry:  registry,
		columns:   columns,
		colWidths: widths,
		sortLabel: sortMode(resource),
	}
	delegate := newTableDelegate(&v.findMode, &v.findTargets)
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

func (v *View) Init() bubbletea.Cmd {
	return nil
}

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
			if r := singleRune(key); r != 0 {
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
		case "enter", "l", "right", "o":
			if selected, ok := v.list.SelectedItem().(item); ok {
				if next := v.forwardView(selected.data, key.String()); next != nil {
					return viewstate.Update{
						Action: viewstate.Push,
						Next:   next,
					}
				}
				dv := detailview.New(selected.data, v.resource, v.registry)
				dv.ContainerViewFactory = func(item resources.ResourceItem, res resources.ResourceType) viewstate.View {
					return New(resources.NewContainerResource(item, res), v.registry)
				}
				return viewstate.Update{
					Action: viewstate.Push,
					Next:   dv,
				}
			}
		case "s":
			if sortable, ok := v.resource.(resources.ToggleSortable); ok {
				sortable.ToggleSort()
				v.refreshItems()
				v.sortLabel = sortable.SortMode()
			}
			return viewstate.Update{Action: viewstate.None, Next: v}
		case "v":
			if cycler, ok := v.resource.(resources.ScenarioCycler); ok {
				cycler.CycleScenario()
				v.refreshItems()
				return viewstate.Update{Action: viewstate.None, Next: v}
			}
		case "y":
			if selected, ok := v.list.SelectedItem().(item); ok {
				return viewstate.Update{
					Action: viewstate.Push,
					Next:   yamlview.New(selected.data, v.resource),
				}
			}
		case "r":
			if selected, ok := v.list.SelectedItem().(item); ok {
				return viewstate.Update{
					Action: viewstate.Push,
					Next:   relatedview.New(selected.data, v.resource, v.registry),
				}
			}
		case "e":
			if selected, ok := v.list.SelectedItem().(item); ok {
				return viewstate.Update{
					Action: viewstate.Push,
					Next:   eventview.New(selected.data, v.resource),
				}
			}
		case "d":
			if selected, ok := v.list.SelectedItem().(item); ok {
				return viewstate.Update{
					Action: viewstate.Push,
					Next:   describeview.New(selected.data, v.resource),
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

func (v *View) View() string {
	base := v.list.View()
	lines := strings.Split(base, "\n")
	if len(lines) < 2 {
		return base
	}

	// Skip leading blank lines emitted by the list model.
	dataStart := 0
	for dataStart < len(lines) && strings.TrimSpace(lines[dataStart]) == "" {
		dataStart++
	}

	label := resources.SingularName(breadcrumbLabel(v.resource.Name()))
	childHint := resources.SingularName(v.NextBreadcrumb())
	header := "  " + headerRowWithHint(v.columns, v.colWidths, label, childHint)
	out := make([]string, 0, len(lines)+2)
	out = append(out, "")     // blank line separating lens/breadcrumb from table
	out = append(out, header) // column header
	out = append(out, lines[dataStart:]...)

	// Trim trailing blank lines so we stay within the same height budget
	// the list model allocated (we added blank+header but may have removed
	// fewer leading blanks than we added).
	for len(out) > len(lines) && len(out) > 0 && strings.TrimSpace(out[len(out)-1]) == "" {
		out = out[:len(out)-1]
	}

	view := strings.Join(out, "\n")
	if banner := v.bannerMessage(); banner != "" {
		view = style.Warning.Render(banner) + "\n" + view
	}
	if len(v.list.VisibleItems()) == 0 {
		view += "\n\n" + style.Muted.Render("  "+v.emptyMessage())
	}
	return filterbar.Append(view, v.list)
}

func (v *View) Breadcrumb() string {
	return breadcrumbLabel(v.resource.Name())
}

func (v *View) SelectedBreadcrumb() string {
	label := breadcrumbLabel(v.resource.Name())
	selected, ok := v.list.SelectedItem().(item)
	if !ok || selected.data.Name == "" {
		return label
	}
	return label + ": " + selected.data.Name
}

func (v *View) Footer() string {
	// Line 1: status indicators + pagination right-aligned.
	var indicators []style.Binding
	if v.sortLabel != "" {
		indicators = append(indicators, style.B("sort", v.sortLabel))
	}
	if cycler, ok := v.resource.(resources.ScenarioCycler); ok && cycler.Scenario() != "normal" {
		indicators = append(indicators, style.B("state", cycler.Scenario()))
	}
	if v.findMode {
		indicators = append(indicators, style.B("f", "…"))
	}
	if v.list.IsFiltered() {
		indicators = append(indicators, style.B("filter", strings.TrimSpace(v.list.FilterValue())))
	}
	line1 := style.StatusFooter(indicators, v.paginationStatus(), v.list.Width())

	// Line 2: view-specific actions + nav keys + ? help.
	var actions []style.Binding
	isWorkloads := strings.EqualFold(v.resource.Name(), "workloads")
	isContainers := strings.EqualFold(v.resource.Name(), "containers")

	actions = append(actions, style.B("s", "sort"))
	if isWorkloads {
		if _, ok := v.resource.(resources.ScenarioCycler); ok {
			actions = append(actions, style.B("v", "state"))
		}
	}
	if !isContainers {
		actions = append(actions, style.B("tab", "lens"), style.B("r", "related"))
	}
	line2 := style.ActionFooter(actions, v.list.Width())

	return line1 + "\n" + line2
}

func (v *View) SetSize(width, height int) {
	if width == 0 || height == 0 {
		return
	}
	v.list.SetSize(width, height)
	v.refreshItems()
}

func (v *View) SuppressGlobalKeys() bool {
	return v.list.SettingFilter() || v.findMode
}

func (v *View) NextBreadcrumb() string {
	selected, ok := v.list.SelectedItem().(item)
	if !ok {
		return ""
	}

	resourceName := strings.ToLower(v.resource.Name())
	if resourceName == "workloads" {
		return "pods"
	}
	if strings.HasPrefix(resourceName, "pods") || resourceName == "pods" {
		containers := v.resource.Detail(selected.data).Containers
		if len(containers) <= 1 {
			return "logs"
		}
		return "containers"
	}
	if resourceName == "containers" {
		return "logs"
	}
	return "detail"
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

func statusStyle(status string) string {
	return style.Status(status)
}

func headerRow(columns []resources.TableColumn, firstLabel string) string {
	return headerRowWithHint(columns, nil, firstLabel, "")
}

func headerRowWithHint(columns []resources.TableColumn, widths []int, firstLabel string, childHint string) string {
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
				hint := " → " + titleCase(childHint)
				visibleLen := len([]rune(label)) + len([]rune(hint))
				if visibleLen <= width {
					padding := width - visibleLen
					headers = append(headers, label+style.Muted.Render(hint)+strings.Repeat(" ", padding))
					continue
				}
			}
			name = label
		}
		headers = append(headers, padCell(name, width))
	}
	return strings.Join(headers, columnSeparator)
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

func tableColumns(resource resources.ResourceType) []resources.TableColumn {
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

func tableRow(resource resources.ResourceType, res resources.ResourceItem) []string {
	if table, ok := resource.(resources.TableResource); ok {
		return table.TableRow(res)
	}
	return []string{res.Name, res.Status, res.Ready, res.Restarts, res.Age}
}

func columnWidths(columns []resources.TableColumn) []int {
	widths := make([]int, 0, len(columns))
	for _, col := range columns {
		widths = append(widths, col.Width)
	}
	return widths
}

func columnWidthsForRows(columns []resources.TableColumn, rows [][]string, availableWidth int) []int {
	if len(columns) == 0 {
		return nil
	}

	if availableWidth <= 0 {
		return columnWidths(columns)
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

		headerWidth := len([]rune(strings.TrimSpace(col.Name)))
		width := headerWidth
		if maxContent > width {
			width = maxContent
		}
		if width < 1 {
			width = 1
		}
		widths[idx] = width
	}

	availableContent := availableWidth - ((len(columns) - 1) * len(columnSeparator))
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

func sortMode(resource resources.ResourceType) string {
	if sortable, ok := resource.(resources.ToggleSortable); ok {
		return sortable.SortMode()
	}
	return ""
}

func (v *View) refreshItems() {
	items := v.resource.Items()
	rows := make([][]string, 0, len(items))
	for _, res := range items {
		rows = append(rows, tableRow(v.resource, res))
	}
	v.colWidths = columnWidthsForRows(v.columns, rows, v.list.Width()-2)
	listItems := make([]list.Item, 0, len(items))
	for idx, res := range items {
		listItems = append(listItems, item{
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

func breadcrumbLabel(resourceName string) string {
	label := strings.TrimSpace(resourceName)
	if open := strings.Index(label, "("); open > 0 {
		label = strings.TrimSpace(label[:open])
	}
	return label
}

func titleCase(value string) string {
	if value == "" {
		return value
	}
	runes := []rune(value)
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}

func (v *View) emptyMessage() string {
	filter := strings.TrimSpace(v.list.FilterValue())
	if provider, ok := v.resource.(resources.EmptyStateProvider); ok {
		return provider.EmptyMessage(v.list.IsFiltered(), filter)
	}

	if strings.EqualFold(v.resource.Name(), "workloads") {
		if v.list.IsFiltered() {
			return "No workloads match `" + filter + "`. Press esc to clear."
		}
		return "No workloads found in this namespace. Switch namespace or adjust filters."
	}

	if v.list.IsFiltered() {
		return "No items match the active filter. Press esc to clear."
	}
	return "No items found."
}

func (v *View) bannerMessage() string {
	if provider, ok := v.resource.(resources.BannerProvider); ok {
		return provider.Banner()
	}
	return ""
}

func singleRune(key bubbletea.KeyMsg) rune {
	if key.Type == bubbletea.KeyRunes && len(key.Runes) == 1 {
		return key.Runes[0]
	}
	return 0
}

func (v *View) jumpToChar(r rune) {
	target := unicode.ToLower(r)
	visible := v.list.VisibleItems()
	for i, li := range visible {
		if it, ok := li.(item); ok {
			name := strings.TrimSpace(it.data.Name)
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
		if it, ok := li.(item); ok {
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

func (v *View) forwardView(selected resources.ResourceItem, key string) viewstate.View {
	resourceName := strings.ToLower(v.resource.Name())

	if resourceName == "workloads" {
		if key == "o" {
			pods := resources.NewWorkloadPods(selected)
			items := pods.Items()
			if len(items) == 0 {
				return podpickerview.New(selected)
			}
			return logview.New(preferredLogPod(items), pods)
		}
		return New(resources.NewWorkloadPods(selected), v.registry)
	}

	if strings.HasPrefix(resourceName, "pods") || resourceName == "pods" {
		containers := v.resource.Detail(selected).Containers
		if len(containers) <= 1 {
			return logview.New(selected, v.resource)
		}
		return New(resources.NewContainerResource(selected, v.resource), v.registry)
	}

	if resourceName == "containers" {
		if cRes, ok := v.resource.(*resources.ContainerResource); ok {
			return logview.NewWithContainer(cRes.PodItem(), cRes.ParentResource(), selected.Name)
		}
		return logview.New(selected, v.resource)
	}

	if strings.HasPrefix(resourceName, "backends") {
		podContext := resources.NewWorkloadPods(resources.ResourceItem{Name: "backend", Kind: "DEP"})
		dv := detailview.New(selected, podContext, v.registry)
		dv.ContainerViewFactory = func(item resources.ResourceItem, res resources.ResourceType) viewstate.View {
			return New(resources.NewContainerResource(item, res), v.registry)
		}
		return dv
	}

	return nil
}

func preferredLogPod(items []resources.ResourceItem) resources.ResourceItem {
	for _, item := range items {
		switch strings.ToLower(item.Status) {
		case "crashloop", "error", "failed", "pending", "unknown":
			return item
		}
	}
	return items[0]
}
