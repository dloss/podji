package app

import (
	"strings"
	"unicode"

	bubbletea "github.com/charmbracelet/bubbletea"
	"github.com/dloss/podji/internal/resources"
	"github.com/dloss/podji/internal/ui/listview"
	"github.com/dloss/podji/internal/ui/style"
	"github.com/dloss/podji/internal/ui/viewstate"
)

type Model struct {
	registry  *resources.Registry
	stack     []viewstate.View
	crumbs    []string
	lens      int
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

type nextBreadcrumbPreviewer interface {
	NextBreadcrumb() string
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

		switch msg.String() {
		case "q", "ctrl+c":
			return m, bubbletea.Quit
		case "home", "pos1":
			m.switchToLensRoot()
			return m, nil
		case "shift+home", "shift+pos1":
			m.lens = 0
			m.switchToLensRoot()
			return m, nil
		case "tab":
			m.lens = (m.lens + 1) % len(lenses)
			m.switchToLensRoot()
			return m, nil
		case "backspace", "h", "left":
			if len(m.stack) > 1 {
				m.stack = m.stack[:len(m.stack)-1]
				m.crumbs = m.crumbs[:len(m.crumbs)-1]
				m.crumbs[len(m.crumbs)-1] = normalizeBreadcrumbPart(m.top().Breadcrumb())
			}
			return m, nil
		case "n":
			m.errorMsg = "namespace picker not wired yet"
			return m, nil
		case "x":
			m.errorMsg = "context picker not wired yet"
			return m, nil
		default:
			runes := []rune(msg.String())
			if len(runes) == 1 {
				key := runes[0]
				if res := m.registry.ResourceByKey(key); res != nil {
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
	head := strings.Join([]string{
		style.Scope.Render(m.scope()),
		m.breadcrumb(),
	}, "\n")
	body := m.top().View()
	footer := style.Footer.Render(strings.TrimSpace(m.top().Footer() + "  home top  shift+home default"))

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

func (m Model) breadcrumb() string {
	if len(m.crumbs) == 0 {
		return ""
	}
	if len(m.stack) == 1 {
		base := style.Active.Render(titleCase(normalizeBreadcrumbPart(m.top().Breadcrumb())))
		if previewer, ok := m.top().(nextBreadcrumbPreviewer); ok {
			next := strings.TrimSpace(previewer.NextBreadcrumb())
			if next != "" {
				base += style.Muted.Render(" > " + titleCase(normalizeBreadcrumbPart(next)))
			}
		}
		return base
	}

	segments := make([]string, 0, len(m.crumbs))
	for idx, part := range m.crumbs {
		rendered := titleCase(part)
		if idx == len(m.crumbs)-1 {
			segments = append(segments, style.Active.Render(rendered))
			continue
		}
		segments = append(segments, style.Crumb.Render(rendered))
	}

	hierarchy := strings.Join(segments, style.Crumb.Render(" > "))
	if previewer, ok := m.top().(nextBreadcrumbPreviewer); ok {
		next := strings.TrimSpace(previewer.NextBreadcrumb())
		if next != "" {
			hierarchy += style.Muted.Render(" > " + titleCase(normalizeBreadcrumbPart(next)))
		}
	}
	return hierarchy
}

func (m Model) scope() string {
	return strings.Join([]string{
		lenses[m.lens].name,
		"ctx:" + m.context,
		"ns:" + m.namespace,
	}, "   ")
}

func (m Model) availableHeight() int {
	if m.height == 0 {
		return 0
	}

	extra := 3
	if m.errorMsg != "" {
		extra = 4
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

	reserved := 3 // 2 header lines + 1 footer line
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
