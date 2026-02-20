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

func New() Model {
	registry := resources.DefaultRegistry()
	workloads := registry.ResourceByKey('W')
	root := listview.New(workloads, registry)

	return Model{
		registry:  registry,
		stack:     []viewstate.View{root},
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
		case "tab":
			m.lens = (m.lens + 1) % len(lenses)
			m.switchToLensRoot()
			return m, nil
		case "backspace", "h", "left":
			if len(m.stack) > 1 {
				m.stack = m.stack[:len(m.stack)-1]
			}
			return m, nil
		case "r":
			m.errorMsg = "related panel not wired yet"
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
					return m, nil
				}
			}
		}
	}

	update := m.top().Update(msg)
	switch update.Action {
	case viewstate.Push:
		update.Next.SetSize(m.width, m.availableHeight())
		m.stack = append(m.stack, update.Next)
	case viewstate.Pop:
		if len(m.stack) > 1 {
			m.stack = m.stack[:len(m.stack)-1]
		}
	case viewstate.Replace:
		update.Next.SetSize(m.width, m.availableHeight())
		m.stack[len(m.stack)-1] = update.Next
	default:
		m.stack[len(m.stack)-1] = update.Next
	}

	return m, update.Cmd
}

func (m Model) View() string {
	breadcrumb := m.breadcrumb()
	head := style.Header.Render(breadcrumb)
	body := m.top().View()
	footer := style.Footer.Render(strings.TrimSpace(m.top().Footer()))

	sections := []string{head, body, footer}
	if m.errorMsg != "" {
		sections = append([]string{style.ErrorBanner.Render(m.errorMsg)}, sections...)
	}

	return strings.Join(sections, "\n")
}

func (m Model) top() viewstate.View {
	return m.stack[len(m.stack)-1]
}

func (m Model) breadcrumb() string {
	parts := []string{lenses[m.lens].name, "ctx:" + m.context, "ns:" + m.namespace}
	for _, view := range m.stack {
		parts = append(parts, titleCase(view.Breadcrumb()))
	}
	return strings.Join(parts, " > ")
}

func (m Model) availableHeight() int {
	if m.height == 0 {
		return 0
	}

	extra := 2
	if m.errorMsg != "" {
		extra = 3
	}

	height := m.height - extra
	if height < 1 {
		return 1
	}
	return height
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
