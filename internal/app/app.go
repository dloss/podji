package app

import (
	"strings"
	"unicode"

	bubbletea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/dloss/podji/internal/resources"
	"github.com/dloss/podji/internal/ui/helpview"
	"github.com/dloss/podji/internal/ui/listview"
	"github.com/dloss/podji/internal/ui/overlaypicker"
	"github.com/dloss/podji/internal/ui/relatedview"
	"github.com/dloss/podji/internal/ui/resourcebrowser"
	"github.com/dloss/podji/internal/ui/style"
	"github.com/dloss/podji/internal/ui/viewstate"
)

type Model struct {
	registry   *resources.Registry
	stack      []viewstate.View
	crumbs     []string
	overlay    *overlaypicker.Picker
	side       viewstate.View
	sideActive bool
	context    string
	namespace  string
	errorMsg   string
	width      int
	height     int
}

type globalKeySuppresser interface {
	SuppressGlobalKeys() bool
}

func New() Model {
	registry := resources.DefaultRegistry()
	workloads := registry.ResourceByKey('W')
	root := listview.New(workloads, registry)
	rootCrumb := normalizeBreadcrumbPart(root.Breadcrumb())

	return Model{
		registry:  registry,
		stack:     []viewstate.View{root},
		crumbs:    []string{rootCrumb},
		context:   "default",
		namespace: "default",
	}
}

func (m Model) Init() bubbletea.Cmd {
	return m.top().Init()
}

func (m Model) Update(msg bubbletea.Msg) (bubbletea.Model, bubbletea.Cmd) {
	routedMsg := msg

	// Route all input to overlay when active.
	if m.overlay != nil {
		if _, ok := msg.(bubbletea.KeyMsg); ok {
			update := m.overlay.Update(msg)
			if update.Action == viewstate.Pop {
				m.overlay = nil
			}
			return m, update.Cmd
		}
	}

	switch msg := msg.(type) {
	case bubbletea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.top().SetSize(m.mainWidth(), m.availableHeight())
		if m.side != nil {
			m.side.SetSize(m.sideContentWidth(), m.availableHeight())
		}
		if m.overlay != nil {
			m.overlay.SetSize(m.width, m.height-1)
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
		// Reload workloads so the new namespace/context takes effect.
		if res := m.registry.ResourceByKey('W'); res != nil {
			view := listview.New(res, m.registry)
			view.SetSize(m.mainWidth(), m.availableHeight())
			m.stack = []viewstate.View{view}
			m.crumbs = []string{normalizeBreadcrumbPart(view.Breadcrumb())}
		}
		m = m.withRefreshedSide()
		return m, nil

	case bubbletea.KeyMsg:
		if suppresser, ok := m.top().(globalKeySuppresser); ok && suppresser.SuppressGlobalKeys() && msg.String() != "ctrl+c" {
			break
		}
		msg = normalizeGlobalKey(msg)
		routedMsg = msg

		switch msg.String() {
		case "q", "ctrl+c":
			return m, bubbletea.Quit
		case "backspace", "esc":
			if len(m.stack) > 1 {
				m.stack = m.stack[:len(m.stack)-1]
				m.crumbs = m.crumbs[:len(m.crumbs)-1]
				m.crumbs[len(m.crumbs)-1] = normalizeBreadcrumbPart(m.top().Breadcrumb())
			}
			m = m.withRefreshedSide()
			return m, nil
		case "h", "left":
			if len(m.stack) > 1 {
				m.stack = m.stack[:len(m.stack)-1]
				m.crumbs = m.crumbs[:len(m.crumbs)-1]
				m.crumbs[len(m.crumbs)-1] = normalizeBreadcrumbPart(m.top().Breadcrumb())
			}
			m = m.withRefreshedSide()
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
			browser.SetSize(m.mainWidth(), m.availableHeight())
			m.stack = []viewstate.View{browser}
			m.crumbs = []string{"resources"}
			return m, nil
		case "r":
			if m.side != nil {
				// Close side panel, restore full width to main.
				m.side = nil
				m.sideActive = false
				if f, ok := m.top().(viewstate.Focusable); ok {
					f.SetFocused(true)
				}
				m.top().SetSize(m.width, m.availableHeight())
				m.notifySideState()
			} else {
				// Open side panel. Assign m.side first so that sideWidth() /
				// mainWidth() return the correct 40/60 split when SetSize is called.
				m.side = relatedview.NewForSelection(m.top())
				m.sideActive = true
				if f, ok := m.side.(viewstate.Focusable); ok {
					f.SetFocused(true)
				}
				if f, ok := m.top().(viewstate.Focusable); ok {
					f.SetFocused(false)
				}
				m.side.SetSize(m.sideContentWidth(), m.availableHeight())
				m.top().SetSize(m.mainWidth(), m.availableHeight())
				m.notifySideState()
			}
			return m, nil
		case "tab":
			if m.side != nil {
				m.sideActive = !m.sideActive
				if f, ok := m.side.(viewstate.Focusable); ok {
					f.SetFocused(m.sideActive)
				}
				if f, ok := m.top().(viewstate.Focusable); ok {
					f.SetFocused(!m.sideActive)
				}
			}
			return m, nil
		case "?":
			if _, isHelp := m.top().(*helpview.View); !isHelp {
				help := helpview.New()
				help.SetSize(m.mainWidth(), m.availableHeight())
				m.stack = append(m.stack, help)
				m.crumbs = append(m.crumbs, m.crumbs[len(m.crumbs)-1])
			}
			return m, nil
		default:
			runes := []rune(msg.String())
			if len(runes) == 1 {
				key := runes[0]
				if res := m.registry.ResourceByKey(key); res != nil {
					view := listview.New(res, m.registry)
					view.SetSize(m.mainWidth(), m.availableHeight())
					m.stack = []viewstate.View{view}
					m.crumbs = []string{normalizeBreadcrumbPart(view.Breadcrumb())}
					m = m.withRefreshedSide()
					return m, nil
				}
			}
		}
	}

	// Route to side panel when it's focused.
	if m.sideActive && m.side != nil {
		update := m.side.Update(routedMsg)
		switch update.Action {
		case viewstate.Push:
			// Navigation from side panel goes to the main stack.
			if len(m.crumbs) > 0 {
				if sel, ok := m.side.(viewstate.SelectionProvider); ok {
					if item := sel.SelectedItem(); item.Name != "" {
						kind := item.Kind
						if kind == "" {
							kind = "item"
						}
						val := normalizeBreadcrumbPart(strings.ToLower(kind) + ": " + item.Name)
						if val != "" {
							m.crumbs[len(m.crumbs)-1] = val
						}
					}
				}
			}
			update.Next.SetSize(m.mainWidth(), m.availableHeight())
			m.stack = append(m.stack, update.Next)
			m.crumbs = append(m.crumbs, normalizeBreadcrumbPart(update.Next.Breadcrumb()))
			m.sideActive = false // focus follows to main
			if f, ok := m.top().(viewstate.Focusable); ok {
				f.SetFocused(true)
			}
		case viewstate.Pop:
			// Side panel closed itself (Esc).
			m.side = nil
			m.sideActive = false
			if f, ok := m.top().(viewstate.Focusable); ok {
				f.SetFocused(true)
			}
			m.top().SetSize(m.width, m.availableHeight())
			m.notifySideState()
		case viewstate.Replace:
			update.Next.SetSize(m.sideWidth(), m.availableHeight())
			if f, ok := update.Next.(viewstate.Focusable); ok {
				f.SetFocused(m.sideActive)
			}
			m.side = update.Next
		case viewstate.OpenRelated:
			// Side panel requested a related panel for a sub-item — ignore.
		default:
			m.side = update.Next
		}
		return m, update.Cmd
	}

	// Capture the selected item name before the update so we can detect
	// cursor movement and refresh the side panel.
	prevName := ""
	if m.side != nil {
		if sel, ok := m.top().(viewstate.SelectionProvider); ok {
			prevName = sel.SelectedItem().Name
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
		update.Next.SetSize(m.mainWidth(), m.availableHeight())
		m.stack = append(m.stack, update.Next)
		m.crumbs = append(m.crumbs, normalizeBreadcrumbPart(update.Next.Breadcrumb()))
		m = m.withRefreshedSide()
	case viewstate.Pop:
		if len(m.stack) > 1 {
			m.stack = m.stack[:len(m.stack)-1]
			m.crumbs = m.crumbs[:len(m.crumbs)-1]
			m.crumbs[len(m.crumbs)-1] = normalizeBreadcrumbPart(m.top().Breadcrumb())
		}
		m = m.withRefreshedSide()
	case viewstate.Replace:
		update.Next.SetSize(m.mainWidth(), m.availableHeight())
		m.stack[len(m.stack)-1] = update.Next
		m.crumbs[len(m.crumbs)-1] = normalizeBreadcrumbPart(update.Next.Breadcrumb())
		m = m.withRefreshedSide()
	case viewstate.OpenRelated:
		if m.side == nil {
			// Assign m.side first so sideWidth()/mainWidth() use the 40/60 split.
			m.side = relatedview.NewForSelection(m.top())
			m.side.SetSize(m.sideContentWidth(), m.availableHeight())
			m.top().SetSize(m.mainWidth(), m.availableHeight())
		}
		m.sideActive = true
		if f, ok := m.side.(viewstate.Focusable); ok {
			f.SetFocused(true)
		}
		if f, ok := m.top().(viewstate.Focusable); ok {
			f.SetFocused(false)
		}
		m.notifySideState()
	default:
		m.stack[len(m.stack)-1] = update.Next
		if m.side != nil {
			if sel, ok := m.top().(viewstate.SelectionProvider); ok {
				if sel.SelectedItem().Name != prevName {
					m = m.withRefreshedSide()
				}
			}
		}
	}

	return m, update.Cmd
}

func (m Model) renderHeader() string {
	head := m.scopeLine() + "\n" + m.breadcrumbLine()
	if m.errorMsg != "" {
		return style.ErrorBanner.Render(m.errorMsg) + "\n" + head
	}
	return head
}

func (m Model) renderBody() string {
	body := m.top().View()
	if m.height > 0 {
		body = fitViewLines(body, m.bodyHeightLimit())
	}
	return body
}

func (m Model) renderFooter() string {
	if m.sideActive && m.side != nil {
		return m.side.Footer()
	}
	return m.top().Footer()
}

func (m Model) renderMain() string {
	header := m.renderHeader()
	body := m.renderBody()
	footer := m.renderFooter()

	sections := []string{header}
	if body != "" {
		sections = append(sections, body)
	}
	sections = append(sections, footer)

	return strings.Join(sections, "\n")
}

func (m Model) View() string {
	main := m.renderMain()
	if m.overlay != nil {
		// Composite the picker box over the full main view so the table stays visible.
		return compositeOverlay(main, m.overlay.View(), m.overlay.AnchorX(), 1)
	}
	if m.side != nil {
		header := m.renderHeader()
		mainBody := m.renderBody()
		bodyHeight := m.bodyHeightLimit()
		sideBody := fitViewLines(m.side.View(), bodyHeight)
		sep := sideSeparator(bodyHeight)
		splitBody := lipgloss.JoinHorizontal(lipgloss.Top, mainBody, sep, sideBody)
		// Set footer width to full terminal width so the footer isn't truncated
		// at the panel's narrower width.
		setFooterWidth(m.top(), m.width)
		setFooterWidth(m.side, m.width)
		footer := m.renderFooter()
		setFooterWidth(m.top(), 0)
		setFooterWidth(m.side, 0)
		return header + "\n" + splitBody + "\n" + footer
	}
	return main
}

// sideSeparator returns a single-column string of │ glyphs used to visually
// divide the main panel from the related side panel.
func sideSeparator(height int) string {
	if height <= 0 {
		height = 40
	}
	sep := style.Muted.Render("│")
	rows := make([]string, height)
	for i := range rows {
		rows[i] = sep
	}
	return strings.Join(rows, "\n")
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

func (m Model) mainWidth() int {
	if m.side != nil {
		return (m.width * 60) / 100
	}
	return m.width
}

func (m Model) sideWidth() int {
	return m.width - m.mainWidth()
}

// sideContentWidth is the width given to the side panel's content, one less
// than sideWidth() to account for the separator column.
func (m Model) sideContentWidth() int {
	w := m.sideWidth() - 1
	if w < 1 {
		return 1
	}
	return w
}

// setFooterWidth sets the footer width override on views that support it.
func setFooterWidth(v viewstate.View, w int) {
	if f, ok := v.(viewstate.FooterWidthSetter); ok {
		f.SetFooterWidth(w)
	}
}

// notifySideState tells the top-of-stack view whether the side panel is
// currently open, so it can adjust its footer hints.
func (m Model) notifySideState() {
	type sideStateSetter interface {
		SetSideOpen(bool)
	}
	if s, ok := m.top().(sideStateSetter); ok {
		s.SetSideOpen(m.side != nil)
	}
}

// withRefreshedSide rebuilds the side panel from the current top-of-stack
// selection so the related panel stays in sync after navigation.  If the new
// top view has no selection the side panel is closed.
func (m Model) withRefreshedSide() Model {
	if m.side == nil {
		return m
	}
	if sel, ok := m.top().(viewstate.SelectionProvider); ok && sel.SelectedItem().Name != "" {
		m.side = relatedview.NewForSelection(m.top())
		m.side.SetSize(m.sideContentWidth(), m.availableHeight())
		if f, ok := m.side.(viewstate.Focusable); ok {
			f.SetFocused(m.sideActive)
		}
	} else {
		// New top view has no selection; close the side panel.
		m.side = nil
		m.sideActive = false
		m.top().SetSize(m.width, m.availableHeight())
	}
	m.notifySideState()
	return m
}

func (m Model) scopeLine() string {
	sep := style.NavSep.Render(" > ")

	contextLabel := style.Scope.Render("Context: ")
	contextValue := style.ScopeValue.Render(m.context)

	nsLabel := style.Scope.Render("Namespace: ")
	nsValue := style.ScopeValue.Render(m.namespace)

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

func (m Model) availableHeight() int {
	if m.height == 0 {
		return 0
	}

	extra := 4 // 2 header lines + 2 footer lines
	if m.errorMsg != "" {
		extra = 5
	}

	height := m.height - extra
	if height < 1 {
		return 1
	}
	return height
}

func (m Model) bodyHeightLimit() int {
	if m.height <= 0 {
		return 0
	}

	reserved := 4 // 2 header lines + 2 footer lines
	if m.errorMsg != "" {
		reserved++
	}

	limit := m.height - reserved
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
