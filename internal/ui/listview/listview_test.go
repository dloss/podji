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
	wants := []string{"/ filter", "s sort", "v state", "r related", "nav", "? help", "W", "P", "D", "S", "E"}
	for _, want := range wants {
		if !strings.Contains(footer, want) {
			t.Fatalf("footer missing %q: %s", want, footer)
		}
	}
}

func TestContainersFooterHidesSortHint(t *testing.T) {
	registry := resources.DefaultRegistry()
	pods := resources.NewPods()
	podItems := pods.Items()
	if len(podItems) == 0 {
		t.Fatal("expected stub pods")
	}
	view := New(resources.NewContainerResource(podItems[0], pods), registry)

	footer := ansi.Strip(view.Footer())
	if strings.Contains(footer, "s sort") {
		t.Fatalf("expected containers footer not to show sort hint, got: %s", footer)
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
	view := New(resources.NewWorkloadPods(resources.ResourceItem{Name: "sync-reports", Kind: "CJ"}, nil), registry)

	rendered := view.View()
	if strings.Contains(rendered, "No items.") {
		t.Fatalf("expected no duplicate default empty state, got: %s", rendered)
	}
	if strings.Contains(rendered, "Hint: press r to view Related") {
		t.Fatalf("expected simplified empty-state message, got: %s", rendered)
	}
	if !strings.Contains(rendered, "  No jobs have run for CronJob `sync-reports` yet.") {
		t.Fatalf("expected indented CronJob empty-state message, got: %s", rendered)
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

func TestExecMenuIgnoresUnrecognizedKey(t *testing.T) {
	registry := resources.DefaultRegistry()
	view := New(resources.NewWorkloads(), registry)
	view.SetSize(120, 40)

	view.Update(keyRunes('x'))
	if view.execState != execMenu {
		t.Fatalf("expected exec menu open, got %v", view.execState)
	}

	view.Update(keyRunes('z'))
	if view.execState != execMenu {
		t.Fatalf("expected exec menu to remain open on unknown key, got %v", view.execState)
	}
}

func TestExecMenuKeepsStateForUnsupportedAction(t *testing.T) {
	registry := resources.DefaultRegistry()
	pods := resources.NewPods()
	podItems := pods.Items()
	if len(podItems) == 0 {
		t.Fatal("expected stub pods")
	}
	view := New(resources.NewContainerResource(podItems[0], pods), registry)
	view.SetSize(120, 40)

	view.Update(keyRunes('x'))
	if view.execState != execMenu {
		t.Fatalf("expected exec menu open, got %v", view.execState)
	}

	// scale is unsupported for container resources; menu should remain open.
	view.Update(keyRunes('s'))
	if view.execState != execMenu {
		t.Fatalf("expected exec menu to remain open when action unsupported, got %v", view.execState)
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

	// Default mode is name — no sort arrow.
	rendered := ansi.Strip(view.View())
	if strings.Contains(rendered, "▲") || strings.Contains(rendered, "▼") {
		t.Fatalf("expected no sort arrow on default mode, got: %s", rendered)
	}

	// s enters sort pick mode; 's' selects status (problem-first, desc=false → ▲STATUS).
	view.Update(keyRunes('s'))
	view.Update(keyRunes('s'))
	rendered = ansi.Strip(view.View())
	if !strings.Contains(rendered, "▲STATUS") {
		t.Fatalf("expected status sort arrow, got: %s", rendered)
	}

	// s then 'k' selects kind ascending → ▲KIND.
	view.Update(keyRunes('s'))
	view.Update(keyRunes('k'))
	rendered = ansi.Strip(view.View())
	if !strings.Contains(rendered, "▲KIND") {
		t.Fatalf("expected kind sort arrow, got: %s", rendered)
	}

	// s then 'a' selects age ascending (newest-first) → ▲AGE.
	view.Update(keyRunes('s'))
	view.Update(keyRunes('a'))
	rendered = ansi.Strip(view.View())
	if !strings.Contains(rendered, "▲AGE") {
		t.Fatalf("expected age sort arrow, got: %s", rendered)
	}

	// s then 'n' returns to name sort — back to default, no arrow shown.
	view.Update(keyRunes('s'))
	view.Update(keyRunes('n'))
	rendered = ansi.Strip(view.View())
	if strings.Contains(rendered, "▲") || strings.Contains(rendered, "▼") {
		t.Fatalf("expected no sort arrow after returning to default name sort, got: %s", rendered)
	}

	// s then 'N' sorts name descending → ▼WORKLOAD.
	view.Update(keyRunes('s'))
	view.Update(keyRunes('N'))
	rendered = ansi.Strip(view.View())
	if !strings.Contains(rendered, "▼WORKLOAD") {
		t.Fatalf("expected descending name sort arrow, got: %s", rendered)
	}
}

func TestEventsStatusSortArrowUsesTypeColumn(t *testing.T) {
	registry := resources.DefaultRegistry()
	view := New(resources.NewEvents(), registry)
	view.SetSize(120, 40)

	// s enters sort pick mode; 's' selects status sort (▲TYPE for events).
	view.Update(keyRunes('s'))
	view.Update(keyRunes('s'))
	rendered := ansi.Strip(view.View())
	if !strings.Contains(rendered, "▲TYPE") {
		t.Fatalf("expected status sort arrow on TYPE for events, got: %s", rendered)
	}
}

func keyEsc() bubbletea.KeyMsg {
	return bubbletea.KeyMsg{Type: bubbletea.KeyEscape}
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
