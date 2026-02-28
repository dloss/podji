package relatedview

import (
	"strings"
	"testing"

	bubbletea "github.com/charmbracelet/bubbletea"
	"github.com/dloss/podji/internal/resources"
	"github.com/dloss/podji/internal/ui/viewstate"
)

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

func TestRelationListSortToggleWorksForScopedEvents(t *testing.T) {
	registry := resources.DefaultRegistry()
	related := newRelationList(resources.NewScopedEvents("api", 3), registry)
	related.SetSize(120, 40)

	sortable, ok := related.resource.(resources.Sortable)
	if !ok {
		t.Fatalf("expected Sortable resource, got %T", related.resource)
	}
	if got := sortable.SortMode(); got != "name" {
		t.Fatalf("expected initial sort mode name, got %q", got)
	}

	related.Update(bubbletea.KeyMsg{Type: bubbletea.KeyRunes, Runes: []rune{'s'}})
	if got := sortable.SortMode(); got != "status" {
		t.Fatalf("expected sort mode status after one toggle, got %q", got)
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

func TestPickerForSelectionReturnsEntriesForWorkloads(t *testing.T) {
	workloads := resources.NewWorkloads()
	// Build a minimal parent that implements selectionProvider + resourceProvider.
	parent := &fakeParent{
		item:     workloads.Items()[0],
		resource: workloads,
	}
	picker := NewPickerForSelection(parent, nil)
	if len(picker.entries) == 0 {
		t.Fatal("expected at least one related entry for workloads")
	}
}

func TestPickerForSelectionReturnsEmptyPickerWhenNoSelection(t *testing.T) {
	picker := NewPickerForSelection(struct{}{}, nil)
	if len(picker.entries) != 0 {
		t.Fatalf("expected empty picker for non-provider, got %d entries", len(picker.entries))
	}
}

func TestPickerEscEmitsPop(t *testing.T) {
	workloads := resources.NewWorkloads()
	parent := &fakeParent{item: workloads.Items()[0], resource: workloads}
	picker := NewPickerForSelection(parent, nil)
	picker.SetSize(120, 40)

	update := picker.Update(bubbletea.KeyMsg{Type: bubbletea.KeyEsc})
	if update.Action != viewstate.Pop {
		t.Fatalf("expected Pop action on Esc, got %v", update.Action)
	}
}

func TestPickerEnterEmitsSelectedMsg(t *testing.T) {
	workloads := resources.NewWorkloads()
	parent := &fakeParent{item: workloads.Items()[0], resource: workloads}
	picker := NewPickerForSelection(parent, nil)
	picker.SetSize(120, 40)

	update := picker.Update(bubbletea.KeyMsg{Type: bubbletea.KeyEnter})
	if update.Action != viewstate.Pop {
		t.Fatalf("expected Pop action on Enter, got %v", update.Action)
	}
	if update.Cmd == nil {
		t.Fatal("expected a Cmd on Enter, got nil")
	}
	msg := update.Cmd()
	if _, ok := msg.(SelectedMsg); !ok {
		t.Fatalf("expected SelectedMsg, got %T", msg)
	}
}

func TestPickerSelectedMsgOpenReturnsView(t *testing.T) {
	workloads := resources.NewWorkloads()
	parent := &fakeParent{item: workloads.Items()[0], resource: workloads}
	picker := NewPickerForSelection(parent, nil)
	picker.SetSize(120, 40)

	update := picker.Update(bubbletea.KeyMsg{Type: bubbletea.KeyEnter})
	msg := update.Cmd().(SelectedMsg)
	next := msg.Open()
	if next == nil {
		t.Fatal("expected Open() to return a non-nil view")
	}
}

func TestPickerNavigationMovescursor(t *testing.T) {
	workloads := resources.NewWorkloads()
	parent := &fakeParent{item: workloads.Items()[0], resource: workloads}
	picker := NewPickerForSelection(parent, nil)
	picker.SetSize(120, 40)

	if picker.cursor != 0 {
		t.Fatalf("expected initial cursor at 0, got %d", picker.cursor)
	}
	picker.Update(keyRunes('j'))
	if picker.cursor != 1 {
		t.Fatalf("expected cursor at 1 after j, got %d", picker.cursor)
	}
	picker.Update(keyRunes('k'))
	if picker.cursor != 0 {
		t.Fatalf("expected cursor back at 0 after k, got %d", picker.cursor)
	}
}

func TestPickerViewRendersEntries(t *testing.T) {
	workloads := resources.NewWorkloads()
	parent := &fakeParent{item: workloads.Items()[0], resource: workloads}
	picker := NewPickerForSelection(parent, nil)
	picker.SetSize(120, 40)

	view := picker.View()
	if !strings.Contains(view, "events") {
		t.Fatalf("expected picker view to contain 'events', got: %s", view)
	}
	if !strings.Contains(view, "pods") {
		t.Fatalf("expected picker view to contain 'pods', got: %s", view)
	}
}

// fakeParent implements selectionProvider and resourceProvider for tests.
type fakeParent struct {
	item     resources.ResourceItem
	resource resources.ResourceType
}

func (f *fakeParent) SelectedItem() resources.ResourceItem { return f.item }
func (f *fakeParent) Resource() resources.ResourceType     { return f.resource }

func keyRunes(r ...rune) bubbletea.KeyMsg {
	return bubbletea.KeyMsg{Type: bubbletea.KeyRunes, Runes: r}
}
