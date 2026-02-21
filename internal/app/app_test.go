package app

import (
	"strings"
	"testing"

	bubbletea "github.com/charmbracelet/bubbletea"
	"github.com/dloss/podji/internal/ui/viewstate"
)

type overflowView struct{}

func (overflowView) Init() bubbletea.Cmd { return nil }

func (overflowView) Update(msg bubbletea.Msg) viewstate.Update {
	return viewstate.Update{Action: viewstate.None, Next: overflowView{}}
}

func (overflowView) View() string {
	return strings.Repeat("row\n", 20)
}

func (overflowView) Breadcrumb() string { return "workloads" }

func (overflowView) Footer() string { return "q quit" }

func (overflowView) SetSize(width, height int) {}

func TestViewClampsBodyToWindowHeight(t *testing.T) {
	m := Model{
		stack:     []viewstate.View{overflowView{}},
		crumbs:    []string{"workloads"},
		lens:      0,
		context:   "default",
		namespace: "default",
		height:    6,
	}

	rendered := m.View()
	lines := strings.Split(rendered, "\n")
	if len(lines) > m.height {
		t.Fatalf("expected <= %d lines, got %d", m.height, len(lines))
	}
	if !strings.Contains(lines[0], "CONTEXT:") || !strings.Contains(lines[0], "[Apps]") {
		t.Fatalf("expected nav line with context and lens, got %q", lines[0])
	}
}

func TestTabCyclesLensForward(t *testing.T) {
	m := New()

	updated, _ := m.Update(bubbletea.KeyMsg{Type: bubbletea.KeyTab})
	got := updated.(Model)

	if got.lens != 1 {
		t.Fatalf("expected lens 1 after tab, got %d", got.lens)
	}
}

func TestShiftTabCyclesLensBackwardFromFirst(t *testing.T) {
	m := New()

	updated, _ := m.Update(bubbletea.KeyMsg{Type: bubbletea.KeyShiftTab})
	got := updated.(Model)

	want := len(lenses) - 1
	if got.lens != want {
		t.Fatalf("expected lens %d after shift+tab from first, got %d", want, got.lens)
	}
}
