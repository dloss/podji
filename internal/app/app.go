package app

import (
	"fmt"
	"sort"
	"strings"
	"unicode"

	bubbletea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/dloss/podji/internal/resources"
	"github.com/dloss/podji/internal/ui/columnpicker"
	"github.com/dloss/podji/internal/ui/commandbar"
	"github.com/dloss/podji/internal/ui/describeview"
	"github.com/dloss/podji/internal/ui/detailview"
	"github.com/dloss/podji/internal/ui/eventview"
	"github.com/dloss/podji/internal/ui/helpview"
	"github.com/dloss/podji/internal/ui/listview"
	"github.com/dloss/podji/internal/ui/overlaypicker"
	"github.com/dloss/podji/internal/ui/relatedview"
	"github.com/dloss/podji/internal/ui/resourcebrowser"
	"github.com/dloss/podji/internal/ui/style"
	"github.com/dloss/podji/internal/ui/viewstate"
	"github.com/dloss/podji/internal/ui/yamlview"
)

// Bookmark captures a full navigation state: the complete view stack, breadcrumbs,
// namespace, and context, so a jump restores the exact screen where it was set.
type Bookmark struct {
	stack     []viewstate.View
	crumbs    []string
	namespace string
	context   string
}

type Model struct {
	registry          *resources.Registry
	stack             []viewstate.View
	crumbs            []string
	overlay           *overlaypicker.Picker
	relatedPicker     *relatedview.Picker
	colPicker         *columnpicker.Picker
	cmdBar            *commandbar.Model
	context           string
	namespace         string
	errorMsg          string
	statusMsg         string
	bookmarks         [9]*Bookmark
	bookmarkMode      bool
	activeResourceKey rune
	width             int
	height            int
}

type globalKeySuppresser interface {
	SuppressGlobalKeys() bool
}

// bodyRowProvider is implemented by views that report the visual line (within
// their own View() output) at which the selected row appears.
type bodyRowProvider interface {
	SelectedBodyRow() int
}

func New() Model {
	registry := resources.DefaultRegistry()
	workloads := registry.ResourceByKey('W')
	root := listview.New(workloads, registry)
	rootCrumb := normalizeBreadcrumbPart(root.Breadcrumb())

	return Model{
		registry:          registry,
		stack:             []viewstate.View{root},
		crumbs:            []string{rootCrumb},
		context:           "default",
		namespace:         "default",
		activeResourceKey: 'W',
	}
}

func (m Model) Init() bubbletea.Cmd {
	return m.top().Init()
}

func (m Model) Update(msg bubbletea.Msg) (bubbletea.Model, bubbletea.Cmd) {
	routedMsg := msg

	// Route all input to the overlay pickers when active.
	if m.overlay != nil {
		if _, ok := msg.(bubbletea.KeyMsg); ok {
			update := m.overlay.Update(msg)
			if update.Action == viewstate.Pop {
				m.overlay = nil
			}
			return m, update.Cmd
		}
	}
	if m.relatedPicker != nil {
		if _, ok := msg.(bubbletea.KeyMsg); ok {
			update := m.relatedPicker.Update(msg)
			if update.Action == viewstate.Pop {
				m.relatedPicker = nil
			}
			return m, update.Cmd
		}
	}
	if m.colPicker != nil {
		if _, ok := msg.(bubbletea.KeyMsg); ok {
			update := m.colPicker.Update(msg)
			if update.Action == viewstate.Pop {
				m.colPicker = nil
			}
			return m, update.Cmd
		}
	}
	if m.cmdBar != nil {
		if key, ok := msg.(bubbletea.KeyMsg); ok {
			if key.String() == "tab" {
				m.cmdBar.Complete(m.commandSuggestion())
				return m, nil
			}
			_, cmd, closeBar := m.cmdBar.Update(key)
			if closeBar {
				if key.String() == "enter" {
					return m, cmd
				}
				m.cmdBar = nil
			}
			return m, cmd
		}
	}

	switch msg := msg.(type) {
	case bubbletea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.top().SetSize(m.width, m.availableHeight())
		if m.overlay != nil {
			m.overlay.SetSize(m.width, m.height-1)
		}
		if m.relatedPicker != nil {
			m.relatedPicker.SetSize(m.width, m.height-1)
		}
		if m.colPicker != nil {
			m.colPicker.SetSize(m.width, m.height-1)
		}
		if m.cmdBar != nil {
			m.cmdBar.SetSize(m.width)
		}
		return m, nil

	case commandbar.SubmitMsg:
		if strings.TrimSpace(msg.Value) == "" {
			m.cmdBar = nil
			return m, nil
		}
		if err := m.runCommand(msg.Value); err != "" {
			m.cmdBar.SetError(err)
			return m, nil
		}
		m.cmdBar = nil
		return m, nil

	case listview.OpenColumnPickerMsg:
		picker := columnpicker.New(msg.ResourceName, msg.Pool, msg.LabelPool, msg.Current)
		picker.SetSize(m.width, m.height-1)
		m.colPicker = picker
		return m, nil

	case columnpicker.PickedMsg:
		if lv, ok := m.top().(*listview.View); ok {
			lv.ApplyColumnConfig(msg.ResourceName, msg.Visible)
		}
		return m, nil

	case overlaypicker.SelectedMsg:
		m.overlay = nil
		if msg.Kind == "namespace" {
			m.namespace = msg.Value
			resources.ActiveNamespace = msg.Value
		} else {
			m.context = msg.Value
		}
		// Choose the best resource using the fallback chain:
		// current resource → parent resource(s) → workloads (W) → first registry resource.
		if res := m.bestResourceForScope(); res != nil {
			view := listview.New(res, m.registry)
			view.SetSize(m.width, m.availableHeight())
			m.stack = []viewstate.View{view}
			m.crumbs = []string{normalizeBreadcrumbPart(view.Breadcrumb())}
			m.activeResourceKey = res.Key()
		}
		return m, nil

	case relatedview.SelectedMsg:
		m.relatedPicker = nil
		next := msg.Open()
		next.SetSize(m.width, m.availableHeight())
		if len(m.crumbs) > 0 {
			if selected, ok := m.top().(viewstate.SelectionProvider); ok {
				item := selected.SelectedItem()
				if item.Name != "" {
					label := breadcrumbLabel(m.top())
					m.crumbs[len(m.crumbs)-1] = normalizeBreadcrumbPart(label + ": " + item.Name)
				}
			}
		}
		m.stack = append(m.stack, next)
		m.crumbs = append(m.crumbs, normalizeBreadcrumbPart(next.Breadcrumb()))
		return m, nil

	case bubbletea.KeyMsg:
		m.statusMsg = ""
		if suppresser, ok := m.top().(globalKeySuppresser); ok && suppresser.SuppressGlobalKeys() && msg.String() != "ctrl+c" {
			break
		}
		msg = normalizeGlobalKey(msg)
		routedMsg = msg

		// Handle bookmark-set mode: next digit sets the slot.
		if m.bookmarkMode {
			m.bookmarkMode = false
			runes := []rune(msg.String())
			if len(runes) == 1 && runes[0] >= '1' && runes[0] <= '9' {
				slot := int(runes[0] - '1')
				m.bookmarks[slot] = &Bookmark{
					stack:     append([]viewstate.View{}, m.stack...),
					crumbs:    append([]string{}, m.crumbs...),
					namespace: m.namespace,
					context:   m.context,
				}
				m.statusMsg = fmt.Sprintf("Bookmark %d set", slot+1)
			}
			return m, nil
		}

		switch msg.String() {
		case ":":
			if _, ok := m.top().(*listview.View); ok {
				m.cmdBar = commandbar.New()
				m.cmdBar.SetSize(m.width)
			}
			return m, nil
		case "q", "ctrl+c":
			return m, bubbletea.Quit
		case "esc":
			if len(m.stack) > 1 {
				m.stack = m.stack[:len(m.stack)-1]
				m.crumbs = m.crumbs[:len(m.crumbs)-1]
				m.crumbs[len(m.crumbs)-1] = normalizeBreadcrumbPart(m.top().Breadcrumb())
			}
			return m, nil
		case "backspace":
			if len(m.stack) > 1 {
				m.stack = m.stack[:len(m.stack)-1]
				m.crumbs = m.crumbs[:len(m.crumbs)-1]
				m.crumbs[len(m.crumbs)-1] = normalizeBreadcrumbPart(m.top().Breadcrumb())
			}
			return m, nil
		case "h", "left":
			if len(m.stack) > 1 {
				m.stack = m.stack[:len(m.stack)-1]
				m.crumbs = m.crumbs[:len(m.crumbs)-1]
				m.crumbs[len(m.crumbs)-1] = normalizeBreadcrumbPart(m.top().Breadcrumb())
			}
			return m, nil
		case "N":
			items := resources.NamespaceNames()
			m.overlay = overlaypicker.New("namespace", items)
			m.overlay.SetAnchor(m.namespaceLabelX())
			m.overlay.SetSize(m.width, m.height-1)
			return m, nil
		case "X":
			items := resources.ContextNames()
			m.overlay = overlaypicker.New("context", items)
			m.overlay.SetAnchor(0)
			m.overlay.SetSize(m.width, m.height-1)
			return m, nil
		case "A":
			browser := resourcebrowser.New(m.registry, resources.StubCRDs())
			browser.SetSize(m.width, m.availableHeight())
			m.stack = []viewstate.View{browser}
			m.crumbs = []string{"resources"}
			return m, nil
		case "r":
			m.relatedPicker = relatedview.NewPickerForSelection(m.top(), m.registry)
			m.relatedPicker.SetSize(m.width, m.height-1)
			return m, nil
		case "?":
			if _, isHelp := m.top().(*helpview.View); !isHelp {
				help := helpview.New()
				help.SetSize(m.width, m.availableHeight())
				m.stack = append(m.stack, help)
				m.crumbs = append(m.crumbs, m.crumbs[len(m.crumbs)-1])
			}
			return m, nil
		case "m":
			m.bookmarkMode = true
			m.statusMsg = "Set bookmark: press 1–9"
			return m, nil
		default:
			runes := []rune(msg.String())
			if len(runes) == 1 {
				key := runes[0]
				if key >= '1' && key <= '9' {
					slot := int(key - '1')
					if m.bookmarks[slot] == nil {
						m.statusMsg = fmt.Sprintf("Bookmark %d not set", slot+1)
					} else {
						b := m.bookmarks[slot]
						m.context = b.context
						m.namespace = b.namespace
						resources.ActiveNamespace = b.namespace
						m.stack = append([]viewstate.View{}, b.stack...)
						m.crumbs = append([]string{}, b.crumbs...)
						m.top().SetSize(m.width, m.availableHeight())
						m.activeResourceKey = m.rootResourceKey()
						m.statusMsg = fmt.Sprintf("Bookmark %d", slot+1)
					}
					return m, nil
				}
				if res := m.registry.ResourceByKey(key); res != nil {
					view := listview.New(res, m.registry)
					view.SetSize(m.width, m.availableHeight())
					m.stack = []viewstate.View{view}
					m.crumbs = []string{normalizeBreadcrumbPart(view.Breadcrumb())}
					m.activeResourceKey = key
					return m, nil
				}
			}
		}
	}

	update := m.top().Update(routedMsg)
	switch update.Action {
	case viewstate.Push:
		if len(m.crumbs) > 0 {
			committed := m.crumbs[len(m.crumbs)-1]
			if selected, ok := m.top().(viewstate.SelectionProvider); ok {
				item := selected.SelectedItem()
				if item.Name != "" {
					label := breadcrumbLabel(m.top())
					committed = normalizeBreadcrumbPart(label + ": " + item.Name)
				}
			}
			m.crumbs[len(m.crumbs)-1] = committed
		}
		update.Next.SetSize(m.width, m.availableHeight())
		m.stack = append(m.stack, update.Next)
		m.crumbs = append(m.crumbs, normalizeBreadcrumbPart(update.Next.Breadcrumb()))
	case viewstate.Pop:
		if len(m.stack) > 1 {
			m.stack = m.stack[:len(m.stack)-1]
			m.crumbs = m.crumbs[:len(m.crumbs)-1]
			m.crumbs[len(m.crumbs)-1] = normalizeBreadcrumbPart(m.top().Breadcrumb())
		}
	case viewstate.Replace:
		update.Next.SetSize(m.width, m.availableHeight())
		m.stack[len(m.stack)-1] = update.Next
		m.crumbs[len(m.crumbs)-1] = normalizeBreadcrumbPart(update.Next.Breadcrumb())
	case viewstate.OpenRelated:
		m.relatedPicker = relatedview.NewPickerForSelection(m.top(), m.registry)
		m.relatedPicker.SetSize(m.width, m.height-1)
	default:
		m.stack[len(m.stack)-1] = update.Next
	}

	return m, update.Cmd
}

func (m Model) renderHeader() string {
	return m.scopeLine() + "\n" + m.breadcrumbLine()
}

func (m Model) renderBody() string {
	body := m.top().View()
	if m.height > 0 {
		body = fitViewLines(body, m.bodyHeightLimit())
	}
	return body
}

func (m Model) renderMain() string {
	header := m.renderHeader()
	body := m.renderBody()
	footer := m.top().Footer()
	if m.cmdBar != nil {
		footer = m.cmdBar.View(m.commandSuggestion())
	} else if m.statusMsg != "" || m.errorMsg != "" {
		msg := m.statusMsg
		if m.errorMsg != "" {
			msg = m.errorMsg
		}
		lines := strings.SplitN(footer, "\n", 2)
		if len(lines) > 1 {
			footer = msg + "\n" + lines[1]
		} else {
			footer = msg
		}
	}

	sections := []string{header}
	if body != "" {
		sections = append(sections, body)
	}
	sections = append(sections, footer)

	return strings.Join(sections, "\n")
}

// relatedPickerRow returns the startRow for compositeOverlay so the picker
// appears just below the selected row. Falls back to above the selected row if
// there isn't room below, and to row 1 as a final fallback.
func (m Model) relatedPickerRow(pickerHeight int) int {
	headerLines := m.headerLineCount()
	if provider, ok := m.top().(bodyRowProvider); ok {
		bodyRow := provider.SelectedBodyRow()
		if bodyRow >= 0 {
			selectedLine := headerLines + bodyRow
			belowStart := selectedLine + 1
			if belowStart+pickerHeight <= m.height {
				return belowStart
			}
			aboveStart := selectedLine - pickerHeight
			if aboveStart >= headerLines {
				return aboveStart
			}
		}
	}
	return 1
}

func (m Model) View() string {
	main := m.renderMain()
	if m.overlay != nil {
		return compositeOverlay(main, m.overlay.View(), m.overlay.AnchorX(), 1)
	}
	if m.relatedPicker != nil {
		pickerView := m.relatedPicker.View()
		pickerHeight := strings.Count(pickerView, "\n") + 1
		return compositeOverlay(main, pickerView, m.relatedPicker.AnchorX(), m.relatedPickerRow(pickerHeight))
	}
	if m.colPicker != nil {
		return compositeOverlay(main, m.colPicker.View(), m.colPicker.AnchorX(), 1)
	}
	return main
}

// compositeOverlay overlays box over bg, placing the first line of box at
// startRow in the bg string, with the box's left edge at anchorX visual columns.
func compositeOverlay(bg, box string, anchorX, startRow int) string {
	bgLines := strings.Split(bg, "\n")
	boxLines := strings.Split(box, "\n")

	result := make([]string, len(bgLines))
	for i, bgLine := range bgLines {
		boxIdx := i - startRow
		if boxIdx >= 0 && boxIdx < len(boxLines) {
			result[i] = mergeLine(bgLine, boxLines[boxIdx], anchorX)
		} else {
			result[i] = bgLine
		}
	}
	return strings.Join(result, "\n")
}

// mergeLine places boxLine at anchorX visual columns over bgLine.
// The portion of bgLine to the left of anchorX is preserved with its ANSI
// styling. Text to the right of the box is shown as plain text.
func mergeLine(bgLine, boxLine string, anchorX int) string {
	left := ansi.Truncate(bgLine, anchorX, "")
	boxWidth := lipgloss.Width(boxLine)
	plainBg := ansi.Strip(bgLine)
	bgRunes := []rune(plainBg)
	rightStart := anchorX + boxWidth
	if rightStart < len(bgRunes) {
		return left + boxLine + string(bgRunes[rightStart:])
	}
	return left + boxLine
}

func (m Model) top() viewstate.View {
	return m.stack[len(m.stack)-1]
}

// rootResourceKey returns the key of the root-level registered resource in the
// current view stack. It walks from the bottom up to find the first listview
// whose resource has a non-zero key (i.e. a top-level resource, not a
// sub-resource like containers). Falls back to m.activeResourceKey.
func (m Model) rootResourceKey() rune {
	for i := 0; i < len(m.stack); i++ {
		if lv, ok := m.stack[i].(*listview.View); ok {
			if key := lv.Resource().Key(); key != 0 {
				return key
			}
		}
	}
	return m.activeResourceKey
}

// bestResourceForScope returns the best resource to show after a scope change.
// It walks the view stack from top to bottom looking for a list view, then
// falls back to workloads ('W'), and finally to the first registered resource.
func (m Model) bestResourceForScope() resources.ResourceType {
	for i := len(m.stack) - 1; i >= 0; i-- {
		if lv, ok := m.stack[i].(*listview.View); ok {
			if res := lv.Resource(); res != nil {
				return res
			}
		}
	}
	if res := m.registry.ResourceByKey('W'); res != nil {
		return res
	}
	if all := m.registry.Resources(); len(all) > 0 {
		return all[0]
	}
	return nil
}

func (m Model) scopeLine() string {
	sep := style.NavSep.Render(" > ")

	contextLabel := style.Scope.Render("Context: ")
	contextValue := style.ScopeValue.Render(m.context)

	nsLabel := style.Scope.Render("Namespace: ")
	var nsValue string
	if m.namespace == resources.AllNamespaces {
		nsValue = style.Muted.Render(m.namespace)
	} else {
		nsValue = style.ScopeValue.Render(m.namespace)
	}

	return contextLabel + contextValue + sep + nsLabel + nsValue
}

// namespaceLabelX returns the visual column where "Namespace:" starts in the scope line.
func (m Model) namespaceLabelX() int {
	return lipgloss.Width(style.Scope.Render("Context: ") +
		style.ScopeValue.Render(m.context) +
		style.NavSep.Render(" > "))
}

func (m Model) breadcrumbLine() string {
	rootTag := style.Scope.Render("[" + crumbText(m.crumbs[0]) + "]")
	ancestors := m.crumbs[:len(m.crumbs)-1]
	if len(ancestors) <= 1 {
		return rootTag
	}

	sep := style.NavSep.Render(" > ")
	segments := make([]string, 0, len(ancestors)-1)
	for _, part := range ancestors[1:] {
		segments = append(segments, formatCrumb(part))
	}
	return rootTag + "  " + strings.Join(segments, sep)
}

// crumbText returns the plain display text for a crumb, used inside styled brackets.
func crumbText(crumb string) string {
	if idx := strings.Index(crumb, ": "); idx >= 0 {
		return titleCase(resources.SingularName(crumb[:idx])) + ": " + crumb[idx+2:]
	}
	return titleCase(resources.SingularName(crumb))
}

func formatCrumb(crumb string) string {
	if idx := strings.Index(crumb, ": "); idx >= 0 {
		label := titleCase(resources.SingularName(crumb[:idx]))
		value := crumb[idx+2:]
		return style.Crumb.Render(label+": ") + style.CrumbValue.Render(value)
	}
	return style.Crumb.Render(titleCase(resources.SingularName(crumb)))
}

// headerLineCount returns the number of lines consumed by the header section.
func (m Model) headerLineCount() int {
	return 2
}

func (m Model) availableHeight() int {
	if m.height == 0 {
		return 0
	}
	footerLines := 2
	if m.cmdBar != nil {
		footerLines = 1
	}
	height := m.height - m.headerLineCount() - footerLines
	if height < 1 {
		return 1
	}
	return height
}

func (m Model) bodyHeightLimit() int {
	if m.height <= 0 {
		return 0
	}
	footerLines := 2
	if m.cmdBar != nil {
		footerLines = 1
	}
	limit := m.height - m.headerLineCount() - footerLines
	if limit < 0 {
		return 0
	}
	return limit
}

func fitViewLines(view string, targetLines int) string {
	if targetLines <= 0 {
		return ""
	}

	lines := strings.Split(view, "\n")
	if view == "" {
		lines = nil
	}
	if len(lines) > targetLines {
		return strings.Join(lines[:targetLines], "\n")
	}
	if len(lines) < targetLines {
		lines = append(lines, make([]string, targetLines-len(lines))...)
	}
	return strings.Join(lines, "\n")
}

func titleCase(value string) string {
	if value == "" {
		return value
	}
	runes := []rune(value)
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}

type parsedCommand struct {
	kindToken string
	name      string
	selector  string
	subview   string
}

func (m *Model) runCommand(raw string) string {
	cmd := parseCommand(raw)
	if cmd.kindToken == "unhealthy" {
		base := m.registry.ResourceByKey('W')
		view := listview.New(resources.NewQueryResource("workloads", resources.UnhealthyItems(), base), m.registry)
		view.SetSize(m.width, m.availableHeight())
		m.stack = append(m.stack, view)
		m.crumbs = append(m.crumbs, "unhealthy")
		return ""
	}
	if cmd.kindToken == "restarts" {
		base := m.registry.ResourceByKey('P')
		view := listview.New(resources.NewQueryResource("pods", resources.PodsByRestarts(), base), m.registry)
		view.SetSize(m.width, m.availableHeight())
		m.stack = append(m.stack, view)
		m.crumbs = append(m.crumbs, "restarts")
		return ""
	}
	res := m.commandResource(cmd.kindToken)
	if res == nil {
		return "unknown command"
	}
	if cmd.selector != "" {
		if cmd.subview != "" {
			return "unknown command"
		}
		var filtered []resources.ResourceItem
		for _, it := range res.Items() {
			if resources.MatchesLabelSelector(it, cmd.selector) {
				filtered = append(filtered, it)
			}
		}
		view := listview.New(resources.NewQueryResource(res.Name(), filtered, res), m.registry)
		view.SetSize(m.width, m.availableHeight())
		m.stack = append(m.stack, view)
		m.crumbs = append(m.crumbs, normalizeBreadcrumbPart(res.Name()+": "+cmd.selector))
		return ""
	}
	if cmd.name == "" {
		if lv, ok := m.top().(*listview.View); ok && lv.Resource().Name() == res.Name() {
			return ""
		}
		view := listview.New(res, m.registry)
		view.SetSize(m.width, m.availableHeight())
		m.stack = []viewstate.View{view}
		m.crumbs = []string{normalizeBreadcrumbPart(view.Breadcrumb())}
		m.activeResourceKey = res.Key()
		return ""
	}
	items := res.Items()
	matches := nameMatches(items, cmd.name)
	if len(matches) == 0 {
		return "no match"
	}
	if len(matches) > 1 {
		view := listview.New(resources.NewQueryResource(res.Name(), matches, res), m.registry)
		view.SetSize(m.width, m.availableHeight())
		m.stack = append(m.stack, view)
		m.crumbs = append(m.crumbs, normalizeBreadcrumbPart(res.Name()+": "+cmd.name))
		return ""
	}
	selected := matches[0]
	if lv, ok := m.top().(*listview.View); !ok || lv.Resource().Name() != res.Name() {
		resView := listview.New(res, m.registry)
		resView.SetSize(m.width, m.availableHeight())
		m.stack = append(m.stack, resView)
		m.crumbs = append(m.crumbs, normalizeBreadcrumbPart(resView.Breadcrumb()))
	}
	if len(m.crumbs) > 0 {
		m.crumbs[len(m.crumbs)-1] = normalizeBreadcrumbPart(res.Name() + ": " + selected.Name)
	}
	var next viewstate.View
	detail := detailViewFor(selected, res, m.registry)
	if cmd.subview == "" || cmd.subview == "detail" {
		detail.SetSize(m.width, m.availableHeight())
		m.stack = append(m.stack, detail)
		m.crumbs = append(m.crumbs, normalizeBreadcrumbPart(detail.Breadcrumb()))
		return ""
	}
	detail.SetSize(m.width, m.availableHeight())
	m.stack = append(m.stack, detail)
	m.crumbs = append(m.crumbs, normalizeBreadcrumbPart(detail.Breadcrumb()))

	v := listview.New(res, m.registry)
	switch cmd.subview {
	case "logs":
		_, next = v.ForwardViewForCommand(selected, cmd.subview)
	case "yaml":
		next = yamlview.New(selected, res)
	case "events":
		next = eventview.New(selected, res)
	case "describe":
		next = describeview.New(selected, res)
	default:
		m.stack = m.stack[:len(m.stack)-1]
		m.crumbs = m.crumbs[:len(m.crumbs)-1]
		return "unknown command"
	}
	if next == nil {
		m.stack = m.stack[:len(m.stack)-1]
		m.crumbs = m.crumbs[:len(m.crumbs)-1]
		return "unknown command"
	}
	next.SetSize(m.width, m.availableHeight())
	m.stack = append(m.stack, next)
	m.crumbs = append(m.crumbs, normalizeBreadcrumbPart(next.Breadcrumb()))
	return ""
}

func detailViewFor(item resources.ResourceItem, res resources.ResourceType, registry *resources.Registry) viewstate.View {
	dv := detailview.New(item, res, registry)
	dv.ContainerViewFactory = func(item resources.ResourceItem, res resources.ResourceType) viewstate.View {
		return listview.New(resources.NewContainerResource(item, res), registry)
	}
	return dv
}

func parseCommand(raw string) parsedCommand {
	toks := strings.Fields(strings.ToLower(strings.TrimSpace(raw)))
	if len(toks) == 0 {
		return parsedCommand{}
	}
	cmd := parsedCommand{kindToken: toks[0]}
	if len(toks) >= 2 {
		if strings.Contains(toks[1], "=") {
			cmd.selector = toks[1]
		} else {
			cmd.name = toks[1]
		}
	}
	if len(toks) >= 3 {
		cmd.subview = toks[2]
	}
	return cmd
}

func nameMatches(items []resources.ResourceItem, frag string) []resources.ResourceItem {
	var pref, subs []resources.ResourceItem
	for _, it := range items {
		name := strings.ToLower(it.Name)
		if strings.HasPrefix(name, frag) {
			pref = append(pref, it)
		} else if strings.Contains(name, frag) {
			subs = append(subs, it)
		}
	}
	return append(pref, subs...)
}

func (m Model) commandResource(token string) resources.ResourceType {
	aliases := map[string]string{"po": "pods", "pods": "pods", "deploy": "deployments", "deployments": "deployments", "svc": "services", "services": "services", "cm": "configmaps", "configmaps": "configmaps", "secret": "secrets", "sec": "secrets", "secrets": "secrets", "node": "nodes", "nodes": "nodes", "ing": "ingresses", "ingresses": "ingresses", "pvc": "pvcs", "pvcs": "pvcs", "ev": "events", "events": "events", "ns": "namespaces", "namespaces": "namespaces"}
	name := aliases[token]
	if name == "" {
		return nil
	}
	return m.registry.ByName(name)
}

func (m Model) commandSuggestion() string {
	if m.cmdBar == nil {
		return ""
	}
	input := strings.ToLower(strings.TrimSpace(m.cmdBar.Input()))
	if input == "" {
		return ""
	}
	endsSpace := strings.HasSuffix(input, " ")
	tokens := strings.Fields(input)
	kinds := []string{"po", "deploy", "svc", "cm", "sec", "node", "ing", "pvc", "ev", "ns", "unhealthy", "restarts"}

	if len(tokens) == 1 && !endsSpace {
		sort.Strings(kinds)
		for _, c := range kinds {
			if strings.HasPrefix(c, tokens[0]) && c != tokens[0] {
				return strings.TrimPrefix(c, tokens[0])
			}
		}
		return ""
	}

	res := m.commandResource(tokens[0])
	if res == nil {
		return ""
	}

	if (len(tokens) == 1 && endsSpace) || (len(tokens) == 2 && !endsSpace && !strings.Contains(tokens[1], "=")) {
		prefix := ""
		if len(tokens) >= 2 {
			prefix = tokens[1]
		}
		names := uniqueSortedNames(res.Items())
		for _, n := range names {
			if strings.HasPrefix(strings.ToLower(n), prefix) && strings.ToLower(n) != prefix {
				return strings.TrimPrefix(strings.ToLower(n), prefix)
			}
		}
		return ""
	}

	if (len(tokens) == 2 && endsSpace && !strings.Contains(tokens[1], "=")) || (len(tokens) == 3 && !endsSpace) {
		prefix := ""
		if len(tokens) == 3 {
			prefix = tokens[2]
		}
		subviews := []string{"logs", "yaml", "events", "describe"}
		for _, sv := range subviews {
			if strings.HasPrefix(sv, prefix) && sv != prefix {
				return strings.TrimPrefix(sv, prefix)
			}
		}
	}
	return ""
}

func uniqueSortedNames(items []resources.ResourceItem) []string {
	seen := map[string]bool{}
	var out []string
	for _, it := range items {
		if it.Name == "" {
			continue
		}
		k := strings.ToLower(it.Name)
		if seen[k] {
			continue
		}
		seen[k] = true
		out = append(out, it.Name)
	}
	sort.Slice(out, func(i, j int) bool { return strings.ToLower(out[i]) < strings.ToLower(out[j]) })
	return out
}

func normalizeBreadcrumbPart(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return value
	}

	open := strings.Index(value, "(")
	if open <= 0 || !strings.HasSuffix(value, ")") {
		return value
	}

	label := strings.TrimSpace(value[:open])
	context := strings.TrimSpace(value[open+1 : len(value)-1])
	if label == "" || context == "" {
		return value
	}
	return label + ": " + context
}

func normalizeGlobalKey(msg bubbletea.KeyMsg) bubbletea.KeyMsg {
	if msg.Type == bubbletea.KeySpace || msg.String() == " " {
		return bubbletea.KeyMsg{Type: bubbletea.KeyPgDown}
	}
	return msg
}

// breadcrumbLabel returns a short label for the view, used when updating crumbs on push.
func breadcrumbLabel(v viewstate.View) string {
	crumb := v.Breadcrumb()
	label := strings.TrimSpace(crumb)
	if open := strings.Index(label, "("); open > 0 {
		label = strings.TrimSpace(label[:open])
	}
	return label
}
