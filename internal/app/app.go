package app

import (
	"strings"
	"unicode"

	bubbletea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/dloss/podji/internal/resources"
	"github.com/dloss/podji/internal/ui/helpview"
	"github.com/dloss/podji/internal/ui/listview"
	"github.com/dloss/podji/internal/ui/style"
	"github.com/dloss/podji/internal/ui/viewstate"
)

const (
	scopeContext   = 0
	scopeNamespace = 1
	scopeLens      = 2
)

type snapshot struct {
	stack  []viewstate.View
	crumbs []string
	lens   int
	scope  int
}

type Model struct {
	registry  *resources.Registry
	stack     []viewstate.View
	crumbs    []string
	lens      int
	scope     int
	history   []snapshot
	context   string
	namespace string
	errorMsg  string
	width     int
	height    int
}

type lens struct {
	name       string
	landingKey rune
}

var lenses = []lens{
	{name: "Apps", landingKey: 'W'},
	{name: "Network", landingKey: 'S'},
	{name: "Infrastructure", landingKey: 'O'},
}

type globalKeySuppresser interface {
	SuppressGlobalKeys() bool
}

type selectedBreadcrumbProvider interface {
	SelectedBreadcrumb() string
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
		lens:      0,
		scope:     scopeLens,
		context:   "default",
		namespace: "default",
	}
}

func (m Model) Init() bubbletea.Cmd {
	return m.top().Init()
}

func (m Model) Update(msg bubbletea.Msg) (bubbletea.Model, bubbletea.Cmd) {
	switch msg := msg.(type) {
	case bubbletea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.top().SetSize(m.width, m.availableHeight())
		return m, nil
	case bubbletea.KeyMsg:
		if suppresser, ok := m.top().(globalKeySuppresser); ok && suppresser.SuppressGlobalKeys() && msg.String() != "ctrl+c" {
			break
		}
		if msg.Type == bubbletea.KeyShiftTab || msg.String() == "shift+tab" || msg.String() == "backtab" {
			m.saveHistory()
			m.lens = (m.lens - 1 + len(lenses)) % len(lenses)
			m.switchToLensRoot()
			return m, nil
		}

		switch msg.String() {
		case "q", "ctrl+c":
			return m, bubbletea.Quit
		case "home", "pos1":
			m.saveHistory()
			m.switchToLensRoot()
			return m, nil
		case "shift+home", "shift+pos1":
			m.saveHistory()
			m.lens = 0
			m.switchToLensRoot()
			return m, nil
		case "tab":
			m.saveHistory()
			m.lens = (m.lens + 1) % len(lenses)
			m.switchToLensRoot()
			return m, nil
		case "backspace", "esc":
			if len(m.stack) > 1 {
				m.stack = m.stack[:len(m.stack)-1]
				m.crumbs = m.crumbs[:len(m.crumbs)-1]
				m.crumbs[len(m.crumbs)-1] = normalizeBreadcrumbPart(m.top().Breadcrumb())
			} else if m.scope == scopeLens {
				m.saveHistory()
				m.switchToScope(scopeNamespace)
			} else if m.scope == scopeNamespace || m.scope == scopeContext {
				m.restoreHistory()
			}
			return m, nil
		case "h", "left":
			if len(m.stack) > 1 {
				m.stack = m.stack[:len(m.stack)-1]
				m.crumbs = m.crumbs[:len(m.crumbs)-1]
				m.crumbs[len(m.crumbs)-1] = normalizeBreadcrumbPart(m.top().Breadcrumb())
			} else if m.scope == scopeLens {
				m.saveHistory()
				m.switchToScope(scopeNamespace)
			} else if m.scope == scopeNamespace {
				m.saveHistory()
				m.switchToScope(scopeContext)
			}
			return m, nil
		case "N":
			if m.scope != scopeNamespace {
				m.saveHistory()
				m.switchToScope(scopeNamespace)
			}
			return m, nil
		case "X":
			if m.scope != scopeContext {
				m.saveHistory()
				m.switchToScope(scopeContext)
			}
			return m, nil
		case "?":
			if _, isHelp := m.top().(*helpview.View); !isHelp {
				help := helpview.New()
				help.SetSize(m.width, m.availableHeight())
				m.stack = append(m.stack, help)
				m.crumbs = append(m.crumbs, m.crumbs[len(m.crumbs)-1])
			}
			return m, nil
		default:
			runes := []rune(msg.String())
			if len(runes) == 1 {
				key := runes[0]
				if res := m.registry.ResourceByKey(key); res != nil {
					m.saveHistory()
					if lensIndex, ok := m.lensByLandingKey(key); ok {
						m.lens = lensIndex
					}
					view := listview.New(res, m.registry)
					view.SetSize(m.width, m.availableHeight())
					m.stack = []viewstate.View{view}
					m.crumbs = []string{normalizeBreadcrumbPart(view.Breadcrumb())}
					return m, nil
				}
			}
		}
	}

	update := m.top().Update(msg)
	switch update.Action {
	case viewstate.Push:
		if m.scope == scopeNamespace || m.scope == scopeContext {
			if selected, ok := m.top().(selectedBreadcrumbProvider); ok {
				if value := normalizeBreadcrumbPart(selected.SelectedBreadcrumb()); value != "" {
					if idx := strings.Index(value, ": "); idx >= 0 {
						name := value[idx+2:]
						if m.scope == scopeNamespace {
							m.namespace = name
							resources.ActiveNamespace = name
						} else {
							m.context = name
						}
					}
				}
			}
			if m.scope == scopeNamespace {
				// Namespace changes always return to the active lens root.
				m.restoreHistory()
				m.switchToLensRoot()
				return m, nil
			}
			m.restoreHistory()
			return m, nil
		}
		if len(m.crumbs) > 0 {
			committed := m.crumbs[len(m.crumbs)-1]
			if selected, ok := m.top().(selectedBreadcrumbProvider); ok {
				if value := normalizeBreadcrumbPart(selected.SelectedBreadcrumb()); value != "" {
					committed = value
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
	default:
		m.stack[len(m.stack)-1] = update.Next
	}

	return m, update.Cmd
}

func (m Model) View() string {
	head := m.scopeLine() + "\n" + m.breadcrumbLine()
	body := m.top().View()
	footerLine1 := strings.TrimSpace(m.top().Footer())
	if m.width > 0 {
		footerLine1 = ansi.Truncate(footerLine1, m.width-2, "…")
	}
	footerLine2 := style.GlobalFooter(m.width)
	footer := footerLine1 + "\n" + footerLine2

	if m.height > 0 {
		body = clampViewLines(body, m.bodyHeightLimit())
	}

	sections := []string{head}
	if body != "" {
		sections = append(sections, body)
	}
	sections = append(sections, footer)
	if m.errorMsg != "" {
		sections = append([]string{style.ErrorBanner.Render(m.errorMsg)}, sections...)
	}

	return strings.Join(sections, "\n")
}

func (m Model) top() viewstate.View {
	return m.stack[len(m.stack)-1]
}

func (m *Model) saveHistory() {
	s := make([]viewstate.View, len(m.stack))
	copy(s, m.stack)
	c := make([]string, len(m.crumbs))
	copy(c, m.crumbs)
	m.history = append(m.history, snapshot{stack: s, crumbs: c, lens: m.lens, scope: m.scope})
}

func (m *Model) restoreHistory() bool {
	if len(m.history) == 0 {
		return false
	}
	last := m.history[len(m.history)-1]
	m.history = m.history[:len(m.history)-1]
	m.stack = last.stack
	m.crumbs = last.crumbs
	m.lens = last.lens
	m.scope = last.scope
	m.top().SetSize(m.width, m.availableHeight())
	return true
}

func (m *Model) switchToScope(scope int) {
	m.scope = scope
	var res resources.ResourceType
	switch scope {
	case scopeNamespace:
		res = m.registry.ResourceByKey('N')
	case scopeContext:
		res = m.registry.ResourceByKey('X')
	default:
		m.switchToLensRoot()
		return
	}
	if res == nil {
		return
	}
	view := listview.New(res, m.registry)
	view.SetSize(m.width, m.availableHeight())
	m.stack = []viewstate.View{view}
	m.crumbs = []string{normalizeBreadcrumbPart(view.Breadcrumb())}
}

func (m Model) scopeLine() string {
	sep := style.NavSep.Render(" > ")

	contextLabel := style.Scope.Render("Context: ")
	contextValue := style.ScopeValue.Render(m.context)
	if m.scope == scopeContext {
		contextLabel = style.ScopeActive.Render("Context: ")
		contextValue = style.ScopeActiveValue.Render(m.context)
	}

	nsLabel := style.Scope.Render("Namespace: ")
	nsValue := style.ScopeValue.Render(m.namespace)
	if m.scope == scopeNamespace {
		nsLabel = style.ScopeActive.Render("Namespace: ")
		nsValue = style.ScopeActiveValue.Render(m.namespace)
	}

	return contextLabel + contextValue + sep + nsLabel + nsValue
}

func (m Model) breadcrumbLine() string {
	if m.scope != scopeLens {
		return style.Scope.Render("[" + m.crumbs[0] + "]")
	}

	lensTag := style.Scope.Render("[" + lenses[m.lens].name + "]")
	if len(m.crumbs) <= 1 {
		return lensTag
	}

	sep := style.NavSep.Render(" > ")
	segments := make([]string, 0, len(m.crumbs)-1)
	for _, part := range m.crumbs[:len(m.crumbs)-1] {
		segments = append(segments, formatCrumb(part))
	}
	return lensTag + "  " + strings.Join(segments, sep)
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

func clampViewLines(view string, maxLines int) string {
	if maxLines <= 0 || view == "" {
		return ""
	}

	lines := strings.Split(view, "\n")
	if len(lines) <= maxLines {
		return view
	}
	return strings.Join(lines[:maxLines], "\n")
}

func (m *Model) switchToLensRoot() {
	m.scope = scopeLens
	l := lenses[m.lens]
	res := m.registry.ResourceByKey(l.landingKey)
	if res == nil {
		return
	}
	view := listview.New(res, m.registry)
	view.SetSize(m.width, m.availableHeight())
	m.stack = []viewstate.View{view}
	m.crumbs = []string{normalizeBreadcrumbPart(view.Breadcrumb())}
}

func (m Model) lensByLandingKey(key rune) (int, bool) {
	for idx, l := range lenses {
		if l.landingKey == key {
			return idx, true
		}
	}
	return 0, false
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
