package resourcebrowser

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/dloss/podji/internal/resources"
)

func TestFooterShowsFilterAndFind(t *testing.T) {
	v := New(resources.DefaultRegistry(), resources.StubCRDs())
	v.SetSize(120, 40)

	footer := ansi.Strip(v.Footer())
	if !strings.Contains(footer, "/ filter") {
		t.Fatalf("expected / filter hint, got: %s", footer)
	}
	if !strings.Contains(footer, "f find") {
		t.Fatalf("expected f find hint, got: %s", footer)
	}
}
