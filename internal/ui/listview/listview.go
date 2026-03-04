package listview

import (
	"fmt"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/paginator"
	"github.com/charmbracelet/bubbles/textinput"
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
type clearActionMsg struct{}

func clearCopiedCmd() bubbletea.Cmd {
	return func() bubbletea.Msg {
		time.Sleep(1500 * time.Millisecond)
		return clearCopiedMsg{}
	}
}

func clearActionCmd() bubbletea.Cmd {
	return func() bubbletea.Msg {
		time.Sleep(1500 * time.Millisecond)
		return clearActionMsg{}
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
type portForwardResultMsg struct{ err error }

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
	data        resources.ResourceItem
	row         []string
	status      string
	widths      []int
	matchColumn int
	dimPodName  bool
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
	sortMode     string
	sortDesc     bool
	findMode     bool
	findTargets  map[int]bool
	searchActive bool
	searchQuery  string
	searchInput  textinput.Model
	matchRows    []int
	matchIndex   int
	copyMode     bool
	copiedMsg    string
	actionMsg    string
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
	listItems := makeListItems(resource, items, rows, widths, columns)

	v := &View{
		resource:  resource,
		registry:  registry,
		columns:   columns,
		colWidths: widths,
		labelPool: labelPool,
		sortMode:  defaultSortMode(resource, columns),
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
	v.searchInput = model.FilterInput
	v.searchInput.Prompt = "/ "
	v.searchInput.SetValue("")
	v.searchInput.Blur()
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
	if _, ok := msg.(clearActionMsg); ok {
		v.actionMsg = ""
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
	if msg, ok := msg.(portForwardResultMsg); ok {
		if msg.err != nil {
			v.execResult = "port-forward failed: " + msg.err.Error()
		} else {
			v.execResult = "port-forward ended"
		}
		return viewstate.Update{Action: viewstate.None, Next: v, Cmd: clearExecResultCmd()}
	}

	if v.searchActive {
		updated, cmd := v.searchInput.Update(msg)
		v.searchInput = updated
		v.searchQuery = v.searchInput.Value()
		if key, ok := msg.(bubbletea.KeyMsg); ok {
			switch key.String() {
			case "enter":
				v.searchActive = false
				v.searchInput.Blur()
				v.recomputeMatches()
				if len(v.matchRows) > 0 {
					v.matchIndex = 0
					v.list.Select(v.matchRows[v.matchIndex])
				}
			case "esc":
				v.searchActive = false
				v.searchInput.Blur()
				v.searchInput.SetValue("")
				v.searchQuery = ""
				v.matchRows = nil
				v.matchIndex = 0
			}
		}
		return viewstate.Update{Action: viewstate.None, Next: v, Cmd: cmd}
	}

	if key, ok := msg.(bubbletea.KeyMsg); ok {
		if v.list.SettingFilter() && key.String() != "esc" {
			updated, cmd := v.list.Update(msg)
			v.list = updated
			v.recomputeMatches()
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
				if r := singleRune(key); r != 0 {
					if colIdx, desc, ok := numericSortKey(r); ok {
						mode := sortModeForColumn(v.resource, v.columns, colIdx)
						if mode != "" {
							// Layout-agnostic behavior: plain number keys toggle direction
							// when selecting the currently active sort column again.
							if r >= '0' && r <= '9' && currentSortMode(v.resource, v.sortMode) == mode {
								desc = !currentSortDesc(v.resource, v.sortDesc)
							}
							setSortState(v.resource, &v.sortMode, &v.sortDesc, mode, desc)
							v.refreshItems()
						}
					} else {
						resourceLabel := resources.SingularName(breadcrumbLabel(v.resource.Name()))
						skeys := sortKeysForView(v.resource, v.columns)
						chars := sortDisplayedChars(v.resource, v.columns, skeys, resourceLabel)
						lc := unicode.ToLower(r)
						desc := unicode.IsUpper(r)
						for _, i := range uniqueSortKeyIndices(chars) {
							ch := chars[i]
							if ch == lc {
								setSortState(v.resource, &v.sortMode, &v.sortDesc, skeys[i].Mode, desc)
								v.refreshItems()
								break
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
				var text string
				switch key.String() {
				case "n":
					text = selected.data.Name
				case "k":
					kind := resources.SingularName(breadcrumbLabel(v.resource.Name()))
					text = kind + "/" + selected.data.Name
				case "p":
					ns := selected.data.Namespace
					if ns == "" {
						if scoped, ok := v.resource.(resources.NamespaceScoped); ok {
							ns = scoped.Namespace()
						} else {
							ns = resources.DefaultNamespace
						}
					}
					text = "-n " + ns + " " + selected.data.Name
				}
				if text != "" {
					if err := clipboard.WriteAll(text); err != nil {
						v.copiedMsg = "clipboard error: " + err.Error()
					} else {
						v.copiedMsg = "copied: " + text
					}
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
				if v.execState == execInputScale {
					resultMsg := "scaled " + label + " to " + v.execInput + " (simulated)"
					v.execState = execNone
					v.execResult = resultMsg
					return viewstate.Update{Action: viewstate.None, Next: v, Cmd: clearExecResultCmd()}
				} else {
					cmd := v.portForwardCmd(strings.TrimSpace(v.execInput))
					v.execState = execNone
					if cmd == nil {
						v.execResult = "port-forward unavailable for " + label
						return viewstate.Update{Action: viewstate.None, Next: v, Cmd: clearExecResultCmd()}
					}
					return viewstate.Update{Action: viewstate.None, Next: v, Cmd: cmd}
				}
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
				v.recomputeMatches()
				return viewstate.Update{Action: viewstate.None, Next: v}
			}
		case "/":
			v.searchActive = true
			v.searchQuery = ""
			v.searchInput.SetValue("")
			v.matchRows = nil
			v.matchIndex = 0
			return viewstate.Update{Action: viewstate.None, Next: v, Cmd: v.searchInput.Focus()}
		case "n":
			if len(v.matchRows) > 0 {
				v.matchIndex = (v.matchIndex + 1) % len(v.matchRows)
				v.list.Select(v.matchRows[v.matchIndex])
				return viewstate.Update{Action: viewstate.None, Next: v}
			}
		case "b":
			if len(v.matchRows) > 0 {
				v.matchIndex = (v.matchIndex - 1 + len(v.matchRows)) % len(v.matchRows)
				v.list.Select(v.matchRows[v.matchIndex])
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
			v.actionMsg = "action unavailable: no selected item"
			return viewstate.Update{Action: viewstate.None, Next: v, Cmd: clearActionCmd()}
		case "w":
			// Wide mode toggle (only if resource implements WideResource).
			if _, ok := v.resource.(resources.WideResource); ok {
				v.wideMode = !v.wideMode
				v.refreshColumns()
				return viewstate.Update{Action: viewstate.None, Next: v}
			}
			v.actionMsg = "w unavailable in this view"
			return viewstate.Update{Action: viewstate.None, Next: v, Cmd: clearActionCmd()}
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
			v.actionMsg = "p unavailable in this view"
			return viewstate.Update{Action: viewstate.None, Next: v, Cmd: clearActionCmd()}
		case "s":
			if len(sortKeysForView(v.resource, v.columns)) > 0 {
				v.sortPickMode = true
			} else {
				v.actionMsg = "s unavailable in this view"
				return viewstate.Update{Action: viewstate.None, Next: v, Cmd: clearActionCmd()}
			}
			return viewstate.Update{Action: viewstate.None, Next: v}
		case "y":
			if selected, ok := v.list.SelectedItem().(item); ok {
				return viewstate.Update{
					Action: viewstate.Push,
					Next:   yamlview.New(selected.data, v.resource),
				}
			}
			v.actionMsg = "action unavailable: no selected item"
			return viewstate.Update{Action: viewstate.None, Next: v, Cmd: clearActionCmd()}
		case "r":
			// Handled by app.go (opens/closes the side panel).
		case "e":
			if selected, ok := v.list.SelectedItem().(item); ok {
				return viewstate.Update{
					Action: viewstate.Push,
					Next:   eventview.New(selected.data, v.resource),
				}
			}
			v.actionMsg = "action unavailable: no selected item"
			return viewstate.Update{Action: viewstate.None, Next: v, Cmd: clearActionCmd()}
		case "d":
			if selected, ok := v.list.SelectedItem().(item); ok {
				return viewstate.Update{
					Action: viewstate.Push,
					Next:   describeview.New(selected.data, v.resource),
				}
			}
			v.actionMsg = "action unavailable: no selected item"
			return viewstate.Update{Action: viewstate.None, Next: v, Cmd: clearActionCmd()}
		case "f":
			v.findMode = true
			v.findTargets = v.computeFindTargets()
			return viewstate.Update{Action: viewstate.None, Next: v}
		case "c":
			if selected, ok := v.list.SelectedItem().(item); ok && selected.data.Name != "" {
				v.copyMode = true
			} else {
				v.actionMsg = "c unavailable: no selected item"
				return viewstate.Update{Action: viewstate.None, Next: v, Cmd: clearActionCmd()}
			}
			return viewstate.Update{Action: viewstate.None, Next: v}
		case "x":
			if selected, ok := v.list.SelectedItem().(item); ok && selected.data.Name != "" {
				v.execState = execMenu
			} else {
				v.actionMsg = "x unavailable: no selected item"
				return viewstate.Update{Action: viewstate.None, Next: v, Cmd: clearActionCmd()}
			}
			return viewstate.Update{Action: viewstate.None, Next: v}
		}
	}

	updated, cmd := v.list.Update(msg)
	v.list = updated
	v.recomputeMatches()
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
	mode := currentSortMode(v.resource, v.sortMode)
	indicator := sortIndicatorSymbol(currentSortDesc(v.resource, v.sortDesc))
	activeSortIdx := activeSortColumn(v.resource, v.columns, mode)
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
	return strings.Join(out, "\n")
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
	// When setting filter, show filter input in status row instead of normal footer
	if v.list.SettingFilter() {
		filterInput := filterbar.FilterInputView(v.list)
		line1 := filterInput
		line2 := style.FormatBindings([]style.Binding{style.B("esc", "cancel")})
		if v.list.Width() > 0 {
			line2 = ansi.Truncate(line2, v.list.Width()-2, "…")
		}
		return line1 + "\n" + line2
	}
	if v.searchActive {
		searchLabel := style.FooterKey.Render("search")
		line1 := searchLabel + "  " + v.searchInput.View()
		if v.list.Width() > 0 {
			line1 = ansi.Truncate(line1, v.list.Width()-2, "…")
		}
		line2 := style.FormatBindings([]style.Binding{
			style.B("enter", "confirm"),
			style.B("esc", "cancel"),
		})
		if v.list.Width() > 0 {
			line2 = ansi.Truncate(line2, v.list.Width()-2, "…")
		}
		return line1 + "\n" + line2
	}

	// Line 1: status indicators + pagination right-aligned.
	var indicators []style.Binding
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
	if len(v.matchRows) > 0 {
		indicators = append(indicators, style.B("match", matchSummary(v.matchIndex, len(v.matchRows))))
	}
	if v.copiedMsg != "" {
		indicators = append(indicators, style.B(v.copiedMsg, ""))
	}
	if v.actionMsg != "" {
		indicators = append(indicators, style.B(v.actionMsg, ""))
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
		execLabel := style.FooterKey.Render("exec")
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
		resourceLabel := resources.SingularName(breadcrumbLabel(v.resource.Name()))
		skeys := sortKeysForView(v.resource, v.columns)
		chars := sortDisplayedChars(v.resource, v.columns, skeys, resourceLabel)
		seen := make(map[rune]bool, len(chars))
		for i, sk := range skeys {
			ch := chars[i]
			colIdx := activeSortColumn(v.resource, v.columns, sk.Mode)
			keyHint := sortColumnNumberLabel(colIdx)
			if ch != 0 && !seen[ch] {
				lower := string(ch)
				upper := string(unicode.ToUpper(ch))
				keyHint = keyHint + "/" + lower + "/" + upper
				seen[ch] = true
			}
			label := strings.ToLower(displayedColHeader(v.columns, colIdx, resourceLabel))
			if label == "" {
				label = sk.Label
			}
			opts = append(opts, style.B(keyHint, label))
		}
		opts = append(opts, style.B("esc", "cancel"))
		line2 = sortLabel + "  " + style.FormatBindings(opts)
		if v.list.Width() > 0 {
			line2 = ansi.Truncate(line2, v.list.Width()-2, "…")
		}

	} else {
		var actions []style.Binding
		isContainers := strings.EqualFold(v.resource.Name(), "containers")

		actions = append(actions, style.B("/", "search"))
		actions = append(actions, style.B("&", "filter"))
		if len(v.matchRows) > 0 {
			actions = append(actions, style.B("n/b", "next/prev"))
		}
		if len(sortKeysForView(v.resource, v.columns)) > 0 {
			actions = append(actions, style.B("s", "sort"))
		}
		if !isContainers {
			actions = append(actions, style.B("r", "related"))
		}
		if _, ok := v.resource.(resources.TableResource); ok {
			actions = append(actions, style.B("p", "cols"))
		}
		if _, ok := v.resource.(resources.WideResource); ok {
			if v.wideMode {
				actions = append(actions, style.B("w", "[wide]"))
			} else {
				actions = append(actions, style.B("w", "wide"))
			}
		}
		actions = append(actions, style.B("c", "copy"))
		actions = append(actions, style.B("x", "exec"))
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
	return v.list.SettingFilter() || v.list.IsFiltered() || v.findMode || v.searchActive || len(v.matchRows) > 0 || v.copyMode || v.execState != execNone || v.sortPickMode
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
	return headerRowWithHint(columns, nil, firstLabel, "", -1, "↑")
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
		if isNameColumn(col) {
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
func makeListItems(resource resources.ResourceType, items []resources.ResourceItem, rows [][]string, widths []int, columns []resources.TableColumn) []list.Item {
	listItems := make([]list.Item, 0, len(items))
	matchColumn := firstColumnWithID(columns, "name")
	if matchColumn < 0 {
		matchColumn = 0
	}
	dimPodName := shouldDimPodNameSuffixes(resource)
	for idx, res := range items {
		var row []string
		if idx < len(rows) {
			row = rows[idx]
		}
		listItems = append(listItems, item{
			data:        res,
			row:         row,
			status:      res.Status,
			widths:      widths,
			matchColumn: matchColumn,
			dimPodName:  dimPodName,
		})
	}
	return listItems
}

func shouldDimPodNameSuffixes(resource resources.ResourceType) bool {
	if resource == nil {
		return false
	}
	return strings.EqualFold(breadcrumbLabel(resource.Name()), "pods")
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
		if firstHeader != "" && isNameColumn(col) {
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
// Any NAME column shows the resource's singular label instead.
func displayedColHeader(columns []resources.TableColumn, idx int, resourceLabel string) string {
	if idx < 0 || idx >= len(columns) {
		return ""
	}
	col := columns[idx]
	if isNameColumn(col) {
		return strings.ToUpper(resourceLabel)
	}
	return col.Name
}

func isNameColumn(col resources.TableColumn) bool {
	if strings.EqualFold(strings.TrimSpace(col.ID), "name") {
		return true
	}
	return strings.EqualFold(strings.TrimSpace(col.Name), "name")
}

// sortDisplayedChars computes sort key characters for all keys, derived from
// displayed column headers.
// It uses the first visible header character (case-insensitive) so "s" + key
// maps directly to the target column's leading character. Numeric column
// selection remains available for disambiguation.
// Returns a parallel slice of runes.
func sortDisplayedChars(resource resources.ResourceType, columns []resources.TableColumn, keys []resources.SortKey, resourceLabel string) []rune {
	result := make([]rune, len(keys))
	for i, sk := range keys {
		colIdx := activeSortColumn(resource, columns, sk.Mode)
		hdr := displayedColHeader(columns, colIdx, resourceLabel)
		ch := sortLeadChar(hdr)
		if ch == 0 {
			ch = sk.Char
		}
		result[i] = ch
	}
	return result
}

func sortLeadChar(header string) rune {
	for _, r := range strings.TrimSpace(header) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return unicode.ToLower(r)
		}
	}
	return 0
}

func uniqueSortKeyIndices(chars []rune) []int {
	seen := make(map[rune]bool, len(chars))
	indices := make([]int, 0, len(chars))
	for i, ch := range chars {
		if seen[ch] {
			continue
		}
		seen[ch] = true
		indices = append(indices, i)
	}
	return indices
}

func sortColumnNumberLabel(colIdx int) string {
	if colIdx < 0 {
		return "-"
	}
	n := colIdx + 1
	if n == 10 {
		return "0"
	}
	return strconv.Itoa(n)
}

func sortKeysForView(resource resources.ResourceType, columns []resources.TableColumn) []resources.SortKey {
	if sortable, ok := resource.(resources.Sortable); ok {
		all := sortable.SortKeys()
		keys := make([]resources.SortKey, 0, len(all))
		for _, sk := range all {
			if activeSortColumn(resource, columns, sk.Mode) >= 0 {
				keys = append(keys, sk)
			}
		}
		return keys
	}
	keys := make([]resources.SortKey, 0, len(columns))
	for _, col := range columns {
		mode := strings.TrimSpace(col.ID)
		if mode == "" {
			continue
		}
		keys = append(keys, resources.SortKey{
			Char:  sortLeadChar(col.Name),
			Mode:  mode,
			Label: strings.ToLower(strings.TrimSpace(col.Name)),
		})
	}
	return keys
}

func defaultSortMode(resource resources.ResourceType, columns []resources.TableColumn) string {
	if sortable, ok := resource.(resources.Sortable); ok {
		if mode := sortable.SortMode(); activeSortColumn(resource, columns, mode) >= 0 {
			return mode
		}
		for _, sk := range sortable.SortKeys() {
			if activeSortColumn(resource, columns, sk.Mode) >= 0 {
				return sk.Mode
			}
		}
		return ""
	}
	if idx := firstColumnWithID(columns, "name"); idx >= 0 {
		return columns[idx].ID
	}
	for _, col := range columns {
		if strings.TrimSpace(col.ID) != "" {
			return col.ID
		}
	}
	return ""
}

func normalizeSortMode(resource resources.ResourceType, columns []resources.TableColumn, mode string) string {
	if mode != "" && activeSortColumn(resource, columns, mode) >= 0 {
		return mode
	}
	return defaultSortMode(resource, columns)
}

func setSortState(resource resources.ResourceType, mode *string, desc *bool, nextMode string, nextDesc bool) {
	if sortable, ok := resource.(resources.Sortable); ok {
		sortable.SetSort(nextMode, nextDesc)
		return
	}
	*mode = nextMode
	*desc = nextDesc
}

// sortModeForColumn returns the sort mode for the given 0-based column index,
// or "" if no sort key maps to that column.
func sortModeForColumn(resource resources.ResourceType, columns []resources.TableColumn, colIdx int) string {
	sortable, ok := resource.(resources.Sortable)
	if !ok {
		if colIdx < 0 || colIdx >= len(columns) {
			return ""
		}
		return columns[colIdx].ID
	}
	for _, sk := range sortable.SortKeys() {
		if activeSortColumn(resource, columns, sk.Mode) == colIdx {
			return sk.Mode
		}
	}
	return ""
}

// numericSortKey interprets digit runes as
// (0-based column index, descending, ok). Digits 1-9 map to columns 0-8,
// and 0 maps to column 9 (10th column). Descending is controlled by reselecting
// the same column key (direction toggle), not by keyboard-layout-specific aliases.
func numericSortKey(r rune) (int, bool, bool) {
	if r >= '1' && r <= '9' {
		return int(r - '1'), false, true
	}
	if r == '0' {
		return 9, false, true
	}
	return 0, false, false
}

func currentSortMode(resource resources.ResourceType, fallback string) string {
	if sortable, ok := resource.(resources.Sortable); ok {
		return sortable.SortMode()
	}
	return fallback
}

func currentSortDesc(resource resources.ResourceType, fallback bool) bool {
	if sortable, ok := resource.(resources.Sortable); ok {
		return sortable.SortDesc()
	}
	return fallback
}

func sortIndicatorSymbol(desc bool) string {
	if desc {
		return "↓"
	}
	return "↑"
}

func activeSortColumn(resource resources.ResourceType, columns []resources.TableColumn, mode string) int {
	if len(columns) == 0 || mode == "" {
		return -1
	}

	switch mode {
	case "name":
		if idx := firstColumnWithID(columns, "name"); idx >= 0 {
			return idx
		}
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
	case "ready":
		return firstColumnWithID(columns, "ready")
	case "restarts":
		return firstColumnWithID(columns, "restarts")
	}
	return firstColumnWithID(columns, mode)
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
	v.sortMode = normalizeSortMode(v.resource, v.columns, v.sortMode)

	if _, ok := v.resource.(resources.Sortable); !ok {
		sort.SliceStable(items, func(i, j int) bool {
			vi := strings.ToLower(strings.TrimSpace(tableRowMap(v.resource, items[i])[v.sortMode]))
			vj := strings.ToLower(strings.TrimSpace(tableRowMap(v.resource, items[j])[v.sortMode]))
			if vi != vj {
				if v.sortDesc {
					return vi > vj
				}
				return vi < vj
			}
			ni := strings.ToLower(strings.TrimSpace(items[i].Name))
			nj := strings.ToLower(strings.TrimSpace(items[j].Name))
			return ni < nj
		})
	}

	rows := assembleRows(v.resource, v.wideMode, v.columns, items)
	firstHeader := strings.ToUpper(resources.SingularName(breadcrumbLabel(v.resource.Name())))
	v.colWidths = columnWidthsForRows(v.columns, rows, v.list.Width()-2, firstHeader)
	listItems := makeListItems(v.resource, items, rows, v.colWidths, v.columns)
	selected := v.list.Index()
	v.list.SetItems(listItems)
	if selected >= 0 && selected < len(listItems) {
		v.list.Select(selected)
	}
	v.recomputeMatches()
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
		return "No workloads found in this namespace. Press N to switch namespace."
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

func (v *View) recomputeMatches() {
	if strings.TrimSpace(v.searchQuery) == "" {
		v.matchRows = nil
		v.matchIndex = 0
		return
	}
	query := strings.ToLower(strings.TrimSpace(v.searchQuery))
	visible := v.list.VisibleItems()
	matches := make([]int, 0, len(visible))
	for i, li := range visible {
		it, ok := li.(item)
		if !ok {
			continue
		}
		rowText := strings.ToLower(strings.Join(it.row, " "))
		if strings.Contains(rowText, query) {
			matches = append(matches, i)
		}
	}
	v.matchRows = matches
	if len(v.matchRows) == 0 {
		v.matchIndex = 0
		return
	}
	if v.matchIndex >= len(v.matchRows) {
		v.matchIndex = len(v.matchRows) - 1
	}
}

func matchSummary(index, total int) string {
	if total <= 0 {
		return "0/0"
	}
	return strconv.Itoa(index+1) + "/" + strconv.Itoa(total)
}

func (v *View) forwardView(selected resources.ResourceItem, key string) (viewstate.Action, viewstate.View) {
	resourceName := strings.ToLower(v.resource.Name())

	if resourceName == "workloads" {
		if livePods, ok := v.liveWorkloadPods(selected); ok {
			base := resources.NewWorkloadPods(selected, v.registry)
			pods := resources.NewQueryResource(base.Name(), livePods, base)
			if key == "o" && len(livePods) > 0 {
				lv := logview.New(preferredLogPod(livePods), pods)
				lv.ContainerViewFactory = func(item resources.ResourceItem, res resources.ResourceType) viewstate.View {
					return New(resources.NewContainerResource(item, res), v.registry)
				}
				return viewstate.Push, lv
			}
			return viewstate.Push, New(pods, v.registry)
		}
		pods := resources.NewWorkloadPods(selected, v.registry)
		items := pods.Items()
		if len(items) == 0 {
			return viewstate.OpenRelated, nil
		}
		if key == "o" {
			lv := logview.New(preferredLogPod(items), pods)
			lv.ContainerViewFactory = func(item resources.ResourceItem, res resources.ResourceType) viewstate.View {
				return New(resources.NewContainerResource(item, res), v.registry)
			}
			return viewstate.Push, lv
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
		if livePods, ok := v.liveWorkloadPods(selected); ok {
			base := resources.NewWorkloadPods(selected, v.registry)
			return viewstate.Push, New(resources.NewQueryResource(base.Name(), livePods, base), v.registry)
		}
		pods := resources.NewWorkloadPods(selected, v.registry)
		if len(pods.Items()) == 0 {
			return viewstate.OpenRelated, nil
		}
		return viewstate.Push, New(pods, v.registry)
	}

	if strings.HasPrefix(resourceName, "services") {
		if backends, ok := v.liveBackendsForService(selected); ok {
			base := resources.NewBackends(selected, v.registry)
			return viewstate.Push, New(resources.NewQueryResource(base.Name(), backends, base), v.registry)
		}
		return viewstate.Push, New(resources.NewBackends(selected, v.registry), v.registry)
	}

	if strings.HasPrefix(resourceName, "ingresses") {
		if svcs, ok := v.liveServicesForIngress(selected); ok {
			base := resources.NewIngressServices(selected.Name)
			return viewstate.Push, New(resources.NewQueryResource(base.Name(), svcs, base), v.registry)
		}
		return viewstate.Push, New(resources.NewIngressServices(selected.Name), v.registry)
	}

	if resourceName == "nodes" {
		if pods, ok := v.livePodsForNode(selected); ok {
			base := resources.NewNodePods(selected.Name)
			return viewstate.Push, New(resources.NewQueryResource(base.Name(), pods, base), v.registry)
		}
		return viewstate.Push, New(resources.NewNodePods(selected.Name), v.registry)
	}

	if resourceName == "persistentvolumeclaims" {
		if pods, ok := v.livePodsForPVC(selected); ok {
			base := resources.NewMountedBy(selected.Name)
			return viewstate.Push, New(resources.NewQueryResource(base.Name(), pods, base), v.registry)
		}
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

func (v *View) liveWorkloadPods(workload resources.ResourceItem) ([]resources.ResourceItem, bool) {
	if strings.TrimSpace(workload.UID) == "" {
		return nil, false
	}
	pods, err := v.listResource("pods")
	if err != nil {
		return nil, false
	}
	out := make([]resources.ResourceItem, 0, len(pods))
	for _, pod := range pods {
		if resources.MatchesSelector(workload.Selector, pod.Labels) {
			out = append(out, pod)
		}
	}
	return out, true
}

func (v *View) liveBackendsForService(service resources.ResourceItem) ([]resources.ResourceItem, bool) {
	pods, err := v.listResource("pods")
	if err != nil {
		return nil, false
	}
	out := make([]resources.ResourceItem, 0, len(pods))
	for _, pod := range pods {
		if resources.MatchesSelector(service.Selector, pod.Labels) {
			out = append(out, pod)
		}
	}
	return out, true
}

func (v *View) liveServicesForIngress(ingress resources.ResourceItem) ([]resources.ResourceItem, bool) {
	services, err := v.listResource("services")
	if err != nil {
		return nil, false
	}
	want := map[string]bool{}
	for _, name := range strings.Split(ingress.Extra["services"], ",") {
		name = strings.TrimSpace(name)
		if name != "" {
			want[name] = true
		}
	}
	out := make([]resources.ResourceItem, 0, len(services))
	for _, svc := range services {
		if want[svc.Name] {
			out = append(out, svc)
		}
	}
	return out, true
}

func (v *View) livePodsForNode(node resources.ResourceItem) ([]resources.ResourceItem, bool) {
	pods, err := v.listResource("pods")
	if err != nil {
		return nil, false
	}
	out := make([]resources.ResourceItem, 0, len(pods))
	for _, pod := range pods {
		if strings.TrimSpace(pod.Extra["node"]) == node.Name {
			out = append(out, pod)
		}
	}
	return out, true
}

func (v *View) livePodsForPVC(pvc resources.ResourceItem) ([]resources.ResourceItem, bool) {
	pods, err := v.listResource("pods")
	if err != nil {
		return nil, false
	}
	out := make([]resources.ResourceItem, 0, len(pods))
	for _, pod := range pods {
		for _, ref := range strings.Split(pod.Extra["pvc-refs"], ",") {
			if strings.TrimSpace(ref) == pvc.Name {
				out = append(out, pod)
				break
			}
		}
	}
	return out, true
}

func (v *View) listResource(resourceName string) ([]resources.ResourceItem, error) {
	lister, ok := v.resource.(interface {
		ListResource(resourceName string) ([]resources.ResourceItem, error)
	})
	if !ok {
		return nil, fmt.Errorf("list resource unavailable")
	}
	return lister.ListResource(resourceName)
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

// portForwardCmd returns a bubbletea command that suspends the TUI and runs
// "kubectl [-n <namespace>] port-forward <pod/name|service/name> <ports>".
func (v *View) portForwardCmd(ports string) bubbletea.Cmd {
	selected, ok := v.list.SelectedItem().(item)
	if !ok {
		return nil
	}
	args, ok := v.portForwardArgs(selected, ports)
	if !ok {
		return nil
	}
	c := exec.Command("kubectl", args...)
	return bubbletea.ExecProcess(c, func(err error) bubbletea.Msg {
		return portForwardResultMsg{err: err}
	})
}

func (v *View) portForwardArgs(selected item, ports string) ([]string, bool) {
	ports = strings.TrimSpace(ports)
	if selected.data.Name == "" || ports == "" {
		return nil, false
	}

	target := "pod/" + selected.data.Name
	if strings.HasPrefix(strings.ToLower(v.resource.Name()), "service") {
		target = "service/" + selected.data.Name
	}

	args := make([]string, 0, 6)
	if ns := v.execNamespace(selected.data); ns != "" && ns != resources.AllNamespaces {
		args = append(args, "-n", ns)
	}
	args = append(args, "port-forward", target, ports)
	return args, true
}

func (v *View) execNamespace(item resources.ResourceItem) string {
	if ns := strings.TrimSpace(item.Namespace); ns != "" {
		return ns
	}
	if scoped, ok := v.resource.(resources.NamespaceScoped); ok {
		return strings.TrimSpace(scoped.Namespace())
	}
	return ""
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
