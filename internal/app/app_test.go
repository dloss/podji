package app

import (
	"strings"
	"testing"

	bubbletea "github.com/charmbracelet/bubbletea"
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

func (*keySpyView) View() string      { return "" }
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
		scope:     scopeResources,
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
		scope:     scopeResources,
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
		scope:     scopeResources,
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
		scope:     scopeResources,
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

	// Navigate to namespace scope (left saves history with scopeResources)
	updated, _ := m.Update(bubbletea.KeyMsg{Type: bubbletea.KeyLeft})
	got := updated.(Model)

	if got.scope != scopeNamespace {
		t.Fatalf("expected namespace scope, got %d", got.scope)
	}
	if len(got.history) != 1 {
		t.Fatalf("expected 1 history entry, got %d", len(got.history))
	}
	if got.history[0].scope != scopeResources {
		t.Fatalf("expected history scope = %d (resources), got %d", scopeResources, got.history[0].scope)
	}
}

func TestNamespaceScopeYOpensYamlWithoutSelectingNamespace(t *testing.T) {
	m := New()

	updated, _ := m.Update(bubbletea.KeyMsg{Type: bubbletea.KeyLeft})
	updated, _ = updated.(Model).Update(bubbletea.KeyMsg{Type: bubbletea.KeyRunes, Runes: []rune{'y'}})
	got := updated.(Model)

	if got.scope != scopeNamespace {
		t.Fatalf("expected to remain in namespace scope, got %d", got.scope)
	}
	if len(got.stack) != 2 {
		t.Fatalf("expected yaml view to be pushed, stack len=%d", len(got.stack))
	}
	if got.top().Breadcrumb() != "yaml" {
		t.Fatalf("expected top breadcrumb yaml, got %q", got.top().Breadcrumb())
	}
}

func TestContextScopeROpensRelatedWithoutSelectingContext(t *testing.T) {
	m := New()

	updated, _ := m.Update(bubbletea.KeyMsg{Type: bubbletea.KeyLeft})
	updated, _ = updated.(Model).Update(bubbletea.KeyMsg{Type: bubbletea.KeyLeft})
	updated, _ = updated.(Model).Update(bubbletea.KeyMsg{Type: bubbletea.KeyRunes, Runes: []rune{'r'}})
	got := updated.(Model)

	if got.scope != scopeContext {
		t.Fatalf("expected to remain in context scope, got %d", got.scope)
	}
	if len(got.stack) != 2 {
		t.Fatalf("expected related view to be pushed, stack len=%d", len(got.stack))
	}
	if got.top().Breadcrumb() != "related" {
		t.Fatalf("expected top breadcrumb related, got %q", got.top().Breadcrumb())
	}
}
