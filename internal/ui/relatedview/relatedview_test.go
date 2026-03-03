package relatedview

import (
	"strings"
	"testing"

	bubbletea "github.com/charmbracelet/bubbletea"
	"github.com/dloss/podji/internal/data"
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

func TestRelationListShowsUnsupportedActionFeedback(t *testing.T) {
	registry := resources.DefaultRegistry()
	related := newRelationList(resources.NewScopedEvents("api", 3), registry)
	related.SetSize(120, 40)

	related.Update(keyRunes('x'))
	footer := related.Footer()
	if !strings.Contains(footer, "x unavailable in this view") {
		t.Fatalf("expected unsupported-action feedback in footer, got %q", footer)
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
	picker := NewPickerForSelection(parent, nil, nil, data.Scope{})
	if len(picker.entries) == 0 {
		t.Fatal("expected at least one related entry for workloads")
	}
}

func TestPickerUsesRelationIndexCountsWhenAvailable(t *testing.T) {
	workloads := resources.NewWorkloads()
	parent := &fakeParent{
		item:     workloads.Items()[0],
		resource: workloads,
	}
	rel := fakeRelationIndex{
		related: map[string][]resources.ResourceItem{
			"pods":     {{Name: "a"}, {Name: "b"}, {Name: "c"}},
			"services": {},
		},
	}
	picker := NewPickerForSelection(parent, nil, rel, data.Scope{Context: "default", Namespace: "default"})

	if got := entryCountByName(picker.entries, "pods"); got != 3 {
		t.Fatalf("expected pods count 3 from relation index, got %d", got)
	}
	if got := entryCountByName(picker.entries, "services"); got != 0 {
		t.Fatalf("expected services count 0 from relation index, got %d", got)
	}
}

func TestPickerOpensIndexedRelationItemsWhenAvailable(t *testing.T) {
	workloads := resources.NewWorkloads()
	parent := &fakeParent{
		item:     workloads.Items()[0],
		resource: workloads,
	}
	rel := fakeRelationIndex{
		related: map[string][]resources.ResourceItem{
			"pods": {{Name: "from-index-a"}, {Name: "from-index-b"}},
		},
	}
	picker := NewPickerForSelection(parent, resources.DefaultRegistry(), rel, data.Scope{Context: "default", Namespace: "default"})
	picker.SetSize(120, 40)
	picker.Update(keyRunes('j')) // move to "pods"

	msg := picker.Update(bubbletea.KeyMsg{Type: bubbletea.KeyEnter}).Cmd().(SelectedMsg)
	next := msg.Open()
	related, ok := next.(*relationList)
	if !ok {
		t.Fatalf("expected relation list, got %T", next)
	}
	if got := len(related.list.Items()); got == 0 {
		t.Fatalf("expected indexed relation items, got %d", got)
	}
}

func TestPickerForSelectionReturnsEmptyPickerWhenNoSelection(t *testing.T) {
	picker := NewPickerForSelection(struct{}{}, nil, nil, data.Scope{})
	if len(picker.entries) != 0 {
		t.Fatalf("expected empty picker for non-provider, got %d entries", len(picker.entries))
	}
}

func TestPickerEscEmitsPop(t *testing.T) {
	workloads := resources.NewWorkloads()
	parent := &fakeParent{item: workloads.Items()[0], resource: workloads}
	picker := NewPickerForSelection(parent, nil, nil, data.Scope{})
	picker.SetSize(120, 40)

	update := picker.Update(bubbletea.KeyMsg{Type: bubbletea.KeyEsc})
	if update.Action != viewstate.Pop {
		t.Fatalf("expected Pop action on Esc, got %v", update.Action)
	}
}

func TestPickerEnterEmitsSelectedMsg(t *testing.T) {
	workloads := resources.NewWorkloads()
	parent := &fakeParent{item: workloads.Items()[0], resource: workloads}
	picker := NewPickerForSelection(parent, nil, nil, data.Scope{})
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
	picker := NewPickerForSelection(parent, nil, nil, data.Scope{})
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
	picker := NewPickerForSelection(parent, nil, nil, data.Scope{})
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

func TestPickerNavigationSkipsZeroCountRows(t *testing.T) {
	picker := &Picker{
		entries: []entry{
			{name: "events", count: 3, open: func() viewstate.View { return nil }},
			{name: "services", count: 0, open: func() viewstate.View { return nil }},
			{name: "pods", count: 2, open: func() viewstate.View { return nil }},
		},
	}

	picker.Update(keyRunes('j'))
	if picker.cursor != 2 {
		t.Fatalf("expected cursor to skip zero-count row and land on 2, got %d", picker.cursor)
	}
}

func TestPickerEnterOnZeroCountDoesNothing(t *testing.T) {
	picker := &Picker{
		entries: []entry{
			{name: "services", count: 0, open: func() viewstate.View { return nil }},
		},
	}

	update := picker.Update(bubbletea.KeyMsg{Type: bubbletea.KeyEnter})
	if update.Action != viewstate.None {
		t.Fatalf("expected no action on Enter for zero-count row, got %v", update.Action)
	}
	if update.Cmd != nil {
		t.Fatal("expected no Cmd on Enter for zero-count row")
	}
}

func TestPickerViewRendersEntries(t *testing.T) {
	workloads := resources.NewWorkloads()
	parent := &fakeParent{item: workloads.Items()[0], resource: workloads}
	picker := NewPickerForSelection(parent, nil, nil, data.Scope{})
	picker.SetSize(120, 40)

	view := picker.View()
	if !strings.Contains(view, "events") {
		t.Fatalf("expected picker view to contain 'events', got: %s", view)
	}
	if !strings.Contains(view, "pods") {
		t.Fatalf("expected picker view to contain 'pods', got: %s", view)
	}
}

func TestPickerViewRendersZeroCountAsZero(t *testing.T) {
	workloads := resources.NewWorkloads()
	parent := &fakeParent{
		item:     workloads.Items()[0],
		resource: workloads,
	}
	rel := fakeRelationIndex{
		related: map[string][]resources.ResourceItem{
			"pods":     {{Name: "a"}},
			"services": {},
		},
	}
	picker := NewPickerForSelection(parent, nil, rel, data.Scope{Context: "default", Namespace: "default"})
	picker.SetSize(120, 40)

	view := picker.View()
	if !strings.Contains(view, "(0)") {
		t.Fatalf("expected picker view to contain explicit zero count '(0)', got: %s", view)
	}
}

func TestPickerViewRendersRelatedToTitle(t *testing.T) {
	workloads := resources.NewWorkloads()
	parent := &fakeParent{item: workloads.Items()[0], resource: workloads}
	picker := NewPickerForSelection(parent, nil, nil, data.Scope{})
	picker.SetSize(120, 40)

	view := picker.View()
	if !strings.Contains(view, "Related to: ") {
		t.Fatalf("expected picker view title prefix 'Related to: ', got: %s", view)
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

func entryCountByName(entries []entry, name string) int {
	for _, e := range entries {
		if e.name == name {
			return e.count
		}
	}
	return -1
}

type fakeRelationIndex struct {
	related map[string][]resources.ResourceItem
}

func (f fakeRelationIndex) Related(data.Scope, string, resources.ResourceItem) map[string][]resources.ResourceItem {
	return f.related
}
