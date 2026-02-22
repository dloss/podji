package relatedview

import (
	"strings"
	"testing"

	bubbletea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/dloss/podji/internal/resources"
)

func TestRelatedFindModeJumpsToMatchingCategory(t *testing.T) {
	registry := resources.DefaultRegistry()
	workloads := resources.NewWorkloads()
	source := workloads.Items()[0]
	view := New(source, workloads, registry)
	view.SetSize(120, 40)

	view.Update(keyRunes('f'))
	if !view.findMode {
		t.Fatal("expected findMode to be true after pressing f")
	}
	if !view.SuppressGlobalKeys() {
		t.Fatal("expected SuppressGlobalKeys to be true in find mode")
	}

	view.Update(keyRunes('p'))
	if view.findMode {
		t.Fatal("expected findMode to be false after entering a character")
	}

	selected, ok := view.list.SelectedItem().(relatedItem)
	if !ok {
		t.Fatal("expected a related item selection")
	}
	name := strings.ToLower(strings.TrimSpace(selected.entry.name))
	if len(name) == 0 || name[0] != 'p' {
		t.Fatalf("expected category starting with 'p', got %q", selected.entry.name)
	}
}

func TestRelatedFindModeFooterIndicator(t *testing.T) {
	registry := resources.DefaultRegistry()
	workloads := resources.NewWorkloads()
	source := workloads.Items()[0]
	view := New(source, workloads, registry)
	view.SetSize(120, 40)

	view.Update(keyRunes('f'))
	footer := ansi.Strip(view.Footer())
	if !strings.Contains(footer, "f") || !strings.Contains(footer, "…") {
		t.Fatalf("expected find mode indicator in footer, got: %s", footer)
	}
}

func TestRelationListFindModeJumpsToMatchingItem(t *testing.T) {
	registry := resources.DefaultRegistry()
	related := newRelationList(resources.NewRelatedConfig("api"), registry)
	related.SetSize(120, 40)

	related.Update(keyRunes('f'))
	if !related.findMode {
		t.Fatal("expected findMode to be true after pressing f")
	}
	if !related.SuppressGlobalKeys() {
		t.Fatal("expected SuppressGlobalKeys to be true in find mode")
	}

	related.Update(keyRunes('a'))
	if related.findMode {
		t.Fatal("expected findMode to be false after entering a character")
	}

	selected, ok := related.list.SelectedItem().(relationItem)
	if !ok {
		t.Fatal("expected a relation item selection")
	}
	name := strings.ToLower(strings.TrimSpace(selected.data.Name))
	if len(name) == 0 || name[0] != 'a' {
		t.Fatalf("expected item starting with 'a', got %q", selected.data.Name)
	}
}

func TestRelationColumnWidthsForRowsFitsAvailableWidth(t *testing.T) {
	columns := []resources.TableColumn{
		{Name: "RELATED", Width: 18},
		{Name: "COUNT", Width: 5},
		{Name: "DESCRIPTION", Width: 58},
	}
	rows := [][]string{
		{"pods", "12", "Owned pods and replica set relations"},
	}

	widths := relationColumnWidthsForRows(columns, rows, 22)
	sum := 0
	for _, width := range widths {
		sum += width
	}
	if got := sum + ((len(widths) - 1) * len(relationColumnSeparator)); got > 22 {
		t.Fatalf("expected widths to fit 22 chars, got total %d (%v)", got, widths)
	}
	if widths[0] < 6 {
		t.Fatalf("expected first column to retain minimum readability, got %v", widths)
	}
}

func keyRunes(r ...rune) bubbletea.KeyMsg {
	return bubbletea.KeyMsg{Type: bubbletea.KeyRunes, Runes: r}
}
