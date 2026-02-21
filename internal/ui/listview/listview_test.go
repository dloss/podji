package listview

import (
	"strings"
	"testing"

	bubbletea "github.com/charmbracelet/bubbletea"
	"github.com/dloss/podji/internal/resources"
	"github.com/dloss/podji/internal/ui/viewstate"
)

func TestWorkloadsFooterContainsSpecHints(t *testing.T) {
	registry := resources.DefaultRegistry()
	view := New(resources.NewWorkloads(), registry)

	footer := view.Footer()
	wants := []string{"-> pods", "L logs", "R related", "y yaml", "tab view", "s sort"}
	for _, want := range wants {
		if !strings.Contains(footer, want) {
			t.Fatalf("footer missing %q: %s", want, footer)
		}
	}
}

func TestWorkloadsViewShowsForbiddenBanner(t *testing.T) {
	registry := resources.DefaultRegistry()
	w := resources.NewWorkloads()
	w.CycleScenario() // empty
	w.CycleScenario() // forbidden
	view := New(w, registry)

	rendered := view.View()
	if !strings.Contains(rendered, "Access denied") {
		t.Fatalf("expected forbidden banner, got: %s", rendered)
	}
}

func TestWorkloadsViewNoRelatedBanner(t *testing.T) {
	registry := resources.DefaultRegistry()
	view := New(resources.NewWorkloads(), registry)
	view.SetSize(120, 40)

	rendered := view.View()
	if strings.Contains(rendered, "Related:") {
		t.Fatalf("old Related: banner should be removed, got: %s", rendered)
	}
}

func TestPreferredLogPodSelectsProblemPodFirst(t *testing.T) {
	items := []resources.ResourceItem{
		{Name: "web-a", Status: "Running"},
		{Name: "web-b", Status: "CrashLoop"},
		{Name: "web-c", Status: "Running"},
	}

	selected := preferredLogPod(items)
	if selected.Name != "web-b" {
		t.Fatalf("expected crashloop pod, got %q", selected.Name)
	}
}

func TestPreferredLogPodFallsBackToFirst(t *testing.T) {
	items := []resources.ResourceItem{
		{Name: "web-a", Status: "Running"},
		{Name: "web-b", Status: "Running"},
	}

	selected := preferredLogPod(items)
	if selected.Name != "web-a" {
		t.Fatalf("expected first pod fallback, got %q", selected.Name)
	}
}

func TestFilterEnterAppliesFilterWithoutOpeningSelection(t *testing.T) {
	registry := resources.DefaultRegistry()
	view := New(resources.NewWorkloads(), registry)

	view.Update(keyRunes('/'))
	view.Update(keyRunes('a'))
	result := view.Update(keyEnter())

	if result.Action != viewstate.None {
		t.Fatalf("expected no navigation on enter while filtering, got %v", result.Action)
	}
	if !view.list.IsFiltered() {
		t.Fatalf("expected filter to be applied after enter")
	}
}

func TestFilterDownAppliesFilterWithoutOpeningSelection(t *testing.T) {
	registry := resources.DefaultRegistry()
	view := New(resources.NewWorkloads(), registry)

	view.Update(keyRunes('/'))
	view.Update(keyRunes('a'))
	result := view.Update(keyDown())

	if result.Action != viewstate.None {
		t.Fatalf("expected no navigation on down while filtering, got %v", result.Action)
	}
	if !view.list.IsFiltered() {
		t.Fatalf("expected filter to be applied after down")
	}
}

func TestItemFilterValueUsesNameOnly(t *testing.T) {
	it := item{
		data: resources.ResourceItem{
			Name:   "api",
			Status: "Degraded",
			Ready:  "2/3",
		},
	}

	if got := it.FilterValue(); got != "api" {
		t.Fatalf("expected name-only filter value, got %q", got)
	}
}

func TestEmptyStateMessageAlignedWithTable(t *testing.T) {
	registry := resources.DefaultRegistry()
	view := New(resources.NewWorkloadPods(resources.ResourceItem{Name: "sync-reports", Kind: "CJ"}), registry)

	rendered := view.View()
	if !strings.Contains(rendered, "  No pods found for workload `sync-reports`.") {
		t.Fatalf("expected indented empty-state message, got: %s", rendered)
	}
}

func keyRunes(r ...rune) bubbletea.KeyMsg {
	return bubbletea.KeyMsg{Type: bubbletea.KeyRunes, Runes: r}
}

func keyEnter() bubbletea.KeyMsg {
	return bubbletea.KeyMsg{Type: bubbletea.KeyEnter}
}

func keyDown() bubbletea.KeyMsg {
	return bubbletea.KeyMsg{Type: bubbletea.KeyDown}
}
