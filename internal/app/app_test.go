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
	if !strings.Contains(lines[0], "Apps") {
		t.Fatalf("expected scope line to be preserved, got %q", lines[0])
	}
	if !strings.Contains(lines[1], "Workloads") {
		t.Fatalf("expected breadcrumb line to be preserved, got %q", lines[1])
	}
}
