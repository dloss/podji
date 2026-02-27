package overlaypicker

import (
	"strings"
	"unicode"

	bubbletea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/dloss/podji/internal/ui/viewstate"
)

// SelectedMsg is emitted as a Cmd when the user confirms a selection.
type SelectedMsg struct {
	Kind  string // "namespace" or "context"
	Value string
}

type Picker struct {
	kind    string
	items   []string
	filter  string
	cursor  int
	width   int
	height  int
}

func New(kind string, items []string) *Picker {
	return &Picker{
		kind:  kind,
		items: items,
	}
}

func (p *Picker) SetSize(w, h int) {
	p.width = w
	p.height = h
}

func (p *Picker) filtered() []string {
	if p.filter == "" {
		return p.items
	}
	lower := strings.ToLower(p.filter)
	var result []string
	for _, item := range p.items {
		if strings.Contains(strings.ToLower(item), lower) {
			result = append(result, item)
		}
	}
	return result
}

func (p *Picker) clampCursor(list []string) {
	if len(list) == 0 {
		p.cursor = 0
		return
	}
	if p.cursor < 0 {
		p.cursor = 0
	}
	if p.cursor >= len(list) {
		p.cursor = len(list) - 1
	}
}

func (p *Picker) Init() bubbletea.Cmd { return nil }

func (p *Picker) Update(msg bubbletea.Msg) viewstate.Update {
	key, ok := msg.(bubbletea.KeyMsg)
	if !ok {
		return viewstate.Update{Action: viewstate.None, Next: p}
	}

	filtered := p.filtered()

	switch key.String() {
	case "esc":
		return viewstate.Update{Action: viewstate.Pop}
	case "enter":
		if len(filtered) > 0 {
			p.clampCursor(filtered)
			selected := filtered[p.cursor]
			kind := p.kind
			return viewstate.Update{
				Action: viewstate.Pop,
				Cmd: func() bubbletea.Msg {
					return SelectedMsg{Kind: kind, Value: selected}
				},
			}
		}
		return viewstate.Update{Action: viewstate.Pop}
	case "up", "k":
		p.cursor--
		p.clampCursor(filtered)
	case "down", "j":
		p.cursor++
		p.clampCursor(filtered)
	case "backspace", "ctrl+h":
		runes := []rune(p.filter)
		if len(runes) > 0 {
			p.filter = string(runes[:len(runes)-1])
			p.cursor = 0
			p.clampCursor(p.filtered())
		}
	default:
		if key.Type == bubbletea.KeyRunes {
			for _, r := range key.Runes {
				if unicode.IsPrint(r) {
					p.filter += string(r)
					p.cursor = 0
				}
			}
		}
	}

	return viewstate.Update{Action: viewstate.None, Next: p}
}

func (p *Picker) View() string {
	filtered := p.filtered()
	p.clampCursor(filtered)

	boxWidth := p.width - 4
	if boxWidth < 20 {
		boxWidth = 20
	}
	if boxWidth > 42 {
		boxWidth = 42
	}

	innerWidth := boxWidth - 2 // minus border

	// Title
	title := "  " + p.kind + "  "
	titleStyle := lipgloss.NewStyle().Bold(true)

	// Filter prompt
	filterLine := "> " + p.filter

	// Item list
	maxItems := p.height - 6 // leave room for border, title, filter, padding
	if maxItems < 1 {
		maxItems = 1
	}
	if len(filtered)+4 < p.height-4 {
		maxItems = len(filtered)
	}

	var lines []string
	lines = append(lines, titleStyle.Render(title))
	lines = append(lines, filterLine)
	lines = append(lines, strings.Repeat("─", innerWidth))

	start := 0
	if p.cursor >= maxItems {
		start = p.cursor - maxItems + 1
	}
	end := start + maxItems
	if end > len(filtered) {
		end = len(filtered)
	}
	if start > end {
		start = end
	}

	for i := start; i < end; i++ {
		item := filtered[i]
		if len([]rune(item)) > innerWidth-2 {
			item = string([]rune(item)[:innerWidth-3]) + "…"
		}
		if i == p.cursor {
			sel := lipgloss.NewStyle().
				Foreground(lipgloss.Color("0")).
				Background(lipgloss.Color("250")).
				Bold(true)
			lines = append(lines, sel.Render(" "+item+" "))
		} else {
			lines = append(lines, " "+item)
		}
	}

	if len(filtered) == 0 {
		lines = append(lines, lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render("  no matches"))
	}

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("241")).
		Width(innerWidth).
		Render(strings.Join(lines, "\n"))

	return lipgloss.Place(p.width, p.height, lipgloss.Center, lipgloss.Center, box)
}

func (p *Picker) Breadcrumb() string { return "" }
func (p *Picker) Footer() string     { return "" }
