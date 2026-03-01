package commandbar

import (
	"strings"

	bubbletea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/dloss/podji/internal/ui/style"
)

type SubmitMsg struct{ Value string }

type Model struct {
	input   string
	error   string
	history []string
	idx     int
	width   int
}

func New() *Model { return &Model{idx: -1} }

func (m *Model) SetSize(width int)   { m.width = width }
func (m *Model) SetError(err string) { m.error = err }
func (m *Model) Complete(suffix string) {
	if suffix == "" {
		return
	}
	m.input += suffix
}

func (m *Model) Update(msg bubbletea.KeyMsg) (*Model, bubbletea.Cmd, bool) {
	switch msg.String() {
	case "esc":
		return m, nil, true
	case "enter":
		val := strings.TrimSpace(m.input)
		if val != "" {
			m.history = append(m.history, val)
		}
		m.idx = -1
		m.input = ""
		m.error = ""
		return m, func() bubbletea.Msg { return SubmitMsg{Value: val} }, true
	case "backspace", "ctrl+h":
		r := []rune(m.input)
		if len(r) > 0 {
			m.input = string(r[:len(r)-1])
		}
	case "up":
		if len(m.history) == 0 {
			return m, nil, false
		}
		if m.idx == -1 {
			m.idx = len(m.history) - 1
		} else if m.idx > 0 {
			m.idx--
		}
		m.input = m.history[m.idx]
	case "down":
		if len(m.history) == 0 || m.idx == -1 {
			return m, nil, false
		}
		if m.idx < len(m.history)-1 {
			m.idx++
			m.input = m.history[m.idx]
		} else {
			m.idx = -1
			m.input = ""
		}
	default:
		if msg.Type == bubbletea.KeyRunes {
			m.input += string(msg.Runes)
		}
	}
	return m, nil, false
}

func (m *Model) Input() string { return m.input }

func (m *Model) View(suggestion string) string {
	line := ": " + m.input
	if suggestion != "" {
		line += style.Muted.Render(suggestion)
	}
	if m.error != "" {
		line += strings.Repeat(" ", 2) + style.Warning.Render(m.error)
	}
	line += "█"
	if m.width > 0 {
		line = ansi.Truncate(line, m.width-1, "…")
	}
	return line
}
