package resourcebrowser

import (
	"fmt"
	"sort"
	"strings"
	"unicode"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/paginator"
	bubbletea "github.com/charmbracelet/bubbletea"
	"github.com/dloss/podji/internal/resources"
	"github.com/dloss/podji/internal/ui/filterbar"
	"github.com/dloss/podji/internal/ui/listview"
	"github.com/dloss/podji/internal/ui/style"
	"github.com/dloss/podji/internal/ui/viewstate"
)

const columnSeparator = "  "

type entry struct {
	resource   resources.ResourceType
	kind       string
	group      string
	version    string
	namespaced bool
	isCRD      bool
	hotkey     rune
}

type browserItem struct {
	entry  entry
	row    []string
	widths []int
}

func (i browserItem) Title() string {
	cells := make([]string, 0, len(i.row))
	for idx, value := range i.row {
		cells = append(cells, padCell(value, i.widths[idx]))
	}
	return strings.Join(cells, columnSeparator)
}

func (i browserItem) Description() string { return "" }

// FilterValue includes both kind and group so users can filter by either.
func (i browserItem) FilterValue() string {
	return i.entry.kind + " " + i.entry.group
}

type browserColumn struct {
	name  string
	width int
}

// builtinInfo holds display metadata for built-in registry resources.
type builtinInfo struct {
	kind       string
	group      string
	version    string
	namespaced bool
}

var builtinResourceInfo = map[string]builtinInfo{
	"workloads":   {kind: "Workload", group: "podji.io", version: "v1", namespaced: true},
	"pods":        {kind: "Pod", group: "core", version: "v1", namespaced: true},
	"deployments": {kind: "Deployment", group: "apps", version: "v1", namespaced: true},
	"services":    {kind: "Service", group: "core", version: "v1", namespaced: true},
	"configmaps":  {kind: "ConfigMap", group: "core", version: "v1", namespaced: true},
	"secrets":     {kind: "Secret", group: "core", version: "v1", namespaced: true},
	"namespaces":  {kind: "Namespace", group: "core", version: "v1", namespaced: false},
	"nodes":       {kind: "Node", group: "core", version: "v1", namespaced: false},
	"events":      {kind: "Event", group: "core", version: "v1", namespaced: true},
	"contexts":    {kind: "Context", group: "kubeconfig", version: "v1", namespaced: false},
}

// View is the resource browser: a filterable list of all resource types
// (built-ins and CRDs). Selecting an entry navigates to its instance list.
type View struct {
	registry    *resources.Registry
	entries     []entry
	list        list.Model
	columns     []browserColumn
	colWidths   []int
	findMode    bool
	findTargets map[int]bool
}

// New creates the resource browser populated with registry built-ins and the
// supplied CRD metadata.
func New(registry *resources.Registry, crds []resources.CRDMeta) *View {
	entries := buildEntries(registry, crds)
	cols := browserColumns()
	rows := entryRows(entries)
	widths := columnWidthsForRows(cols, rows, 0)

	v := &View{
		registry:  registry,
		entries:   entries,
		columns:   cols,
		colWidths: widths,
	}
	delegate := newBrowserDelegate(&v.findMode, &v.findTargets)
	model := list.New(buildListItems(entries, rows, widths), delegate, 0, 0)
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
		case "enter", "l", "right":
			if selected, ok := v.list.SelectedItem().(browserItem); ok {
				lv := listview.New(selected.entry.resource, v.registry)
				return viewstate.Update{Action: viewstate.Push, Next: lv}
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

	header := "  " + buildHeaderRow(v.columns, v.colWidths)
	out := make([]string, len(lines))
	for i := range out {
		out[i] = ""
	}
	if len(out) > 1 {
		out[1] = header
	}
	dst := 2
	for _, line := range lines[dataStart:] {
		if dst >= len(out) {
			break
		}
		out[dst] = line
		dst++
	}
	if len(v.list.VisibleItems()) == 0 {
		msgRow := 3
		if msgRow >= len(out) {
			msgRow = len(out) - 1
		}
		if msgRow >= 0 {
			if v.list.IsFiltered() {
				out[msgRow] = style.Muted.Render("  No resource types match the active filter. Press esc to clear.")
			} else {
				out[msgRow] = style.Muted.Render("  No resource types found.")
			}
		}
	}
	return filterbar.Append(strings.Join(out, "\n"), v.list)
}

func (v *View) Breadcrumb() string { return "resources" }

func (v *View) Footer() string {
	var indicators []style.Binding
	if v.findMode {
		indicators = append(indicators, style.B("f", "…"))
	}
	if v.list.IsFiltered() {
		indicators = append(indicators, style.B("filter", strings.TrimSpace(v.list.FilterValue())))
	}
	line1 := style.StatusFooter(indicators, v.paginationStatus(), v.list.Width())
	line2 := style.ActionFooter([]style.Binding{
		style.B("/", "filter"),
		style.B("f", "find"),
	}, v.list.Width())
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
		return fmt.Sprintf("Showing %d-%d of %d filtered (%d total)",
			start+1, end, totalVisible, len(v.list.Items()))
	}
	return fmt.Sprintf("Showing %d-%d of %d", start+1, end, totalVisible)
}

func (v *View) refreshItems() {
	rows := entryRows(v.entries)
	v.colWidths = columnWidthsForRows(v.columns, rows, v.list.Width()-2)
	selected := v.list.Index()
	v.list.SetItems(buildListItems(v.entries, rows, v.colWidths))
	if selected >= 0 && selected < len(v.entries) {
		v.list.Select(selected)
	}
}

func (v *View) jumpToChar(r rune) {
	target := unicode.ToLower(r)
	for i, li := range v.list.VisibleItems() {
		if it, ok := li.(browserItem); ok {
			if len(it.entry.kind) > 0 && unicode.ToLower([]rune(it.entry.kind)[0]) == target {
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
		if it, ok := li.(browserItem); ok {
			if len(it.entry.kind) > 0 {
				ch := unicode.ToLower([]rune(it.entry.kind)[0])
				if !seen[ch] {
					seen[ch] = true
					targets[i] = true
				}
			}
		}
	}
	return targets
}

func buildEntries(registry *resources.Registry, crds []resources.CRDMeta) []entry {
	var entries []entry
	for _, res := range registry.Resources() {
		info, ok := builtinResourceInfo[strings.ToLower(res.Name())]
		if !ok {
			info = builtinInfo{
				kind:    titleCase(res.Name()),
				group:   "core",
				version: "v1",
			}
		}
		entries = append(entries, entry{
			resource:   res,
			kind:       info.kind,
			group:      info.group,
			version:    info.version,
			namespaced: info.namespaced,
			isCRD:      false,
			hotkey:     res.Key(),
		})
	}
	for _, meta := range crds {
		entries = append(entries, entry{
			resource:   resources.NewCRDResource(meta),
			kind:       meta.Kind,
			group:      meta.Group,
			version:    meta.Version,
			namespaced: meta.Namespaced,
			isCRD:      true,
		})
	}
	sort.SliceStable(entries, func(i, j int) bool {
		if entries[i].isCRD != entries[j].isCRD {
			return !entries[i].isCRD
		}
		return strings.ToLower(entries[i].kind) < strings.ToLower(entries[j].kind)
	})
	return entries
}

func entryRow(e entry) []string {
	scope := "Namespaced"
	if !e.namespaced {
		scope = "Cluster"
	}
	source := "CRD"
	if !e.isCRD {
		source = "Built-in"
	}
	hotkey := ""
	if e.hotkey != 0 {
		hotkey = "[" + string(e.hotkey) + "]"
	}
	return []string{e.kind, e.group, e.version, scope, source, hotkey}
}

func entryRows(entries []entry) [][]string {
	rows := make([][]string, 0, len(entries))
	for _, e := range entries {
		rows = append(rows, entryRow(e))
	}
	return rows
}

func browserColumns() []browserColumn {
	return []browserColumn{
		{name: "KIND", width: 20},
		{name: "GROUP", width: 28},
		{name: "VERSION", width: 8},
		{name: "SCOPE", width: 10},
		{name: "SOURCE", width: 8},
		{name: "KEY", width: 3},
	}
}

func buildHeaderRow(cols []browserColumn, widths []int) string {
	headers := make([]string, 0, len(cols))
	for idx, col := range cols {
		width := col.width
		if idx < len(widths) && widths[idx] > 0 {
			width = widths[idx]
		}
		headers = append(headers, padCell(col.name, width))
	}
	return strings.Join(headers, columnSeparator)
}

func buildListItems(entries []entry, rows [][]string, widths []int) []list.Item {
	listItems := make([]list.Item, 0, len(entries))
	for idx, e := range entries {
		listItems = append(listItems, browserItem{
			entry:  e,
			row:    rows[idx],
			widths: widths,
		})
	}
	return listItems
}

func columnWidthsForRows(cols []browserColumn, rows [][]string, availableWidth int) []int {
	if len(cols) == 0 {
		return nil
	}
	widths := make([]int, len(cols))
	for idx, col := range cols {
		maxContent := 0
		for _, row := range rows {
			if idx >= len(row) {
				continue
			}
			w := len([]rune(strings.TrimSpace(row[idx])))
			if w > maxContent {
				maxContent = w
			}
		}
		headerWidth := len([]rune(strings.TrimSpace(col.name)))
		width := headerWidth
		if maxContent > width {
			width = maxContent
		}
		if width < 1 {
			width = 1
		}
		widths[idx] = width
	}
	if availableWidth <= 0 {
		return widths
	}
	availableContent := availableWidth - (len(cols)-1)*len(columnSeparator)
	if availableContent < len(cols) {
		availableContent = len(cols)
	}
	sum := 0
	for _, w := range widths {
		sum += w
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
	return widths
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

func titleCase(value string) string {
	if value == "" {
		return value
	}
	runes := []rune(value)
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}

func singleRune(key bubbletea.KeyMsg) rune {
	if key.Type == bubbletea.KeyRunes && len(key.Runes) == 1 {
		return key.Runes[0]
	}
	return 0
}
