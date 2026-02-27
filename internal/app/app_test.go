package app

import (
	"strings"
	"testing"

	bubbletea "github.com/charmbracelet/bubbletea"
	"github.com/dloss/podji/internal/ui/overlaypicker"
	"github.com/dloss/podji/internal/ui/viewstate"
)

type overflowView struct{}
type shortView struct{}
type keySpyView struct {
	lastKey  string
	suppress bool
}

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

func (shortView) Init() bubbletea.Cmd { return nil }

func (shortView) Update(msg bubbletea.Msg) viewstate.Update {
	return viewstate.Update{Action: viewstate.None, Next: shortView{}}
}

func (shortView) View() string { return "line 1\nline 2" }

func (shortView) Breadcrumb() string { return "workloads" }

func (shortView) Footer() string { return "status\nq quit" }

func (shortView) SetSize(width, height int) {}

func (v *keySpyView) Init() bubbletea.Cmd { return nil }

func (v *keySpyView) Update(msg bubbletea.Msg) viewstate.Update {
	if key, ok := msg.(bubbletea.KeyMsg); ok {
		v.lastKey = key.String()
	}
	return viewstate.Update{Action: viewstate.None, Next: v}
}

func (*keySpyView) View() string       { return "" }
func (*keySpyView) Breadcrumb() string { return "workloads" }
func (*keySpyView) Footer() string     { return "status\nq quit" }
func (*keySpyView) SetSize(width, height int) {}
func (v *keySpyView) SuppressGlobalKeys() bool {
	return v.suppress
}

func TestViewClampsBodyToWindowHeight(t *testing.T) {
	m := Model{
		stack:     []viewstate.View{overflowView{}},
		crumbs:    []string{"workloads"},
		context:   "default",
		namespace: "default",
		height:    6,
	}

	rendered := m.View()
	lines := strings.Split(rendered, "\n")
	if len(lines) > m.height {
		t.Fatalf("expected <= %d lines, got %d", m.height, len(lines))
	}
	if !strings.Contains(lines[0], "Context:") || !strings.Contains(lines[0], "Namespace:") {
		t.Fatalf("expected scope line with context and namespace, got %q", lines[0])
	}
	if !strings.Contains(lines[1], "[Workload]") {
		t.Fatalf("expected breadcrumb line with root resource tag, got %q", lines[1])
	}
}

func TestViewPadsBodyToKeepFooterAtBottom(t *testing.T) {
	m := Model{
		stack:     []viewstate.View{shortView{}},
		crumbs:    []string{"workloads"},
		context:   "default",
		namespace: "default",
		height:    8,
	}

	rendered := m.View()
	lines := strings.Split(rendered, "\n")
	if len(lines) != m.height {
		t.Fatalf("expected %d lines, got %d", m.height, len(lines))
	}
	if lines[len(lines)-2] != "status" {
		t.Fatalf("expected footer status on second-to-last line, got %q", lines[len(lines)-2])
	}
	if lines[len(lines)-1] != "q quit" {
		t.Fatalf("expected footer action on last line, got %q", lines[len(lines)-1])
	}
}

func TestSpaceMapsToPageDownWhenGlobalsAllowed(t *testing.T) {
	spy := &keySpyView{}
	m := Model{
		stack:     []viewstate.View{spy},
		crumbs:    []string{"workloads"},
		context:   "default",
		namespace: "default",
	}

	updated, _ := m.Update(bubbletea.KeyMsg{Type: bubbletea.KeySpace})
	got := updated.(Model)
	nextSpy := got.top().(*keySpyView)
	if nextSpy.lastKey != "pgdown" {
		t.Fatalf("expected space to map to pgdown, got %q", nextSpy.lastKey)
	}
}

func TestSpaceDoesNotMapWhenGlobalsSuppressed(t *testing.T) {
	spy := &keySpyView{suppress: true}
	m := Model{
		stack:     []viewstate.View{spy},
		crumbs:    []string{"workloads"},
		context:   "default",
		namespace: "default",
	}

	updated, _ := m.Update(bubbletea.KeyMsg{Type: bubbletea.KeySpace})
	got := updated.(Model)
	nextSpy := got.top().(*keySpyView)
	if nextSpy.lastKey != " " {
		t.Fatalf("expected raw space when globals suppressed, got %q", nextSpy.lastKey)
	}
}

func TestNKeyOpensNamespaceOverlay(t *testing.T) {
	m := New()

	updated, _ := m.Update(bubbletea.KeyMsg{Type: bubbletea.KeyRunes, Runes: []rune{'N'}})
	got := updated.(Model)

	if got.overlay == nil {
		t.Fatal("expected overlay to be non-nil after pressing N")
	}
}

func TestXKeyOpensContextOverlay(t *testing.T) {
	m := New()

	updated, _ := m.Update(bubbletea.KeyMsg{Type: bubbletea.KeyRunes, Runes: []rune{'X'}})
	got := updated.(Model)

	if got.overlay == nil {
		t.Fatal("expected overlay to be non-nil after pressing X")
	}
}

func TestSelectedNamespaceMsgUpdatesNamespaceAndReloads(t *testing.T) {
	m := New()

	updated, _ := m.Update(overlaypicker.SelectedMsg{Kind: "namespace", Value: "staging"})
	got := updated.(Model)

	if got.namespace != "staging" {
		t.Fatalf("expected namespace=staging, got %q", got.namespace)
	}
	if got.crumbs[0] != "workloads" {
		t.Fatalf("expected workloads crumb after namespace switch, got %q", got.crumbs[0])
	}
}

func TestInputRoutedToOverlayWhenActive(t *testing.T) {
	spy := &keySpyView{}
	m := Model{
		stack:   []viewstate.View{spy},
		crumbs:  []string{"workloads"},
		overlay: overlaypicker.New("namespace", []string{"default", "staging"}),
		context: "default",
		namespace: "default",
	}
	m.overlay.SetSize(120, 40)

	// Send a key — should be consumed by overlay, not reach spy.
	updated, _ := m.Update(bubbletea.KeyMsg{Type: bubbletea.KeyRunes, Runes: []rune{'a'}})
	got := updated.(Model)
	nextSpy := got.top().(*keySpyView)
	if nextSpy.lastKey != "" {
		t.Fatalf("expected spy not to receive key when overlay is active, but got %q", nextSpy.lastKey)
	}
}

func TestBackspaceWithSingleStackIsNoop(t *testing.T) {
	m := New()

	updated, _ := m.Update(bubbletea.KeyMsg{Type: bubbletea.KeyBackspace})
	got := updated.(Model)

	if len(got.stack) != 1 {
		t.Fatalf("expected stack len 1 after backspace at root, got %d", len(got.stack))
	}
}

func TestTabWithNoSideIsNoop(t *testing.T) {
	m := New()

	updated, _ := m.Update(bubbletea.KeyMsg{Type: bubbletea.KeyTab})
	got := updated.(Model)

	if got.sideActive {
		t.Fatal("expected sideActive=false when no side panel")
	}
}
