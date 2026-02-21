package app

import (
	"strings"
	"testing"

	bubbletea "github.com/charmbracelet/bubbletea"
	"github.com/dloss/podji/internal/ui/listview"
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

type scopePickerView struct {
	breadcrumb string
	selected   string
}

func (v scopePickerView) Init() bubbletea.Cmd { return nil }

func (v scopePickerView) Update(msg bubbletea.Msg) viewstate.Update {
	return viewstate.Update{Action: viewstate.Push, Next: v}
}

func (v scopePickerView) View() string { return "" }

func (v scopePickerView) Breadcrumb() string { return v.breadcrumb }

func (v scopePickerView) SelectedBreadcrumb() string { return v.selected }

func (v scopePickerView) Footer() string { return "" }

func (v scopePickerView) SetSize(width, height int) {}

func TestViewClampsBodyToWindowHeight(t *testing.T) {
	m := Model{
		stack:     []viewstate.View{overflowView{}},
		crumbs:    []string{"workloads"},
		lens:      0,
		scope:     scopeLens,
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
	if !strings.Contains(lines[1], "[Apps]") {
		t.Fatalf("expected breadcrumb line with lens tag, got %q", lines[1])
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

func TestLeftAtLensRootSwitchesToNamespace(t *testing.T) {
	m := New()

	updated, _ := m.Update(bubbletea.KeyMsg{Type: bubbletea.KeyLeft})
	got := updated.(Model)

	if got.scope != scopeNamespace {
		t.Fatalf("expected scope %d (namespace) after left at lens root, got %d", scopeNamespace, got.scope)
	}
	if got.crumbs[0] != "namespaces" {
		t.Fatalf("expected crumbs[0] = 'namespaces', got %q", got.crumbs[0])
	}
}

func TestLeftAtNamespaceSwitchesToContext(t *testing.T) {
	m := New()

	// First left → namespace
	updated, _ := m.Update(bubbletea.KeyMsg{Type: bubbletea.KeyLeft})
	// Second left → context
	updated, _ = updated.(Model).Update(bubbletea.KeyMsg{Type: bubbletea.KeyLeft})
	got := updated.(Model)

	if got.scope != scopeContext {
		t.Fatalf("expected scope %d (context) after second left, got %d", scopeContext, got.scope)
	}
	if got.crumbs[0] != "contexts" {
		t.Fatalf("expected crumbs[0] = 'contexts', got %q", got.crumbs[0])
	}
}

func TestUppercaseNSwitchesToNamespace(t *testing.T) {
	m := New()

	updated, _ := m.Update(bubbletea.KeyMsg{Type: bubbletea.KeyRunes, Runes: []rune{'N'}})
	got := updated.(Model)

	if got.scope != scopeNamespace {
		t.Fatalf("expected scope %d (namespace) after N, got %d", scopeNamespace, got.scope)
	}
	if got.crumbs[0] != "namespaces" {
		t.Fatalf("expected crumbs[0] = 'namespaces', got %q", got.crumbs[0])
	}
}

func TestLeftAtContextIsNoop(t *testing.T) {
	m := New()

	// Navigate to context scope
	updated, _ := m.Update(bubbletea.KeyMsg{Type: bubbletea.KeyLeft})
	updated, _ = updated.(Model).Update(bubbletea.KeyMsg{Type: bubbletea.KeyLeft})
	// Third left → should stay at context
	updated, _ = updated.(Model).Update(bubbletea.KeyMsg{Type: bubbletea.KeyLeft})
	got := updated.(Model)

	if got.scope != scopeContext {
		t.Fatalf("expected scope to remain %d (context), got %d", scopeContext, got.scope)
	}
}

func TestHistorySaveRestoreIncludesScope(t *testing.T) {
	m := New()

	// Navigate to namespace scope (left saves history with scopeLens)
	updated, _ := m.Update(bubbletea.KeyMsg{Type: bubbletea.KeyLeft})
	got := updated.(Model)

	if got.scope != scopeNamespace {
		t.Fatalf("expected namespace scope, got %d", got.scope)
	}
	if len(got.history) != 1 {
		t.Fatalf("expected 1 history entry, got %d", len(got.history))
	}
	if got.history[0].scope != scopeLens {
		t.Fatalf("expected history scope = %d (lens), got %d", scopeLens, got.history[0].scope)
	}
}

func TestNamespaceSelectionReturnsToActiveLensRoot(t *testing.T) {
	m := New()
	m.scope = scopeNamespace
	m.stack = []viewstate.View{
		scopePickerView{
			breadcrumb: "namespaces",
			selected:   "namespaces: team-a",
		},
	}
	m.crumbs = []string{"namespaces"}
	m.history = []snapshot{
		{
			stack:  []viewstate.View{overflowView{}},
			crumbs: []string{"pods: api-123"},
			lens:   1,
			scope:  scopeLens,
		},
	}

	updated, _ := m.Update(bubbletea.KeyMsg{Type: bubbletea.KeyEnter})
	got := updated.(Model)

	if got.namespace != "team-a" {
		t.Fatalf("expected namespace team-a, got %q", got.namespace)
	}
	if got.scope != scopeLens {
		t.Fatalf("expected scope %d (lens), got %d", scopeLens, got.scope)
	}
	if got.lens != 1 {
		t.Fatalf("expected lens 1, got %d", got.lens)
	}
	if len(got.stack) != 1 {
		t.Fatalf("expected lens root stack depth 1, got %d", len(got.stack))
	}
	res := got.registry.ResourceByKey(lenses[got.lens].landingKey)
	expected := normalizeBreadcrumbPart(listview.New(res, got.registry).Breadcrumb())
	if got.crumbs[0] != expected {
		t.Fatalf("expected root breadcrumb %q, got %q", expected, got.crumbs[0])
	}
}
