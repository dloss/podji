package app

import (
	"strings"

	bubbletea "github.com/charmbracelet/bubbletea"
	"github.com/dloss/kubira/internal/resources"
	"github.com/dloss/kubira/internal/ui/listview"
	"github.com/dloss/kubira/internal/ui/style"
	"github.com/dloss/kubira/internal/ui/viewstate"
)

type Model struct {
	registry  *resources.Registry
	stack     []viewstate.View
	context   string
	namespace string
	errorMsg  string
	width     int
	height    int
}

func New() Model {
	registry := resources.DefaultRegistry()
	pods := registry.ResourceByKey('P')
	root := listview.New(pods, registry)

	return Model{
		registry:  registry,
		stack:     []viewstate.View{root},
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
		switch msg.String() {
		case "q", "ctrl+c":
			return m, bubbletea.Quit
		case "backspace", "h", "left":
			if len(m.stack) > 1 {
				m.stack = m.stack[:len(m.stack)-1]
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
	footer := style.Footer.Render(m.top().Footer())

	sections := []string{head, body, footer}
	if m.errorMsg != "" {
		sections = append([]string{style.ErrorBanner.Render(m.errorMsg)}, sections...)
	}

	return strings.Join(sections, "\n\n")
}

func (m Model) top() viewstate.View {
	return m.stack[len(m.stack)-1]
}

func (m Model) breadcrumb() string {
	parts := []string{m.context, m.namespace}
	for _, view := range m.stack {
		parts = append(parts, view.Breadcrumb())
	}
	return strings.Join(parts, " > ")
}

func (m Model) availableHeight() int {
	if m.height == 0 {
		return 0
	}

	extra := 4
	if m.errorMsg != "" {
		extra = 6
	}

	height := m.height - extra
	if height < 1 {
		return 1
	}
	return height
}
