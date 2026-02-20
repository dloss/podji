package listview

import (
	"strings"
	"testing"

	"github.com/dloss/podji/internal/resources"
)

func TestWorkloadsFooterContainsSpecHints(t *testing.T) {
	registry := resources.DefaultRegistry()
	view := New(resources.NewWorkloads(), registry)

	footer := view.Footer()
	wants := []string{"-> pods", "L logs", "r related", "tab view", "s sort"}
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
