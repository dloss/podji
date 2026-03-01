package columnpicker

import (
	"strings"

	bubbletea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/dloss/podji/internal/resources"
	"github.com/dloss/podji/internal/ui/viewstate"
)

// PickedMsg is emitted as a Cmd when the user confirms a column selection.
type PickedMsg struct {
	ResourceName string
	Visible      []string // column IDs in pool order
}

type rowKind int

const (
	rowColumn rowKind = iota
	rowHeader
)

type pickerRow struct {
	kind       rowKind
	headerText string
	col        resources.TableColumn
	checked    bool
	locked     bool // cannot be toggled (NAME, NAMESPACE)
	isWide     bool // wide-only column, shown with [wide] tag
}

// Picker is a floating overlay for selecting visible table columns.
type Picker struct {
	resourceName string
	rows         []pickerRow
	pool         []resources.TableColumn // ordered pool for emitting results
	cursor       int
	initial      []string // snapshot for reset
	width        int
	height       int
}

// New creates a column picker.
//   - pool: all resource-defined columns (normal + wide-only extras), in display order
//   - labelPool: dynamic label-derived columns
//   - current: currently active column IDs
func New(resourceName string, pool, labelPool []resources.TableColumn, current []string) *Picker {
	currentSet := make(map[string]bool, len(current))
	for _, id := range current {
		currentSet[id] = true
	}

	// Determine which column IDs are "default" (non-wide, resource-defined).
	defaultIDs := make(map[string]bool)
	for _, col := range pool {
		if col.Default {
			defaultIDs[col.ID] = true
		}
	}

	var rows []pickerRow

	// Standard + wide columns section.
	rows = append(rows, pickerRow{kind: rowHeader, headerText: "standard"})
	for _, col := range pool {
		locked := col.ID == "name" || col.ID == "namespace"
		wide := !defaultIDs[col.ID] && col.ID != "name" && col.ID != "namespace"
		checked := currentSet[col.ID] || locked
		rows = append(rows, pickerRow{
			kind:    rowColumn,
			col:     col,
			checked: checked,
			locked:  locked,
			isWide:  wide,
		})
	}

	// Label columns section.
	if len(labelPool) > 0 {
		rows = append(rows, pickerRow{kind: rowHeader, headerText: "labels"})
		for _, col := range labelPool {
			rows = append(rows, pickerRow{
				kind:    rowColumn,
				col:     col,
				checked: currentSet[col.ID],
			})
		}
	}

	// Snapshot current selection for reset.
	initial := make([]string, len(current))
	copy(initial, current)

	p := &Picker{
		resourceName: resourceName,
		rows:         rows,
		pool:         append(pool, labelPool...),
		initial:      initial,
	}
	p.cursor = p.firstSelectable()
	return p
}

func (p *Picker) SetSize(w, h int) {
	p.width = w
	p.height = h
}

// AnchorX returns the left-edge column for overlay compositing (right side of screen).
func (p *Picker) AnchorX() int {
	boxWidth := p.boxWidth()
	if p.width > 0 && p.width-boxWidth > 0 {
		return p.width - boxWidth
	}
	return 0
}

func (p *Picker) boxWidth() int {
	w := 36
	if p.width > 0 && w > p.width-4 {
		w = p.width - 4
	}
	if w < 24 {
		w = 24
	}
	return w
}

func (p *Picker) firstSelectable() int {
	for i, row := range p.rows {
		if row.kind == rowColumn && !row.locked {
			return i
		}
	}
	return 0
}

func (p *Picker) isSelectable(i int) bool {
	if i < 0 || i >= len(p.rows) {
		return false
	}
	return p.rows[i].kind == rowColumn && !p.rows[i].locked
}

func (p *Picker) moveCursor(delta int) {
	next := p.cursor + delta
	for next >= 0 && next < len(p.rows) {
		if p.isSelectable(next) {
			p.cursor = next
			return
		}
		next += delta
	}
}

// visibleIDs returns the column IDs in pool order for currently checked rows.
func (p *Picker) visibleIDs() []string {
	checkedSet := make(map[string]bool)
	for _, row := range p.rows {
		if row.kind == rowColumn && row.checked {
			checkedSet[row.col.ID] = true
		}
	}
	var result []string
	// Emit in pool order (preserves canonical ordering).
	for _, col := range p.pool {
		if checkedSet[col.ID] {
			result = append(result, col.ID)
		}
	}
	return result
}

// resetToInitial restores checkboxes to the snapshot taken at construction.
func (p *Picker) resetToInitial() {
	initialSet := make(map[string]bool, len(p.initial))
	for _, id := range p.initial {
		initialSet[id] = true
	}
	for i := range p.rows {
		if p.rows[i].kind == rowColumn {
			p.rows[i].checked = initialSet[p.rows[i].col.ID] || p.rows[i].locked
		}
	}
}

func (p *Picker) Init() bubbletea.Cmd { return nil }

func (p *Picker) Update(msg bubbletea.Msg) viewstate.Update {
	key, ok := msg.(bubbletea.KeyMsg)
	if !ok {
		return viewstate.Update{Action: viewstate.None}
	}

	switch key.String() {
	case "esc":
		return viewstate.Update{Action: viewstate.Pop}

	case "enter":
		visible := p.visibleIDs()
		resourceName := p.resourceName
		return viewstate.Update{
			Action: viewstate.Pop,
			Cmd: func() bubbletea.Msg {
				return PickedMsg{ResourceName: resourceName, Visible: visible}
			},
		}

	case "r":
		p.resetToInitial()

	case "up", "k":
		p.moveCursor(-1)

	case "down", "j":
		p.moveCursor(1)

	case " ":
		if p.isSelectable(p.cursor) {
			p.rows[p.cursor].checked = !p.rows[p.cursor].checked
		}
	}

	return viewstate.Update{Action: viewstate.None}
}

func (p *Picker) View() string {
	innerWidth := p.boxWidth() - 2 // minus border

	var lines []string
	titleStyle := lipgloss.NewStyle().Bold(true)
	lines = append(lines, titleStyle.Render("  columns  "))
	lines = append(lines, strings.Repeat("─", innerWidth))

	muted := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	cursorStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("0")).
		Background(lipgloss.Color("250")).
		Bold(true)
	lockedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	wideTag := muted.Render(" [wide]")

	maxItems := p.height - 8 // reserve space for border, title, sep, footer
	if maxItems < 3 {
		maxItems = 3
	}

	// Determine scroll window around cursor.
	start := 0
	if p.cursor-start >= maxItems {
		start = p.cursor - maxItems + 1
	}
	end := start + maxItems
	if end > len(p.rows) {
		end = len(p.rows)
	}
	if start > end {
		start = end
	}

	for i := start; i < end; i++ {
		row := p.rows[i]
		switch row.kind {
		case rowHeader:
			sep := muted.Render("── " + row.headerText + " " + strings.Repeat("─", innerWidth-len(row.headerText)-4))
			lines = append(lines, sep)

		case rowColumn:
			var checkbox string
			if row.locked {
				checkbox = "•"
			} else if row.checked {
				checkbox = "✓"
			} else {
				checkbox = "○"
			}

			name := row.col.Name
			if len([]rune(name)) > innerWidth-6 {
				name = string([]rune(name)[:innerWidth-7]) + "…"
			}

			var line string
			if row.locked {
				line = "  " + checkbox + " " + lockedStyle.Render(name)
			} else {
				line = "  " + checkbox + " " + name
				if row.isWide {
					line += wideTag
				}
			}

			if i == p.cursor {
				// Pad to inner width before applying cursor style.
				plain := lipgloss.NewStyle().Render(line)
				runeCount := len([]rune(lipgloss.NewStyle().Render(plain)))
				if runeCount < innerWidth {
					plain += strings.Repeat(" ", innerWidth-runeCount)
				}
				line = cursorStyle.Render("▶" + plain[1:])
			}

			lines = append(lines, line)
		}
	}

	lines = append(lines, strings.Repeat("─", innerWidth))
	footer := muted.Render("spc toggle  enter apply  r reset  esc cancel")
	if len([]rune(footer)) > innerWidth {
		footer = muted.Render("spc toggle  enter apply")
	}
	lines = append(lines, footer)

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("241")).
		Width(innerWidth).
		Render(strings.Join(lines, "\n"))

	return box
}

func (p *Picker) Breadcrumb() string { return "" }
func (p *Picker) Footer() string     { return "" }
