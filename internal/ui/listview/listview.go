package listview

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/paginator"
	bubbletea "github.com/charmbracelet/bubbletea"
	"github.com/dloss/podji/internal/resources"
	"github.com/dloss/podji/internal/ui/detailview"
	"github.com/dloss/podji/internal/ui/logview"
	"github.com/dloss/podji/internal/ui/podpickerview"
	"github.com/dloss/podji/internal/ui/relatedview"
	"github.com/dloss/podji/internal/ui/style"
	"github.com/dloss/podji/internal/ui/viewstate"
)

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
	return strings.Join(cells, " ")
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
	resource  resources.ResourceType
	registry  *resources.Registry
	list      list.Model
	columns   []resources.TableColumn
	sortLabel string
}

func New(resource resources.ResourceType, registry *resources.Registry) *View {
	columns := tableColumns(resource)
	items := resource.Items()
	listItems := make([]list.Item, 0, len(items))
	for _, res := range items {
		row := tableRow(resource, res)
		listItems = append(listItems, item{
			data:   res,
			row:    row,
			status: res.Status,
			widths: columnWidths(columns),
		})
	}

	delegate := list.NewDefaultDelegate()
	delegate.SetHeight(1)
	delegate.SetSpacing(0)
	delegate.ShowDescription = false

	model := list.New(listItems, delegate, 0, 0)
	model.SetShowHelp(false)
	model.SetShowStatusBar(false)
	model.DisableQuitKeybindings()
	model.SetFilteringEnabled(true)
	model.Paginator.Type = paginator.Arabic
	return &View{
		resource:  resource,
		registry:  registry,
		list:      model,
		columns:   columns,
		sortLabel: sortMode(resource),
	}
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
				if next := v.forwardView(selected.data, key.String()); next != nil {
					return viewstate.Update{
						Action: viewstate.Push,
						Next:   next,
					}
				}
				return viewstate.Update{
					Action: viewstate.Push,
					Next:   detailview.New(selected.data, v.resource, v.registry),
				}
			}
		case "s":
			if sortable, ok := v.resource.(resources.ToggleSortable); ok {
				sortable.ToggleSort()
				v.refreshItems()
				v.sortLabel = sortable.SortMode()
				return viewstate.Update{Action: viewstate.None, Next: v}
			}
		case "v":
			if cycler, ok := v.resource.(resources.ScenarioCycler); ok {
				cycler.CycleScenario()
				v.refreshItems()
				return viewstate.Update{Action: viewstate.None, Next: v}
			}
		case "r":
			if selected, ok := v.list.SelectedItem().(item); ok {
				return viewstate.Update{
					Action: viewstate.Push,
					Next:   relatedview.New(selected.data, v.resource, v.registry),
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

	header := "  " + headerRow(v.columns)
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

	view := strings.Join(out, "\n")
	if banner := v.bannerMessage(); banner != "" {
		view = style.Warning.Render(banner) + "\n" + view
	}
	if len(v.list.VisibleItems()) == 0 {
		return view + "\n\n" + style.Muted.Render(v.emptyMessage())
	}
	return view
}

func (v *View) Breadcrumb() string {
	return v.resource.Name()
}

func (v *View) Footer() string {
	parts := []string{v.paginationStatus()}
	if v.list.Paginator.TotalPages > 1 && len(v.list.VisibleItems()) > 0 {
		parts = append(parts, "pgup prev-page  pgdn next-page")
	}
	if strings.EqualFold(v.resource.Name(), "workloads") {
		parts = append(parts, "-> pods", "l logs", "r related", "/ filter", "tab view")
		if _, ok := v.resource.(resources.ToggleSortable); ok {
			parts = append(parts, "s sort:"+v.sortLabel)
		}
		if cycler, ok := v.resource.(resources.ScenarioCycler); ok {
			parts = append(parts, "v state:"+cycler.Scenario())
		}
		return strings.Join(parts, "  ")
	}
	parts = append(parts, "L logs", "r related", "/ filter", "esc clear", "? help", "q quit")
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
	switch status {
	case "CrashLoop", "Error", "Failed":
		return style.Error.Render(status)
	case "Pending", "Warning", "Degraded", "Progressing", "Suspended":
		return style.Warning.Render(status)
	default:
		return style.Healthy.Render(status)
	}
}

func headerRow(columns []resources.TableColumn) string {
	headers := make([]string, 0, len(columns))
	for _, col := range columns {
		headers = append(headers, padCell(col.Name, col.Width))
	}
	return strings.Join(headers, " ")
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

func sortMode(resource resources.ResourceType) string {
	if sortable, ok := resource.(resources.ToggleSortable); ok {
		return sortable.SortMode()
	}
	return ""
}

func (v *View) refreshItems() {
	items := v.resource.Items()
	listItems := make([]list.Item, 0, len(items))
	for _, res := range items {
		listItems = append(listItems, item{
			data:   res,
			row:    tableRow(v.resource, res),
			status: res.Status,
			widths: columnWidths(v.columns),
		})
	}
	v.list.SetItems(listItems)
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

func (v *View) forwardView(selected resources.ResourceItem, key string) viewstate.View {
	resourceName := strings.ToLower(v.resource.Name())

	if resourceName == "workloads" {
		// Deterministic path: workloads always drill into owned pods.
		if key == "l" {
			return podpickerview.New(selected)
		}
		return New(resources.NewWorkloadPods(selected), v.registry)
	}

	if strings.HasPrefix(resourceName, "pods") || resourceName == "pods" {
		containers := v.resource.Detail(selected).Containers
		if len(containers) <= 1 {
			return logview.New(selected, v.resource)
		}
		return detailview.NewContainerPicker(selected, v.resource)
	}

	if strings.HasPrefix(resourceName, "backends") {
		podContext := resources.NewWorkloadPods(resources.ResourceItem{Name: "backend", Kind: "DEP"})
		return detailview.New(selected, podContext, v.registry)
	}

	return nil
}
