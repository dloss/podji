package relatedview

import (
	"strings"
	"testing"

	bubbletea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/dloss/podji/internal/resources"
	"github.com/dloss/podji/internal/ui/viewstate"
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

func TestRelatedFooterShowsPanelIndicatorOnlyWhenFocused(t *testing.T) {
	registry := resources.DefaultRegistry()
	workloads := resources.NewWorkloads()
	source := workloads.Items()[0]
	view := New(source, workloads, registry)
	view.SetSize(120, 40)

	focusedFooter := ansi.Strip(view.Footer())
	if !strings.Contains(focusedFooter, "Panel: Related") {
		t.Fatalf("expected focused footer to show panel indicator, got: %s", focusedFooter)
	}

	view.SetFocused(false)
	unfocusedFooter := ansi.Strip(view.Footer())
	if strings.Contains(unfocusedFooter, "Panel: Related") {
		t.Fatalf("expected unfocused footer to hide panel indicator, got: %s", unfocusedFooter)
	}
}

func TestRelationListFooterShowsPanelIndicatorOnlyWhenFocused(t *testing.T) {
	registry := resources.DefaultRegistry()
	related := newRelationList(resources.NewRelatedConfig("api"), registry)
	related.SetSize(120, 40)

	focusedFooter := ansi.Strip(related.Footer())
	if !strings.Contains(focusedFooter, "Panel: Related") {
		t.Fatalf("expected focused footer to show panel indicator, got: %s", focusedFooter)
	}

	related.SetFocused(false)
	unfocusedFooter := ansi.Strip(related.Footer())
	if strings.Contains(unfocusedFooter, "Panel: Related") {
		t.Fatalf("expected unfocused footer to hide panel indicator, got: %s", unfocusedFooter)
	}
}

func TestRelatedSelectionMarkerHiddenWhenUnfocused(t *testing.T) {
	registry := resources.DefaultRegistry()
	workloads := resources.NewWorkloads()
	source := workloads.Items()[0]
	view := New(source, workloads, registry)
	view.SetSize(120, 40)

	focusedView := ansi.Strip(view.View())
	if !strings.Contains(focusedView, "▌") {
		t.Fatalf("expected focused related view to show selection marker, got: %s", focusedView)
	}

	view.SetFocused(false)
	unfocusedView := ansi.Strip(view.View())
	if strings.Contains(unfocusedView, "▌") {
		t.Fatalf("expected unfocused related view to hide selection marker, got: %s", unfocusedView)
	}
}

func TestRelationListSelectionMarkerHiddenWhenUnfocused(t *testing.T) {
	registry := resources.DefaultRegistry()
	related := newRelationList(resources.NewRelatedConfig("api"), registry)
	related.SetSize(120, 40)

	focusedView := ansi.Strip(related.View())
	if !strings.Contains(focusedView, "▌") {
		t.Fatalf("expected focused relation list to show selection marker, got: %s", focusedView)
	}

	related.SetFocused(false)
	unfocusedView := ansi.Strip(related.View())
	if strings.Contains(unfocusedView, "▌") {
		t.Fatalf("expected unfocused relation list to hide selection marker, got: %s", unfocusedView)
	}
}

func TestPodRelatedEventsOpenEventsResourceList(t *testing.T) {
	registry := resources.DefaultRegistry()
	pods := resources.NewPods()
	source := pods.Items()[0]
	view := New(source, pods, registry)
	view.SetSize(120, 40)

	update := view.Update(bubbletea.KeyMsg{Type: bubbletea.KeyEnter})
	if update.Action != viewstate.Push {
		t.Fatalf("expected push action, got %v", update.Action)
	}
	if update.Next == nil {
		t.Fatal("expected next view to be set")
	}
	if got := update.Next.Breadcrumb(); got != "events" {
		t.Fatalf("expected events breadcrumb, got %q", got)
	}

	eventsList, ok := update.Next.(*relationList)
	if !ok {
		t.Fatalf("expected relationList, got %T", update.Next)
	}
	items := eventsList.resource.Items()
	if len(items) != 3 {
		t.Fatalf("expected 3 scoped events, got %d", len(items))
	}
	for _, item := range items {
		if !strings.HasPrefix(item.Name, source.Name+".") {
			t.Fatalf("expected scoped event name prefix %q, got %q", source.Name+".", item.Name)
		}
	}
}

func TestRelationListSortToggleWorksForScopedEvents(t *testing.T) {
	registry := resources.DefaultRegistry()
	related := newRelationList(resources.NewScopedEvents("api", 3), registry)
	related.SetSize(120, 40)

	sortable, ok := related.resource.(resources.ToggleSortable)
	if !ok {
		t.Fatalf("expected ToggleSortable resource, got %T", related.resource)
	}
	if got := sortable.SortMode(); got != "name" {
		t.Fatalf("expected initial sort mode name, got %q", got)
	}

	related.Update(bubbletea.KeyMsg{Type: bubbletea.KeyRunes, Runes: []rune{'s'}})
	if got := sortable.SortMode(); got != "status" {
		t.Fatalf("expected sort mode status after one toggle, got %q", got)
	}
}

func TestEmptyViewShowsConsistentBodyAndFooter(t *testing.T) {
	view := NewEmpty()
	view.SetSize(120, 40)

	body := ansi.Strip(view.View())
	if !strings.Contains(body, "Related to: none") {
		t.Fatalf("expected empty view to include Related to: none title, got: %s", body)
	}
	if !strings.Contains(body, "No selection in main panel.") {
		t.Fatalf("expected empty view guidance, got: %s", body)
	}

	footer := ansi.Strip(view.Footer())
	if !strings.Contains(footer, "Panel: Related") {
		t.Fatalf("expected panel indicator in empty view footer, got: %s", footer)
	}
	if !strings.Contains(footer, "tab main") {
		t.Fatalf("expected tab main action in empty view footer, got: %s", footer)
	}
	if !strings.Contains(footer, "esc close") {
		t.Fatalf("expected esc close action in empty view footer, got: %s", footer)
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

	widths := relationColumnWidthsForRows(columns, rows, 22, "related")
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

func TestRelationColumnWidthsCanExceedPreferredWidthWhenRoomy(t *testing.T) {
	columns := []resources.TableColumn{
		{Name: "READY", Width: 7},
	}
	rows := [][]string{
		{"configmap"},
	}

	widths := relationColumnWidthsForRows(columns, rows, 40, "related")
	if widths[0] != len("configmap") {
		t.Fatalf("expected width %d, got %v", len("configmap"), widths)
	}
}

func keyRunes(r ...rune) bubbletea.KeyMsg {
	return bubbletea.KeyMsg{Type: bubbletea.KeyRunes, Runes: r}
}
