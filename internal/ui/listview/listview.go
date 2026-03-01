package listview

import (
	"fmt"
	"os/exec"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/paginator"
	bubbletea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/dloss/podji/internal/columnconfig"
	"github.com/dloss/podji/internal/resources"
	"github.com/dloss/podji/internal/ui/describeview"
	"github.com/dloss/podji/internal/ui/detailview"
	"github.com/dloss/podji/internal/ui/eventview"
	"github.com/dloss/podji/internal/ui/filterbar"
	"github.com/dloss/podji/internal/ui/logview"
	"github.com/dloss/podji/internal/ui/style"
	"github.com/dloss/podji/internal/ui/viewstate"
	"github.com/dloss/podji/internal/ui/yamlview"
)

const columnSeparator = "  "

// OpenColumnPickerMsg signals app.go to open the column picker overlay.
type OpenColumnPickerMsg struct {
	ResourceName string
	Pool         []resources.TableColumn // resource-defined columns (normal + wide extras)
	LabelPool    []resources.TableColumn // dynamic label-derived columns
	Current      []string                // currently active column IDs
}

type clearCopiedMsg struct{}

func clearCopiedCmd() bubbletea.Cmd {
	return func() bubbletea.Msg {
		time.Sleep(1500 * time.Millisecond)
		return clearCopiedMsg{}
	}
}

type clearExecResultMsg struct{}

func clearExecResultCmd() bubbletea.Cmd {
	return func() bubbletea.Msg {
		time.Sleep(2500 * time.Millisecond)
		return clearExecResultMsg{}
	}
}

type shellExecResultMsg struct{ err error }

type executeState int

const (
	execNone executeState = iota
	execMenu
	execConfirmDelete
	execConfirmRestart
	execInputScale
	execInputPortFwd
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
	resource     resources.ResourceType
	registry     *resources.Registry
	list         list.Model
	columns      []resources.TableColumn
	colWidths    []int
	wideMode     bool
	labelPool    []resources.TableColumn
	sortPickMode bool
	findMode     bool
	findTargets  map[int]bool
	copyMode     bool
	copiedMsg    string
	execState    executeState
	execInput    string
	execResult   string
}

func New(resource resources.ResourceType, registry *resources.Registry) *View {
	items := resource.Items()
	labelPool := labelColumnsFromItems(items)
	pool := buildColumnPool(resource, labelPool)
	columns := columnconfig.Default().Get(resource.Name(), pool)
	rows := assembleRows(resource, false, columns, items)
	firstHeader := strings.ToUpper(resources.SingularName(breadcrumbLabel(resource.Name())))
	widths := columnWidthsForRows(columns, rows, 0, firstHeader)
	listItems := makeListItems(items, rows, widths)

	v := &View{
		resource:  resource,
		registry:  registry,
		columns:   columns,
		colWidths: widths,
		labelPool: labelPool,
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
	if _, ok := msg.(clearCopiedMsg); ok {
		v.copiedMsg = ""
		return viewstate.Update{Action: viewstate.None, Next: v}
	}

	if _, ok := msg.(clearExecResultMsg); ok {
		v.execResult = ""
		return viewstate.Update{Action: viewstate.None, Next: v}
	}

	if msg, ok := msg.(shellExecResultMsg); ok {
		if msg.err != nil {
			v.execResult = "exec failed: " + msg.err.Error()
		} else {
			v.execResult = "exec session ended"
		}
		return viewstate.Update{Action: viewstate.None, Next: v, Cmd: clearExecResultCmd()}
	}

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

		if v.sortPickMode {
			v.sortPickMode = false
			if key.String() != "esc" {
				if sortable, ok := v.resource.(resources.Sortable); ok {
					if r := singleRune(key); r != 0 {
						if colIdx, desc, ok := numericSortKey(r); ok {
							mode := sortModeForColumn(v.resource, v.columns, colIdx)
							if mode != "" {
								sortable.SetSort(mode, desc)
								v.refreshItems()
							}
						} else {
							resourceLabel := resources.SingularName(breadcrumbLabel(v.resource.Name()))
							skeys := sortable.SortKeys()
							chars := sortDisplayedChars(v.resource, v.columns, skeys, resourceLabel)
							lc := unicode.ToLower(r)
							desc := unicode.IsUpper(r)
							for i, ch := range chars {
								if ch == lc {
									sortable.SetSort(skeys[i].Mode, desc)
									v.refreshItems()
									break
								}
							}
						}
					}
				}
			}
			return viewstate.Update{Action: viewstate.None, Next: v}
		}

		if v.copyMode {
			v.copyMode = false
			if selected, ok := v.list.SelectedItem().(item); ok {
				switch key.String() {
				case "n":
					v.copiedMsg = "copied: " + selected.data.Name
					return viewstate.Update{Action: viewstate.None, Next: v, Cmd: clearCopiedCmd()}
				case "k":
					kind := resources.SingularName(breadcrumbLabel(v.resource.Name()))
					v.copiedMsg = "copied: " + kind + "/" + selected.data.Name
					return viewstate.Update{Action: viewstate.None, Next: v, Cmd: clearCopiedCmd()}
				case "p":
					v.copiedMsg = "copied: -n " + resources.ActiveNamespace + " " + selected.data.Name
					return viewstate.Update{Action: viewstate.None, Next: v, Cmd: clearCopiedCmd()}
				}
			}
			return viewstate.Update{Action: viewstate.None, Next: v}
		}

		// Execute mode: sub-menu (x was pressed; pick an operation).
		if v.execState == execMenu {
			switch key.String() {
			case "esc":
				v.execState = execNone
			case "d":
				v.execState = execNone
				v.execState = execConfirmDelete
			case "r":
				if v.supportsRestart() {
					v.execState = execNone
					v.execState = execConfirmRestart
				}
			case "s":
				if v.supportsScale() {
					v.execState = execNone
					v.execState = execInputScale
					v.execInput = v.currentReplicas()
				}
			case "f":
				if v.supportsPortFwd() {
					v.execState = execNone
					v.execState = execInputPortFwd
					v.execInput = "8080:8080"
				}
			case "x":
				if v.supportsShellExec() {
					v.execState = execNone
					return viewstate.Update{Action: viewstate.None, Next: v, Cmd: v.shellExecCmd()}
				}
			}
			return viewstate.Update{Action: viewstate.None, Next: v}
		}

		// Execute mode: delete / restart confirmation.
		if v.execState == execConfirmDelete || v.execState == execConfirmRestart {
			op := v.execState
			switch key.String() {
			case "y":
				v.execState = execNone
				label := v.execTargetLabel()
				var resultMsg string
				if op == execConfirmDelete {
					resultMsg = "deleted " + label + " (simulated)"
				} else {
					resultMsg = "restarted " + label + " (simulated)"
				}
				v.execResult = resultMsg
				return viewstate.Update{Action: viewstate.None, Next: v, Cmd: clearExecResultCmd()}
			case "esc":
				v.execState = execNone
			}
			return viewstate.Update{Action: viewstate.None, Next: v}
		}

		// Execute mode: scale / port-forward text input.
		if v.execState == execInputScale || v.execState == execInputPortFwd {
			switch key.String() {
			case "enter":
				label := v.execTargetLabel()
				var resultMsg string
				if v.execState == execInputScale {
					resultMsg = "scaled " + label + " to " + v.execInput + " (simulated)"
				} else {
					resultMsg = "port-fwd " + label + " " + v.execInput + " (simulated)"
				}
				v.execState = execNone
				v.execResult = resultMsg
				return viewstate.Update{Action: viewstate.None, Next: v, Cmd: clearExecResultCmd()}
			case "esc":
				v.execState = execNone
			case "backspace", "ctrl+h":
				runes := []rune(v.execInput)
				if len(runes) > 0 {
					v.execInput = string(runes[:len(runes)-1])
				}
			default:
				if key.Type == bubbletea.KeyRunes && len(key.Runes) == 1 {
					r := key.Runes[0]
					if v.execState == execInputScale {
						if r >= '0' && r <= '9' {
							v.execInput += string(r)
						}
					} else {
						v.execInput += string(r)
					}
				}
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
				action, next := v.forwardView(selected.data, key.String())
				if action == viewstate.OpenRelated {
					return viewstate.Update{Action: viewstate.OpenRelated}
				}
				if next != nil {
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
		case "w":
			// Wide mode toggle (only if resource implements WideResource).
			if _, ok := v.resource.(resources.WideResource); ok {
				v.wideMode = !v.wideMode
				v.refreshColumns()
				return viewstate.Update{Action: viewstate.None, Next: v}
			}
		case "p":
			// Column picker (exit wide mode first if active).
			if _, ok := v.resource.(resources.TableResource); ok {
				if v.wideMode {
					v.wideMode = false
					v.refreshColumns()
				}
				pool := buildColumnPool(v.resource, v.labelPool)
				current := columnIDs(v.columns)
				resourceName := v.resource.Name()
				labelPool := v.labelPool
				return viewstate.Update{
					Action: viewstate.None,
					Next:   v,
					Cmd: func() bubbletea.Msg {
						return OpenColumnPickerMsg{
							ResourceName: resourceName,
							Pool:         pool,
							LabelPool:    labelPool,
							Current:      current,
						}
					},
				}
			}
		case "s":
			if _, ok := v.resource.(resources.Sortable); ok {
				v.sortPickMode = true
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
			// Handled by app.go (opens/closes the side panel).
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
		case "c":
			if selected, ok := v.list.SelectedItem().(item); ok && selected.data.Name != "" {
				v.copyMode = true
			}
			return viewstate.Update{Action: viewstate.None, Next: v}
		case "x":
			if selected, ok := v.list.SelectedItem().(item); ok && selected.data.Name != "" {
				v.execState = execMenu
			}
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
	mode := sortMode(v.resource)
	indicator := sortIndicatorSymbol(v.resource)
	activeSortIdx := -1
	if !isDefaultSort(v.resource) {
		activeSortIdx = activeSortColumn(v.resource, v.columns, mode)
	}
	headerPrefix := "  "
	if activeSortIdx == 0 {
		headerPrefix = " " + indicator
	}
	header := headerPrefix + headerRowWithHint(v.columns, v.colWidths, label, childHint, activeSortIdx, indicator)
	// Keep the same line budget as the base list view so the footer doesn't
	// jump, then place our table header into the first two rows.
	out := make([]string, len(lines))
	for i := range out {
		out[i] = ""
	}
	out[0] = ""
	if len(out) > 1 {
		out[1] = header
	}
	dst := 2
	hasVisibleItems := len(v.list.VisibleItems()) > 0
	for _, line := range lines[dataStart:] {
		if !hasVisibleItems && strings.TrimSpace(ansi.Strip(line)) == "No items." {
			continue
		}
		if dst >= len(out) {
			break
		}
		out[dst] = line
		dst++
	}
	if banner := v.bannerMessage(); banner != "" {
		// Reserve one content row for the banner while preserving line count.
		for i := len(out) - 1; i > 0; i-- {
			out[i] = out[i-1]
		}
		out[0] = style.Warning.Render(banner)
	}
	if len(v.list.VisibleItems()) == 0 {
		// Render empty-state text inside the existing budget.
		msgRow := 3
		if msgRow >= len(out) {
			msgRow = len(out) - 1
		}
		if msgRow >= 0 {
			out[msgRow] = style.Muted.Render("  " + v.emptyMessage())
		}
	}
	view := strings.Join(out, "\n")
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
	if cycler, ok := v.resource.(resources.ScenarioCycler); ok && cycler.Scenario() != "normal" {
		indicators = append(indicators, style.B("state", cycler.Scenario()))
	}
	if v.wideMode {
		indicators = append(indicators, style.B("wide", ""))
	}
	if columnconfig.Default().IsCustom(v.resource.Name()) && !v.wideMode {
		indicators = append(indicators, style.B("columns", "custom"))
	}
	if v.findMode {
		indicators = append(indicators, style.B("f", "…"))
	}
	if v.list.IsFiltered() {
		indicators = append(indicators, style.B("filter", strings.TrimSpace(v.list.FilterValue())))
	}
	if v.copiedMsg != "" {
		indicators = append(indicators, style.B(v.copiedMsg, ""))
	}
	if v.execResult != "" {
		indicators = append(indicators, style.B(v.execResult, ""))
	}
	line1 := style.StatusFooter(indicators, v.paginationStatus(), v.list.Width())

	// Line 2: mode prompt or normal actions.
	var line2 string
	if v.copyMode {
		copyLabel := style.FooterKey.Render("copy")
		opts := style.FormatBindings([]style.Binding{
			style.B("n", "name"),
			style.B("k", "kind/name"),
			style.B("p", "-n ns name"),
			style.B("esc", "cancel"),
		})
		line2 = copyLabel + "  " + opts
		if v.list.Width() > 0 {
			line2 = ansi.Truncate(line2, v.list.Width()-2, "…")
		}
	} else if v.execState == execMenu {
		execLabel := style.FooterKey.Render("execute")
		var opts []style.Binding
		opts = append(opts, style.B("d", "delete"))
		if v.supportsRestart() {
			opts = append(opts, style.B("r", "restart"))
		}
		if v.supportsScale() {
			opts = append(opts, style.B("s", "scale"))
		}
		if v.supportsPortFwd() {
			opts = append(opts, style.B("f", "port-fwd"))
		}
		if v.supportsShellExec() {
			opts = append(opts, style.B("x", "shell"))
		}
		opts = append(opts, style.B("esc", "cancel"))
		line2 = execLabel + "  " + style.FormatBindings(opts)
		if v.list.Width() > 0 {
			line2 = ansi.Truncate(line2, v.list.Width()-2, "…")
		}
	} else if v.execState == execConfirmDelete || v.execState == execConfirmRestart {
		var opName string
		if v.execState == execConfirmDelete {
			opName = "delete"
		} else {
			opName = "restart"
		}
		opLabel := style.FooterKey.Render(opName)
		target := style.FooterLabel.Render(v.execTargetLabel() + "?")
		opts := style.FormatBindings([]style.Binding{
			style.B("y", "confirm"),
			style.B("esc", "cancel"),
		})
		line2 = opLabel + " " + target + "  " + opts
		if v.list.Width() > 0 {
			line2 = ansi.Truncate(line2, v.list.Width()-2, "…")
		}
	} else if v.execState == execInputScale {
		scaleLabel := style.FooterKey.Render("scale")
		target := style.FooterLabel.Render(v.execTargetLabel())
		prompt := style.FooterLabel.Render("  replicas: ")
		inputVal := style.FooterKey.Render(v.execInput + "█")
		opts := "  " + style.FormatBindings([]style.Binding{
			style.B("enter", "confirm"),
			style.B("esc", "cancel"),
		})
		line2 = scaleLabel + " " + target + prompt + inputVal + opts
		if v.list.Width() > 0 {
			line2 = ansi.Truncate(line2, v.list.Width()-2, "…")
		}
	} else if v.execState == execInputPortFwd {
		fwdLabel := style.FooterKey.Render("port-fwd")
		target := style.FooterLabel.Render(v.execTargetLabel())
		prompt := style.FooterLabel.Render("  ports: ")
		inputVal := style.FooterKey.Render(v.execInput + "█")
		opts := "  " + style.FormatBindings([]style.Binding{
			style.B("enter", "confirm"),
			style.B("esc", "cancel"),
		})
		line2 = fwdLabel + " " + target + prompt + inputVal + opts
		if v.list.Width() > 0 {
			line2 = ansi.Truncate(line2, v.list.Width()-2, "…")
		}
	} else if v.sortPickMode {
		sortLabel := style.FooterKey.Render("sort")
		var opts []style.Binding
		if sortable, ok := v.resource.(resources.Sortable); ok {
			resourceLabel := resources.SingularName(breadcrumbLabel(v.resource.Name()))
			skeys := sortable.SortKeys()
			chars := sortDisplayedChars(v.resource, v.columns, skeys, resourceLabel)
			for i, sk := range skeys {
				ch := chars[i]
				lower := string(ch)
				upper := string(unicode.ToUpper(ch))
				colIdx := activeSortColumn(v.resource, v.columns, sk.Mode)
				label := strings.ToLower(displayedColHeader(v.columns, colIdx, resourceLabel))
				if label == "" {
					label = sk.Label
				}
				opts = append(opts, style.B(lower+"/"+upper, label))
			}
		}
		opts = append(opts, style.B("1-9/⇧", "col"))
		opts = append(opts, style.B("esc", "cancel"))
		line2 = sortLabel + "  " + style.FormatBindings(opts)
		if v.list.Width() > 0 {
			line2 = ansi.Truncate(line2, v.list.Width()-2, "…")
		}
	} else {
		var actions []style.Binding
		isWorkloads := strings.EqualFold(v.resource.Name(), "workloads")
		isContainers := strings.EqualFold(v.resource.Name(), "containers")

		actions = append(actions, style.B("/", "filter"))
		if _, ok := v.resource.(resources.Sortable); ok {
			actions = append(actions, style.B("s", "sort"))
		}
		if isWorkloads {
			if _, ok := v.resource.(resources.ScenarioCycler); ok {
				actions = append(actions, style.B("v", "state"))
			}
		}
		if !isContainers {
			actions = append(actions, style.B("r", "related"))
		}
		if _, ok := v.resource.(resources.TableResource); ok {
			actions = append(actions, style.B("p", "columns"))
		}
		if _, ok := v.resource.(resources.WideResource); ok {
			if v.wideMode {
				actions = append(actions, style.B("w", "[wide]"))
			} else {
				actions = append(actions, style.B("w", "wide"))
			}
		}
		actions = append(actions, style.B("c", "copy"))
		actions = append(actions, style.B("x", "execute"))
		line2 = style.ActionFooter(actions, v.list.Width())
	}

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
	return v.list.SettingFilter() || v.list.IsFiltered() || v.findMode || v.copyMode || v.execState != execNone || v.sortPickMode
}

// SelectedItem returns the currently highlighted resource item.
func (v *View) SelectedItem() resources.ResourceItem {
	if selected, ok := v.list.SelectedItem().(item); ok {
		return selected.data
	}
	return resources.ResourceItem{}
}

// SelectedBodyRow returns the 0-based line index within the body rendered by
// View() at which the selected row appears. Returns -1 when no items are
// visible. Used by the app layer to anchor overlays near the selection.
func (v *View) SelectedBodyRow() int {
	visible := v.list.VisibleItems()
	if len(visible) == 0 {
		return -1
	}
	start, _ := v.list.Paginator.GetSliceBounds(len(visible))
	pageIdx := v.list.Index() - start
	if pageIdx < 0 {
		return -1
	}
	// Body layout: out[0]=blank, out[1]=table header, out[2+i]=data row i.
	// A banner shifts all rows down by 1.
	offset := 2
	if v.bannerMessage() != "" {
		offset = 3
	}
	return offset + pageIdx
}

// Resource returns the underlying resource type for this view.
func (v *View) Resource() resources.ResourceType {
	return v.resource
}

// ApplyColumnConfig is called by app.go after the column picker confirms a selection.
func (v *View) ApplyColumnConfig(resourceName string, visible []string) {
	if resourceName != v.resource.Name() {
		return
	}
	columnconfig.Default().Set(resourceName, visible)
	pool := buildColumnPool(v.resource, v.labelPool)
	v.columns = columnconfig.Default().Get(resourceName, pool)
	v.refreshItems()
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

// ForwardViewForCommand exposes subview routing for app-level command handling.
func (v *View) ForwardViewForCommand(item resources.ResourceItem, subview string) (viewstate.Action, viewstate.View) {
	key := subview
	switch subview {
	case "yaml":
		key = "y"
	case "events":
		key = "e"
	case "describe":
		key = "d"
	case "logs":
		key = "l"
	}
	return v.forwardView(item, key)
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
	return headerRowWithHint(columns, nil, firstLabel, "", -1, "▲")
}

func headerRowWithHint(
	columns []resources.TableColumn,
	widths []int,
	firstLabel string,
	childHint string,
	activeSortIdx int,
	indicator string,
) string {
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
	if len(headers) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString(headers[0])
	for idx := 1; idx < len(headers); idx++ {
		sep := columnSeparator
		if idx == activeSortIdx {
			sep = " " + indicator
		}
		b.WriteString(sep)
		b.WriteString(headers[idx])
	}
	return b.String()
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
		{ID: "name", Name: "NAME", Width: 48, Default: true},
		{ID: "status", Name: "STATUS", Width: 12, Default: true},
		{ID: "ready", Name: "READY", Width: 7, Default: true},
		{ID: "restarts", Name: "RESTARTS", Width: 14, Default: true},
		{ID: "age", Name: "AGE", Width: 6, Default: true},
	}
}

func tableRowMap(resource resources.ResourceType, res resources.ResourceItem) map[string]string {
	if table, ok := resource.(resources.TableResource); ok {
		return table.TableRow(res)
	}
	return map[string]string{
		"name":     res.Name,
		"status":   res.Status,
		"ready":    res.Ready,
		"restarts": res.Restarts,
		"age":      res.Age,
	}
}

// buildColumnPool returns the full set of columns available for a resource:
// normal columns + wide-only extras + label columns.
func buildColumnPool(resource resources.ResourceType, labelPool []resources.TableColumn) []resources.TableColumn {
	pool := tableColumns(resource)

	if wide, ok := resource.(resources.WideResource); ok {
		poolIDs := make(map[string]bool, len(pool))
		for _, col := range pool {
			poolIDs[col.ID] = true
		}
		for _, col := range wide.TableColumnsWide() {
			if !poolIDs[col.ID] {
				pool = append(pool, col)
			}
		}
	}

	pool = append(pool, labelPool...)
	return pool
}

// assembleRow builds a []string row for a single item.
// Always uses TableRowWide when available so that wide columns selected via the
// picker are populated even when wideMode is off. wideMode only controls which
// columns are visible, not the data source.
func assembleRow(resource resources.ResourceType, wideMode bool, columns []resources.TableColumn, res resources.ResourceItem) []string {
	var rowMap map[string]string
	if wide, ok := resource.(resources.WideResource); ok {
		rowMap = wide.TableRowWide(res)
	}
	if rowMap == nil {
		rowMap = tableRowMap(resource, res)
	}

	row := make([]string, len(columns))
	for i, col := range columns {
		if strings.HasPrefix(col.ID, "label:") {
			labelKey := strings.TrimPrefix(col.ID, "label:")
			row[i] = res.Labels[labelKey]
		} else {
			row[i] = rowMap[col.ID]
		}
	}
	return row
}

// assembleRows builds all rows for a resource's item list.
func assembleRows(resource resources.ResourceType, wideMode bool, columns []resources.TableColumn, items []resources.ResourceItem) [][]string {
	rows := make([][]string, 0, len(items))
	for _, res := range items {
		rows = append(rows, assembleRow(resource, wideMode, columns, res))
	}
	return rows
}

// makeListItems creates bubbletea list items from resource items and pre-computed rows/widths.
func makeListItems(items []resources.ResourceItem, rows [][]string, widths []int) []list.Item {
	listItems := make([]list.Item, 0, len(items))
	for idx, res := range items {
		var row []string
		if idx < len(rows) {
			row = rows[idx]
		}
		listItems = append(listItems, item{
			data:   res,
			row:    row,
			status: res.Status,
			widths: widths,
		})
	}
	return listItems
}

func columnWidths(columns []resources.TableColumn) []int {
	widths := make([]int, 0, len(columns))
	for _, col := range columns {
		widths = append(widths, col.Width)
	}
	return widths
}

func columnWidthsForRows(columns []resources.TableColumn, rows [][]string, availableWidth int, firstHeader string) []int {
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

// displayedColHeader returns the column header as shown in the table header row.
// Column 0 with Name "NAME" shows the resource's singular label instead.
func displayedColHeader(columns []resources.TableColumn, idx int, resourceLabel string) string {
	if idx < 0 || idx >= len(columns) {
		return ""
	}
	col := columns[idx]
	if idx == 0 && strings.EqualFold(strings.TrimSpace(col.Name), "name") {
		return strings.ToUpper(resourceLabel)
	}
	return col.Name
}

// sortDisplayedChars computes sort key characters for all keys, derived from
// displayed column headers. When two keys would derive the same character
// (e.g. two modes mapped to the same column), later ones fall back to their
// static SortKey.Char. Returns a parallel slice of runes.
func sortDisplayedChars(resource resources.ResourceType, columns []resources.TableColumn, keys []resources.SortKey, resourceLabel string) []rune {
	result := make([]rune, len(keys))
	used := make(map[rune]bool)
	for i, sk := range keys {
		colIdx := activeSortColumn(resource, columns, sk.Mode)
		hdr := displayedColHeader(columns, colIdx, resourceLabel)
		var ch rune
		if hdr != "" {
			if runes := []rune(strings.ToLower(hdr)); len(runes) > 0 {
				candidate := runes[0]
				if !used[candidate] {
					ch = candidate
				}
			}
		}
		if ch == 0 {
			ch = sk.Char
		}
		used[ch] = true
		result[i] = ch
	}
	return result
}

// sortModeForColumn returns the sort mode for the given 0-based column index,
// or "" if no sort key maps to that column.
func sortModeForColumn(resource resources.ResourceType, columns []resources.TableColumn, colIdx int) string {
	sortable, ok := resource.(resources.Sortable)
	if !ok {
		return ""
	}
	for _, sk := range sortable.SortKeys() {
		if activeSortColumn(resource, columns, sk.Mode) == colIdx {
			return sk.Mode
		}
	}
	return ""
}

// numericSortKey interprets digit and US-keyboard shift-digit runes as
// (0-based column index, descending, ok). Digits 1-9 map to columns 0-8
// ascending; !@#$%^&*( (shift+1-9) map to columns 0-8 descending.
func numericSortKey(r rune) (int, bool, bool) {
	if r >= '1' && r <= '9' {
		return int(r - '1'), false, true
	}
	const shiftDigits = "!@#$%^&*("
	for i, s := range shiftDigits {
		if r == s {
			return i, true, true
		}
	}
	return 0, false, false
}

func sortMode(resource resources.ResourceType) string {
	if sortable, ok := resource.(resources.Sortable); ok {
		return sortable.SortMode()
	}
	return ""
}

func sortIndicatorSymbol(resource resources.ResourceType) string {
	if sortable, ok := resource.(resources.Sortable); ok && sortable.SortDesc() {
		return "▼"
	}
	return "▲"
}

func activeSortColumn(resource resources.ResourceType, columns []resources.TableColumn, mode string) int {
	if len(columns) == 0 || mode == "" {
		return -1
	}

	switch mode {
	case "name":
		return 0
	case "age":
		return firstColumnWithID(columns, "age")
	case "kind":
		if idx := firstColumnWithID(columns, "kind"); idx >= 0 {
			return idx
		}
		return firstColumnWithID(columns, "type")
	case "status":
		if idx := firstColumnWithID(columns, "status"); idx >= 0 {
			return idx
		}
		// Events surface severity in TYPE and also support status sort.
		if resource != nil && strings.EqualFold(resource.Name(), "events") {
			return firstColumnWithID(columns, "type")
		}
	}
	return -1
}

func isDefaultSort(resource resources.ResourceType) bool {
	sortable, ok := resource.(resources.Sortable)
	if !ok {
		return true
	}
	keys := sortable.SortKeys()
	if len(keys) == 0 {
		return true
	}
	return sortable.SortMode() == keys[0].Mode && !sortable.SortDesc()
}

func firstColumnWithID(columns []resources.TableColumn, id string) int {
	for idx, col := range columns {
		if col.ID == id {
			return idx
		}
	}
	return -1
}

// refreshColumns swaps columns (and reloads items) based on current wideMode.
func (v *View) refreshColumns() {
	if v.wideMode {
		if wide, ok := v.resource.(resources.WideResource); ok {
			v.columns = wide.TableColumnsWide()
			v.refreshItems()
			return
		}
	}
	// Normal mode: apply column config.
	pool := buildColumnPool(v.resource, v.labelPool)
	v.columns = columnconfig.Default().Get(v.resource.Name(), pool)
	v.refreshItems()
}

func (v *View) refreshItems() {
	items := v.resource.Items()

	// Refresh label pool from current items.
	v.labelPool = labelColumnsFromItems(items)

	// In normal mode, re-apply column config (label pool may have changed).
	if !v.wideMode {
		pool := buildColumnPool(v.resource, v.labelPool)
		v.columns = columnconfig.Default().Get(v.resource.Name(), pool)
	}

	rows := assembleRows(v.resource, v.wideMode, v.columns, items)
	firstHeader := strings.ToUpper(resources.SingularName(breadcrumbLabel(v.resource.Name())))
	v.colWidths = columnWidthsForRows(v.columns, rows, v.list.Width()-2, firstHeader)
	listItems := makeListItems(items, rows, v.colWidths)
	selected := v.list.Index()
	v.list.SetItems(listItems)
	if selected >= 0 && selected < len(listItems) {
		v.list.Select(selected)
	}
}

// columnIDs returns the IDs of the given columns in order.
func columnIDs(columns []resources.TableColumn) []string {
	ids := make([]string, len(columns))
	for i, col := range columns {
		ids[i] = col.ID
	}
	return ids
}

// labelColumnsFromItems discovers unique label keys across all items and returns
// synthetic TableColumn entries for Phase 3 label columns.
func labelColumnsFromItems(items []resources.ResourceItem) []resources.TableColumn {
	seen := make(map[string]bool)
	var keys []string
	for _, item := range items {
		for k := range item.Labels {
			if !seen[k] {
				seen[k] = true
				keys = append(keys, k)
			}
		}
	}
	sort.Strings(keys)
	cols := make([]resources.TableColumn, 0, len(keys))
	for _, k := range keys {
		width := len(k)
		if width < 12 {
			width = 12
		}
		if width > 20 {
			width = 20
		}
		cols = append(cols, resources.TableColumn{
			ID:      "label:" + k,
			Name:    strings.ToUpper(k),
			Width:   width,
			Default: false,
		})
	}
	return cols
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

func (v *View) forwardView(selected resources.ResourceItem, key string) (viewstate.Action, viewstate.View) {
	resourceName := strings.ToLower(v.resource.Name())

	if resourceName == "workloads" {
		pods := resources.NewWorkloadPods(selected, v.registry)
		items := pods.Items()
		if len(items) == 0 {
			return viewstate.OpenRelated, nil
		}
		if key == "o" {
			return viewstate.Push, logview.New(preferredLogPod(items), pods)
		}
		return viewstate.Push, New(pods, v.registry)
	}

	if strings.HasPrefix(resourceName, "pods") || resourceName == "pods" {
		containers := v.resource.Detail(selected).Containers
		if len(containers) <= 1 {
			return viewstate.Push, logview.New(selected, v.resource)
		}
		return viewstate.Push, New(resources.NewContainerResource(selected, v.resource), v.registry)
	}

	if resourceName == "containers" {
		if cRes, ok := v.resource.(*resources.ContainerResource); ok {
			return viewstate.Push, logview.NewWithContainer(cRes.PodItem(), cRes.ParentResource(), selected.Name)
		}
		return viewstate.Push, logview.New(selected, v.resource)
	}

	if resourceName == "deployments" {
		pods := resources.NewWorkloadPods(selected, v.registry)
		if len(pods.Items()) == 0 {
			return viewstate.OpenRelated, nil
		}
		return viewstate.Push, New(pods, v.registry)
	}

	if strings.HasPrefix(resourceName, "services") {
		return viewstate.Push, New(resources.NewBackends(selected, v.registry), v.registry)
	}

	if strings.HasPrefix(resourceName, "ingresses") {
		return viewstate.Push, New(resources.NewIngressServices(selected.Name), v.registry)
	}

	if resourceName == "nodes" {
		return viewstate.Push, New(resources.NewNodePods(selected.Name), v.registry)
	}

	if resourceName == "persistentvolumeclaims" {
		return viewstate.Push, New(resources.NewMountedBy(selected.Name), v.registry)
	}

	if strings.HasPrefix(resourceName, "backends") {
		podContext := resources.NewWorkloadPods(resources.ResourceItem{Name: "backend", Kind: "DEP"}, nil)
		dv := detailview.New(selected, podContext, v.registry)
		dv.ContainerViewFactory = func(item resources.ResourceItem, res resources.ResourceType) viewstate.View {
			return New(resources.NewContainerResource(item, res), v.registry)
		}
		return viewstate.Push, dv
	}

	if strings.HasPrefix(resourceName, "mounted-by") {
		containers := v.resource.Detail(selected).Containers
		if len(containers) <= 1 {
			return viewstate.Push, logview.New(selected, v.resource)
		}
		return viewstate.Push, New(resources.NewContainerResource(selected, v.resource), v.registry)
	}

	return viewstate.None, nil
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

// supportsDelete reports whether the current resource type supports deletion.
func (v *View) supportsDelete() bool {
	return true
}

// supportsRestart reports whether the current resource supports rollout restart.
func (v *View) supportsRestart() bool {
	name := strings.ToLower(v.resource.Name())
	return name == "deployments" || name == "workloads" ||
		strings.HasPrefix(name, "statefulset")
}

// supportsScale reports whether the current resource supports replica scaling.
func (v *View) supportsScale() bool {
	name := strings.ToLower(v.resource.Name())
	return name == "deployments" || strings.HasPrefix(name, "statefulset")
}

// supportsPortFwd reports whether the current resource supports port-forward.
func (v *View) supportsPortFwd() bool {
	name := strings.ToLower(v.resource.Name())
	return name == "pods" || strings.HasPrefix(name, "pods") ||
		strings.HasPrefix(name, "service")
}

// supportsShellExec reports whether the current resource supports kubectl exec shell.
func (v *View) supportsShellExec() bool {
	name := strings.ToLower(v.resource.Name())
	return name == "pods" || strings.HasPrefix(name, "pods") || name == "containers"
}

// shellExecCmd returns a bubbletea command that suspends the TUI and runs
// "kubectl exec -it <pod> [-c <container>] -- sh".
func (v *View) shellExecCmd() bubbletea.Cmd {
	selected, ok := v.list.SelectedItem().(item)
	if !ok {
		return nil
	}
	var podName, containerName string
	if cr, ok := v.resource.(*resources.ContainerResource); ok {
		podName = cr.PodItem().Name
		containerName = selected.data.Name
	} else {
		podName = selected.data.Name
	}
	args := []string{"exec", "-it", podName}
	if containerName != "" {
		args = append(args, "-c", containerName)
	}
	args = append(args, "--", "sh")
	c := exec.Command("kubectl", args...)
	return bubbletea.ExecProcess(c, func(err error) bubbletea.Msg {
		return shellExecResultMsg{err: err}
	})
}

// currentReplicas extracts the desired replica count from the selected item's
// Ready field (e.g. "2/3" → "3"). Falls back to "1".
func (v *View) currentReplicas() string {
	selected, ok := v.list.SelectedItem().(item)
	if !ok {
		return "1"
	}
	ready := strings.TrimSpace(selected.data.Ready)
	if idx := strings.Index(ready, "/"); idx >= 0 {
		return ready[idx+1:]
	}
	if ready != "" {
		return ready
	}
	return "1"
}

// execTargetLabel returns "kind/name" for the currently selected item.
func (v *View) execTargetLabel() string {
	selected, ok := v.list.SelectedItem().(item)
	if !ok {
		return ""
	}
	kind := resources.SingularName(strings.ToLower(breadcrumbLabel(v.resource.Name())))
	return kind + "/" + selected.data.Name
}
