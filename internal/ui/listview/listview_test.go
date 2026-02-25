package listview

import (
	"strings"
	"testing"

	bubbletea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/dloss/podji/internal/resources"
	"github.com/dloss/podji/internal/ui/viewstate"
)

func TestWorkloadsFooterContainsSpecHints(t *testing.T) {
	registry := resources.DefaultRegistry()
	view := New(resources.NewWorkloads(), registry)

	footer := ansi.Strip(view.Footer())
	wants := []string{"s sort", "v state", "tab cols", "r related", "nav", "? help", "W", "P", "D", "S", "E"}
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
	if strings.Contains(rendered, "No items.") {
		t.Fatalf("expected no duplicate default empty state, got: %s", rendered)
	}
	if strings.Contains(rendered, "Hint: press r to view Related") {
		t.Fatalf("expected simplified empty-state message, got: %s", rendered)
	}
	if !strings.Contains(rendered, "  No pods found for workload `sync-reports`.") {
		t.Fatalf("expected indented empty-state message, got: %s", rendered)
	}
}

func TestFindModeJumpsToMatchingItem(t *testing.T) {
	registry := resources.DefaultRegistry()
	view := New(resources.NewWorkloads(), registry)
	view.SetSize(120, 40)

	// Press f to enter find mode.
	result := view.Update(keyRunes('f'))
	if result.Action != viewstate.None {
		t.Fatalf("expected no navigation on f, got %v", result.Action)
	}
	if !view.findMode {
		t.Fatal("expected findMode to be true after pressing f")
	}
	if !view.SuppressGlobalKeys() {
		t.Fatal("expected SuppressGlobalKeys to be true in find mode")
	}

	// Press a character to jump.
	view.Update(keyRunes('a'))
	if view.findMode {
		t.Fatal("expected findMode to be false after pressing a character")
	}

	// The selected item should start with 'a' (case-insensitive).
	if selected, ok := view.list.SelectedItem().(item); ok {
		name := strings.ToLower(strings.TrimSpace(selected.data.Name))
		if len(name) == 0 || name[0] != 'a' {
			t.Fatalf("expected item starting with 'a', got %q", selected.data.Name)
		}
	}
}

func TestFindModeEscCancels(t *testing.T) {
	registry := resources.DefaultRegistry()
	view := New(resources.NewWorkloads(), registry)
	view.SetSize(120, 40)

	view.Update(keyRunes('f'))
	if !view.findMode {
		t.Fatal("expected findMode to be true")
	}

	view.Update(keyEsc())
	if view.findMode {
		t.Fatal("expected findMode to be false after esc")
	}
}

func TestFindModeSuppressesGlobalKeys(t *testing.T) {
	registry := resources.DefaultRegistry()
	view := New(resources.NewWorkloads(), registry)
	view.SetSize(120, 40)

	if view.SuppressGlobalKeys() {
		t.Fatal("expected SuppressGlobalKeys to be false initially")
	}

	view.Update(keyRunes('f'))
	if !view.SuppressGlobalKeys() {
		t.Fatal("expected SuppressGlobalKeys to be true in find mode")
	}

	// Press a character to exit find mode.
	view.Update(keyRunes('z'))
	if view.SuppressGlobalKeys() {
		t.Fatal("expected SuppressGlobalKeys to be false after exiting find mode")
	}
}

func TestFindModeFooterIndicator(t *testing.T) {
	registry := resources.DefaultRegistry()
	view := New(resources.NewWorkloads(), registry)
	view.SetSize(120, 40)

	view.Update(keyRunes('f'))
	footer := ansi.Strip(view.Footer())
	if !strings.Contains(footer, "f") || !strings.Contains(footer, "…") {
		t.Fatalf("expected find mode indicator in footer, got: %s", footer)
	}
}

func TestComputeFindTargets(t *testing.T) {
	registry := resources.DefaultRegistry()
	view := New(resources.NewWorkloads(), registry)
	view.SetSize(120, 40)

	targets := view.computeFindTargets()
	if len(targets) == 0 {
		t.Fatal("expected at least one find target")
	}

	// Each target index should be the first occurrence of its starting letter.
	seen := make(map[rune]bool)
	visible := view.list.VisibleItems()
	for i, li := range visible {
		if it, ok := li.(item); ok {
			name := strings.TrimSpace(it.data.Name)
			if len(name) > 0 {
				ch := []rune(strings.ToLower(name))[0]
				if !seen[ch] {
					seen[ch] = true
					if !targets[i] {
						t.Fatalf("expected index %d to be a find target for char %c", i, ch)
					}
				}
			}
		}
	}
}

func TestColumnWidthsForRowsShrinkToContent(t *testing.T) {
	columns := []resources.TableColumn{
		{Name: "NAME", Width: 48},
		{Name: "STATUS", Width: 12},
		{Name: "AGE", Width: 6},
	}
	rows := [][]string{
		{"api", "Running", "1d"},
		{"web", "Pending", "2h"},
	}

	widths := columnWidthsForRows(columns, rows, 120, "name")
	if widths[0] != 4 || widths[1] != 7 || widths[2] != 3 {
		t.Fatalf("expected content-sized widths [4 7 3], got %v", widths)
	}
}

func TestColumnWidthsForRowsCanExceedPreferredWidthWhenRoomy(t *testing.T) {
	columns := []resources.TableColumn{
		{Name: "READY", Width: 7},
	}
	rows := [][]string{
		{"configmap"},
	}

	widths := columnWidthsForRows(columns, rows, 40, "name")
	if widths[0] != len("configmap") {
		t.Fatalf("expected width %d, got %v", len("configmap"), widths)
	}
}

func TestColumnWidthsForRowsPrioritizesFirstColumnWhenTight(t *testing.T) {
	columns := []resources.TableColumn{
		{Name: "NAME", Width: 32},
		{Name: "STATUS", Width: 18},
		{Name: "RESTARTS", Width: 14},
		{Name: "AGE", Width: 10},
	}
	rows := [][]string{
		{"very-long-workload-name", "CrashLoopBackOff", "1234", "90d"},
	}

	widths := columnWidthsForRows(columns, rows, 24, "name")
	sum := 0
	for _, width := range widths {
		sum += width
	}
	if got := sum + ((len(widths) - 1) * len(columnSeparator)); got > 24 {
		t.Fatalf("expected widths to fit 24 chars, got total %d (%v)", got, widths)
	}
	if widths[0] <= widths[1] {
		t.Fatalf("expected first column to keep priority over status, got %v", widths)
	}
}

func TestViewHeaderShowsSortArrowAndMovesWithSortMode(t *testing.T) {
	registry := resources.DefaultRegistry()
	view := New(resources.NewWorkloads(), registry)
	view.SetSize(120, 40)

	// Default mode is name.
	rendered := ansi.Strip(view.View())
	if strings.Contains(rendered, "▲") {
		t.Fatalf("expected no sort arrow on default mode, got: %s", rendered)
	}

	// name -> status
	view.Update(keyRunes('s'))
	rendered = ansi.Strip(view.View())
	if !strings.Contains(rendered, "▼STATUS") {
		t.Fatalf("expected status sort arrow, got: %s", rendered)
	}

	// status -> kind
	view.Update(keyRunes('s'))
	rendered = ansi.Strip(view.View())
	if !strings.Contains(rendered, "▲KIND") {
		t.Fatalf("expected kind sort arrow, got: %s", rendered)
	}

	// kind -> age
	view.Update(keyRunes('s'))
	rendered = ansi.Strip(view.View())
	if !strings.Contains(rendered, "▲AGE") {
		t.Fatalf("expected age sort arrow, got: %s", rendered)
	}

	// age -> name (default), arrow remains because user changed sort in this view
	view.Update(keyRunes('s'))
	rendered = ansi.Strip(view.View())
	if !strings.Contains(rendered, "▲WORKLOAD") {
		t.Fatalf("expected sort arrow to remain visible after returning to default mode, got: %s", rendered)
	}
}

func TestEventsStatusSortArrowUsesTypeColumn(t *testing.T) {
	registry := resources.DefaultRegistry()
	view := New(resources.NewEvents(), registry)
	view.SetSize(120, 40)

	// name -> status
	view.Update(keyRunes('s'))
	rendered := ansi.Strip(view.View())
	if !strings.Contains(rendered, "▼TYPE") {
		t.Fatalf("expected status sort arrow on TYPE for events, got: %s", rendered)
	}
}

func TestTabCyclesColumnOffset(t *testing.T) {
	registry := resources.DefaultRegistry()
	view := New(resources.NewWorkloads(), registry)
	view.SetSize(40, 20)

	if view.colOffset != 0 {
		t.Fatalf("expected initial colOffset=0, got %d", view.colOffset)
	}
	extra := len(view.columns) - 1
	k := max(1, view.visibleNonFirstCount())
	want := k % extra
	view.Update(keyTab())
	if view.colOffset != want {
		t.Fatalf("expected colOffset=%d after tab (page size %d), got %d", want, k, view.colOffset)
	}
}

func TestShiftTabCyclesColumnOffsetBackward(t *testing.T) {
	registry := resources.DefaultRegistry()
	view := New(resources.NewWorkloads(), registry)
	view.SetSize(40, 20)

	view.Update(keyTab())
	view.Update(keyShiftTab())
	if view.colOffset != 0 {
		t.Fatalf("expected colOffset=0 after tab then shift+tab, got %d", view.colOffset)
	}
}

func TestTabWrapsColumnOffset(t *testing.T) {
	registry := resources.DefaultRegistry()
	view := New(resources.NewWorkloads(), registry)
	view.SetSize(40, 20)

	extra := len(view.columns) - 1
	for i := 0; i < extra; i++ {
		view.Update(keyTab())
	}
	if view.colOffset != 0 {
		t.Fatalf("expected colOffset to wrap to 0 after %d tabs, got %d", extra, view.colOffset)
	}
}

func TestTabPagesForwardByVisibleColumnCount(t *testing.T) {
	registry := resources.DefaultRegistry()
	view := New(resources.NewWorkloads(), registry)
	view.SetSize(50, 20) // wide enough for >1 non-first column but not all
	view.refreshItems()  // ensure colWidths are computed

	k := view.visibleNonFirstCount()
	if k <= 1 {
		t.Skipf("screen not narrow enough to test multi-column paging (k=%d); increase width or change resource", k)
	}
	extra := len(view.columns) - 1
	want := k % extra
	view.Update(keyTab())
	if view.colOffset != want {
		t.Fatalf("expected paging by %d to colOffset=%d, got %d", k, want, view.colOffset)
	}
}

func TestTabSticksFirstColumn(t *testing.T) {
	registry := resources.DefaultRegistry()
	view := New(resources.NewWorkloads(), registry)
	view.SetSize(40, 20)

	firstBefore := view.visibleColumns()[0].Name
	view.Update(keyTab())
	firstAfter := view.visibleColumns()[0].Name
	if firstBefore != firstAfter {
		t.Fatalf("expected first column to remain %q after tab, got %q", firstBefore, firstAfter)
	}
}

func keyEsc() bubbletea.KeyMsg {
	return bubbletea.KeyMsg{Type: bubbletea.KeyEscape}
}

func keyTab() bubbletea.KeyMsg {
	return bubbletea.KeyMsg{Type: bubbletea.KeyTab}
}

func keyShiftTab() bubbletea.KeyMsg {
	return bubbletea.KeyMsg{Type: bubbletea.KeyShiftTab}
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
