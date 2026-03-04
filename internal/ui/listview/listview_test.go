package listview

import (
	"errors"
	"reflect"
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
	wants := []string{"/ search", "& filter", "s sort", "r related", "X context", "N namespace", "nav", "? help", "W", "P", "D", "S", "E"}
	for _, want := range wants {
		if !strings.Contains(footer, want) {
			t.Fatalf("footer missing %q: %s", want, footer)
		}
	}
}

func TestContainersFooterShowsSortHint(t *testing.T) {
	registry := resources.DefaultRegistry()
	pods := resources.NewPods()
	podItems := pods.Items()
	if len(podItems) == 0 {
		t.Fatal("expected stub pods")
	}
	view := New(resources.NewContainerResource(podItems[0], pods), registry)

	footer := ansi.Strip(view.Footer())
	if !strings.Contains(footer, "s sort") {
		t.Fatalf("expected containers footer to show sort hint, got: %s", footer)
	}
}

func TestContainersSortPickSupportsCharAndCount(t *testing.T) {
	registry := resources.DefaultRegistry()
	pods := resources.NewPods()
	podItems := pods.Items()
	if len(podItems) == 0 {
		t.Fatal("expected stub pods")
	}
	view := New(resources.NewContainerResource(podItems[0], pods), registry)
	view.SetSize(120, 40)

	view.Update(keyRunes('s'))
	view.Update(keyRunes('s'))
	if view.sortMode != "status" {
		t.Fatalf("expected status mode from char key, got %q", view.sortMode)
	}

	view.Update(keyRunes('s'))
	view.Update(keyRunes('3'))
	if view.sortMode != "ready" {
		t.Fatalf("expected 3rd column mode from count key, got %q", view.sortMode)
	}
}

func TestSortPickerHidesDuplicateLeadKeys(t *testing.T) {
	registry := resources.DefaultRegistry()
	registry.SetNamespace(resources.AllNamespaces)

	pods := resources.NewPods()
	pods.SetNamespace(resources.AllNamespaces)
	items := pods.Items()
	if len(items) == 0 {
		t.Fatal("expected stub pods")
	}
	query := resources.NewQueryResource("pods(query)", items, pods)
	view := New(query, registry)
	view.SetSize(140, 40)

	view.Update(keyRunes('s'))
	footer := ansi.Strip(view.Footer())
	if got := strings.Count(footer, "n/N"); got != 1 {
		t.Fatalf("expected one n/N binding in sort picker labels for duplicated key, got %d: %s", got, footer)
	}
}

func TestWorkloadsViewShowsForbiddenBanner(t *testing.T) {
	t.Setenv("PODJI_SCENARIO", "forbidden")
	registry := resources.DefaultRegistry()
	w := resources.NewWorkloads()
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

	view.Update(keyRunes('&'))
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

	view.Update(keyRunes('&'))
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

func TestListViewShowsUnsupportedActionFeedbackWhenNoSelection(t *testing.T) {
	t.Setenv("PODJI_MOCK_SCENARIO", "empty")
	registry := resources.DefaultRegistry()
	view := New(resources.NewWorkloads(), registry)
	view.SetSize(120, 40)

	view.Update(keyRunes('x'))
	footer := ansi.Strip(view.Footer())
	if !strings.Contains(footer, "x unavailable: no selected item") {
		t.Fatalf("expected unsupported-action feedback in footer, got %q", footer)
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

func TestPortForwardArgsForPodUsesResourceNamespace(t *testing.T) {
	registry := resources.DefaultRegistry()
	view := New(resources.NewPods(), registry)

	args, ok := view.portForwardArgs(item{data: resources.ResourceItem{Name: "web-123"}}, "8080:80")
	if !ok {
		t.Fatal("expected args to be generated")
	}
	want := []string{"-n", resources.DefaultNamespace, "port-forward", "pod/web-123", "8080:80"}
	if !reflect.DeepEqual(args, want) {
		t.Fatalf("unexpected args: got %v want %v", args, want)
	}
}

func TestPortForwardArgsForServiceUsesItemNamespace(t *testing.T) {
	registry := resources.DefaultRegistry()
	view := New(resources.NewServices(), registry)

	args, ok := view.portForwardArgs(item{data: resources.ResourceItem{Name: "api", Namespace: "kube-system"}}, "8443:443")
	if !ok {
		t.Fatal("expected args to be generated")
	}
	want := []string{"-n", "kube-system", "port-forward", "service/api", "8443:443"}
	if !reflect.DeepEqual(args, want) {
		t.Fatalf("unexpected args: got %v want %v", args, want)
	}
}

func TestPortForwardArgsRejectsEmptyPorts(t *testing.T) {
	registry := resources.DefaultRegistry()
	view := New(resources.NewPods(), registry)

	if _, ok := view.portForwardArgs(item{data: resources.ResourceItem{Name: "web-123"}}, " "); ok {
		t.Fatal("expected empty ports to be rejected")
	}
}

func TestFilterModeFooterIndicator(t *testing.T) {
	registry := resources.DefaultRegistry()
	view := New(resources.NewWorkloads(), registry)
	view.SetSize(120, 40)

	view.Update(keyRunes('&'))
	// Add some text to trigger cursor display
	view.Update(keyRunes('t', 'e', 's', 't'))
	footer := ansi.Strip(view.Footer())
	if !strings.Contains(footer, "&") {
		t.Fatalf("expected filter input in footer, got: %s", footer)
	}
	if !strings.Contains(footer, "test") {
		t.Fatalf("expected filter text in footer, got: %s", footer)
	}
	if !strings.Contains(footer, "esc") {
		t.Fatalf("expected esc cancel hint in filter footer, got: %s", footer)
	}
}

func TestSearchModeFooterIndicator(t *testing.T) {
	registry := resources.DefaultRegistry()
	view := New(resources.NewWorkloads(), registry)
	view.SetSize(120, 40)

	view.Update(keyRunes('/'))
	view.Update(keyRunes('a', 'p', 'i'))
	footer := ansi.Strip(view.Footer())
	if !strings.Contains(footer, "search") {
		t.Fatalf("expected search indicator in footer, got: %s", footer)
	}
	if !strings.Contains(footer, "/ api") {
		t.Fatalf("expected search text in footer, got: %s", footer)
	}
	if !strings.Contains(footer, "esc") {
		t.Fatalf("expected esc cancel hint in search footer, got: %s", footer)
	}
}

func TestSearchEnterFindsMatchesWithoutFiltering(t *testing.T) {
	registry := resources.DefaultRegistry()
	view := New(resources.NewWorkloads(), registry)
	view.SetSize(120, 40)

	view.Update(keyRunes('/'))
	view.Update(keyRunes('a', 'p', 'i'))
	view.Update(keyEnter())

	if view.list.IsFiltered() {
		t.Fatal("expected search to avoid list filtering")
	}
	if len(view.matchRows) == 0 {
		t.Fatal("expected search to produce matches")
	}
}

func TestSearchBackKeyMovesToPreviousMatch(t *testing.T) {
	registry := resources.DefaultRegistry()
	view := New(resources.NewWorkloads(), registry)
	view.SetSize(120, 40)

	view.Update(keyRunes('/'))
	view.Update(keyRunes('e'))
	view.Update(keyEnter())
	if len(view.matchRows) < 2 {
		t.Fatalf("expected at least 2 matches for test, got %d", len(view.matchRows))
	}

	first := view.matchRows[0]
	view.Update(keyRunes('n'))
	second := view.matchRows[view.matchIndex]
	if second == first {
		t.Fatal("expected n to advance to a different match")
	}

	view.Update(keyRunes('b'))
	got := view.matchRows[view.matchIndex]
	if got != first {
		t.Fatalf("expected b to return to first match index %d, got %d", first, got)
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

func TestColumnWidthsForRowsUsesNamespaceHeaderWhenNamespaceIsFirstColumn(t *testing.T) {
	columns := []resources.TableColumn{
		{Name: "NAMESPACE", Width: 16},
		{Name: "NAME", Width: 48},
	}
	rows := [][]string{
		{"dev", "web"},
	}

	widths := columnWidthsForRows(columns, rows, 120, "WORKLOAD")
	if widths[0] != len("NAMESPACE") {
		t.Fatalf("expected namespace width %d, got %v", len("NAMESPACE"), widths)
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

	// Default mode is name ascending — arrow always shown on WORKLOAD column.
	rendered := ansi.Strip(view.View())
	if !strings.Contains(rendered, "↑WORKLOAD") {
		t.Fatalf("expected ↑WORKLOAD on default sort, got: %s", rendered)
	}

	// s enters sort pick mode; 's' selects status (problem-first, desc=false → ↑STATUS).
	view.Update(keyRunes('s'))
	view.Update(keyRunes('s'))
	rendered = ansi.Strip(view.View())
	if !strings.Contains(rendered, "↑STATUS") {
		t.Fatalf("expected status sort arrow, got: %s", rendered)
	}

	// s then 'k' selects kind ascending → ↑KIND.
	view.Update(keyRunes('s'))
	view.Update(keyRunes('k'))
	rendered = ansi.Strip(view.View())
	if !strings.Contains(rendered, "↑KIND") {
		t.Fatalf("expected kind sort arrow, got: %s", rendered)
	}

	// s then 'a' selects age ascending (newest-first) → ↑AGE.
	view.Update(keyRunes('s'))
	view.Update(keyRunes('a'))
	rendered = ansi.Strip(view.View())
	if !strings.Contains(rendered, "↑AGE") {
		t.Fatalf("expected age sort arrow, got: %s", rendered)
	}

	// s then 'w' returns to name sort — arrow back on ↑WORKLOAD.
	view.Update(keyRunes('s'))
	view.Update(keyRunes('w'))
	rendered = ansi.Strip(view.View())
	if !strings.Contains(rendered, "↑WORKLOAD") {
		t.Fatalf("expected ↑WORKLOAD after returning to default name sort, got: %s", rendered)
	}

	// s then 'W' sorts name descending → ↓WORKLOAD.
	view.Update(keyRunes('s'))
	view.Update(keyRunes('W'))
	rendered = ansi.Strip(view.View())
	if !strings.Contains(rendered, "↓WORKLOAD") {
		t.Fatalf("expected descending name sort arrow, got: %s", rendered)
	}
}

func TestAllNamespacesDefaultNameSortArrowOnNameColumn(t *testing.T) {
	registry := resources.DefaultRegistry()
	registry.SetNamespace(resources.AllNamespaces)
	workloads := resources.NewWorkloads()
	workloads.SetNamespace(resources.AllNamespaces)
	view := New(workloads, registry)
	view.SetSize(120, 40)

	rendered := ansi.Strip(view.View())
	if strings.Contains(rendered, "↑NAMESPACE") {
		t.Fatalf("expected default name sort arrow not to be on namespace column, got: %s", rendered)
	}
	if !strings.Contains(rendered, "↑NAME") {
		t.Fatalf("expected default name sort arrow on name column, got: %s", rendered)
	}
}

func TestSortByColumnNumber(t *testing.T) {
	registry := resources.DefaultRegistry()
	view := New(resources.NewWorkloads(), registry)
	view.SetSize(120, 40)

	// '2' sorts by column 2 (KIND for workloads) ascending → ↑KIND.
	view.Update(keyRunes('s'))
	view.Update(keyRunes('2'))
	rendered := ansi.Strip(view.View())
	if !strings.Contains(rendered, "↑KIND") {
		t.Fatalf("expected kind sort arrow from numeric key, got: %s", rendered)
	}

	// Pressing '2' again toggles column 2 descending → ↓KIND.
	view.Update(keyRunes('s'))
	view.Update(keyRunes('2'))
	rendered = ansi.Strip(view.View())
	if !strings.Contains(rendered, "↓KIND") {
		t.Fatalf("expected descending kind sort arrow from repeated numeric key, got: %s", rendered)
	}

	// '1' sorts by column 1 (WORKLOAD/name) ascending — arrow on ↑WORKLOAD.
	view.Update(keyRunes('s'))
	view.Update(keyRunes('1'))
	rendered = ansi.Strip(view.View())
	if !strings.Contains(rendered, "↑WORKLOAD") {
		t.Fatalf("expected ↑WORKLOAD for default sort via numeric key, got: %s", rendered)
	}
}

func TestSortByTenthColumnNumber(t *testing.T) {
	registry := resources.DefaultRegistry()
	view := New(resources.NewWorkloads(), registry)
	view.SetSize(160, 40)

	// Enable wide mode so workloads expose 10 columns.
	view.Update(keyRunes('w'))

	// '0' sorts by column 10 (SERVICEACCOUNT) ascending.
	view.Update(keyRunes('s'))
	view.Update(keyRunes('0'))
	rendered := ansi.Strip(view.View())
	if !strings.Contains(rendered, "↑SERVICEACCOUNT") {
		t.Fatalf("expected service-account sort arrow from numeric key 0, got: %s", rendered)
	}

	// Pressing '0' again toggles column 10 descending.
	view.Update(keyRunes('s'))
	view.Update(keyRunes('0'))
	rendered = ansi.Strip(view.View())
	if !strings.Contains(rendered, "↓SERVICEACCOUNT") {
		t.Fatalf("expected descending service-account sort arrow from repeated numeric key, got: %s", rendered)
	}
}

func TestEventsStatusSortArrowUsesTypeColumn(t *testing.T) {
	registry := resources.DefaultRegistry()
	view := New(resources.NewEvents(), registry)
	view.SetSize(120, 40)

	// s enters sort pick mode; 't' selects status sort (first char of "TYPE") → ↑TYPE for events.
	view.Update(keyRunes('s'))
	view.Update(keyRunes('t'))
	rendered := ansi.Strip(view.View())
	if !strings.Contains(rendered, "↑TYPE") {
		t.Fatalf("expected status sort arrow on TYPE for events, got: %s", rendered)
	}
}

type fakeWorkloadsLiveResource struct {
	items   []resources.ResourceItem
	pods    []resources.ResourceItem
	podsErr error
}

func (f fakeWorkloadsLiveResource) Name() string { return "workloads" }
func (f fakeWorkloadsLiveResource) Key() rune    { return 'W' }
func (f fakeWorkloadsLiveResource) Items() []resources.ResourceItem {
	return f.items
}
func (f fakeWorkloadsLiveResource) Sort([]resources.ResourceItem) {}
func (f fakeWorkloadsLiveResource) Detail(resources.ResourceItem) resources.DetailData {
	return resources.DetailData{}
}
func (f fakeWorkloadsLiveResource) Logs(resources.ResourceItem) []string   { return nil }
func (f fakeWorkloadsLiveResource) Events(resources.ResourceItem) []string { return nil }
func (f fakeWorkloadsLiveResource) YAML(resources.ResourceItem) string     { return "" }
func (f fakeWorkloadsLiveResource) Describe(resources.ResourceItem) string { return "" }
func (f fakeWorkloadsLiveResource) ListResource(name string) ([]resources.ResourceItem, error) {
	if name != "pods" {
		return nil, errors.New("unsupported")
	}
	if f.podsErr != nil {
		return nil, f.podsErr
	}
	return f.pods, nil
}

type fakeLiveListResource struct {
	name    string
	items   []resources.ResourceItem
	lists   map[string][]resources.ResourceItem
	listErr map[string]error
}

func (f fakeLiveListResource) Name() string { return f.name }
func (f fakeLiveListResource) Key() rune    { return 0 }
func (f fakeLiveListResource) Items() []resources.ResourceItem {
	return f.items
}
func (f fakeLiveListResource) Sort([]resources.ResourceItem) {}
func (f fakeLiveListResource) Detail(resources.ResourceItem) resources.DetailData {
	return resources.DetailData{}
}
func (f fakeLiveListResource) Logs(resources.ResourceItem) []string   { return nil }
func (f fakeLiveListResource) Events(resources.ResourceItem) []string { return nil }
func (f fakeLiveListResource) YAML(resources.ResourceItem) string     { return "" }
func (f fakeLiveListResource) Describe(resources.ResourceItem) string { return "" }
func (f fakeLiveListResource) ListResource(name string) ([]resources.ResourceItem, error) {
	if err := f.listErr[name]; err != nil {
		return nil, err
	}
	return f.lists[name], nil
}

func TestWorkloadForwardUsesLivePodsForDirectNavigation(t *testing.T) {
	workload := resources.ResourceItem{
		UID:       "uid-1",
		Name:      "coredns",
		Namespace: "kube-system",
		Kind:      "DEP",
		Selector:  map[string]string{"k8s-app": "kube-dns"},
	}
	resource := fakeWorkloadsLiveResource{
		items: []resources.ResourceItem{workload},
		pods: []resources.ResourceItem{
			{
				Name:      "coredns-7d9c7c9d4f-qwz8p",
				Namespace: "kube-system",
				Status:    "Running",
				Ready:     "2/2",
				Labels:    map[string]string{"k8s-app": "kube-dns"},
			},
		},
	}
	view := New(resource, resources.DefaultRegistry())

	action, next := view.ForwardViewForCommand(workload, "")
	if action != viewstate.Push {
		t.Fatalf("expected push into pods view, got %v", action)
	}
	nextList, ok := next.(*View)
	if !ok {
		t.Fatalf("expected list view, got %T", next)
	}
	if !strings.HasPrefix(strings.ToLower(nextList.resource.Name()), "pods") {
		t.Fatalf("expected pods resource, got %q", nextList.resource.Name())
	}
	items := nextList.resource.Items()
	if len(items) != 1 || items[0].Namespace != "kube-system" {
		t.Fatalf("expected live pod with namespace, got %#v", items)
	}
}

func TestDeploymentForwardUsesLivePodsForDirectNavigation(t *testing.T) {
	deployment := resources.ResourceItem{
		UID:      "dep-uid-1",
		Name:     "api",
		Kind:     "DEP",
		Selector: map[string]string{"app": "api"},
	}
	view := New(fakeLiveListResource{
		name:  "deployments",
		items: []resources.ResourceItem{deployment},
		lists: map[string][]resources.ResourceItem{
			"pods": {
				{Name: "api-1", Namespace: "default", Labels: map[string]string{"app": "api"}},
			},
		},
	}, resources.DefaultRegistry())

	action, next := view.ForwardViewForCommand(deployment, "")
	if action != viewstate.Push {
		t.Fatalf("expected push into pods view, got %v", action)
	}
	nextList := next.(*View)
	if !strings.HasPrefix(strings.ToLower(nextList.resource.Name()), "pods") {
		t.Fatalf("expected pods resource, got %q", nextList.resource.Name())
	}
	if got := nextList.resource.Items(); len(got) != 1 || got[0].Name != "api-1" {
		t.Fatalf("expected live deployment pod, got %#v", got)
	}
}

func TestServiceForwardUsesLiveBackends(t *testing.T) {
	service := resources.ResourceItem{
		Name:     "kube-dns",
		Selector: map[string]string{"k8s-app": "kube-dns"},
	}
	view := New(fakeLiveListResource{
		name:  "services",
		items: []resources.ResourceItem{service},
		lists: map[string][]resources.ResourceItem{
			"pods": {
				{Name: "coredns-a", Namespace: "kube-system", Labels: map[string]string{"k8s-app": "kube-dns"}},
			},
		},
	}, resources.DefaultRegistry())
	action, next := view.ForwardViewForCommand(service, "")
	if action != viewstate.Push {
		t.Fatalf("expected push into backends view, got %v", action)
	}
	nextList := next.(*View)
	if !strings.HasPrefix(strings.ToLower(nextList.resource.Name()), "backends") {
		t.Fatalf("expected backends resource, got %q", nextList.resource.Name())
	}
	if got := nextList.resource.Items(); len(got) != 1 || got[0].Name != "coredns-a" {
		t.Fatalf("expected live backend pod, got %#v", got)
	}
}

func TestIngressForwardUsesLiveServices(t *testing.T) {
	ing := resources.ResourceItem{
		Name:  "web",
		Extra: map[string]string{"services": "api-svc,web-svc"},
	}
	view := New(fakeLiveListResource{
		name:  "ingresses",
		items: []resources.ResourceItem{ing},
		lists: map[string][]resources.ResourceItem{
			"services": {
				{Name: "api-svc", Namespace: "default"},
				{Name: "other-svc", Namespace: "default"},
			},
		},
	}, resources.DefaultRegistry())
	action, next := view.ForwardViewForCommand(ing, "")
	if action != viewstate.Push {
		t.Fatalf("expected push into services view, got %v", action)
	}
	nextList := next.(*View)
	if !strings.HasPrefix(strings.ToLower(nextList.resource.Name()), "services") {
		t.Fatalf("expected services resource, got %q", nextList.resource.Name())
	}
	if got := nextList.resource.Items(); len(got) != 1 || got[0].Name != "api-svc" {
		t.Fatalf("expected filtered live services, got %#v", got)
	}
}

func TestNodeForwardUsesLivePods(t *testing.T) {
	node := resources.ResourceItem{Name: "worker-01"}
	view := New(fakeLiveListResource{
		name:  "nodes",
		items: []resources.ResourceItem{node},
		lists: map[string][]resources.ResourceItem{
			"pods": {
				{Name: "api-1", Namespace: "default", Extra: map[string]string{"node": "worker-01"}},
				{Name: "api-2", Namespace: "default", Extra: map[string]string{"node": "worker-02"}},
			},
		},
	}, resources.DefaultRegistry())
	action, next := view.ForwardViewForCommand(node, "")
	if action != viewstate.Push {
		t.Fatalf("expected push into node pods view, got %v", action)
	}
	nextList := next.(*View)
	if !strings.HasPrefix(strings.ToLower(nextList.resource.Name()), "pods") {
		t.Fatalf("expected pods resource, got %q", nextList.resource.Name())
	}
	if got := nextList.resource.Items(); len(got) != 1 || got[0].Name != "api-1" {
		t.Fatalf("expected node-matched pod, got %#v", got)
	}
}

func TestPVCForwardUsesLiveMountedByPods(t *testing.T) {
	pvc := resources.ResourceItem{Name: "data-pvc"}
	view := New(fakeLiveListResource{
		name:  "persistentvolumeclaims",
		items: []resources.ResourceItem{pvc},
		lists: map[string][]resources.ResourceItem{
			"pods": {
				{Name: "db-0", Namespace: "default", Extra: map[string]string{"pvc-refs": "data-pvc,cache-pvc"}},
				{Name: "api-1", Namespace: "default", Extra: map[string]string{"pvc-refs": "other-pvc"}},
			},
		},
	}, resources.DefaultRegistry())
	action, next := view.ForwardViewForCommand(pvc, "")
	if action != viewstate.Push {
		t.Fatalf("expected push into mounted-by view, got %v", action)
	}
	nextList := next.(*View)
	if !strings.HasPrefix(strings.ToLower(nextList.resource.Name()), "mounted-by") {
		t.Fatalf("expected mounted-by resource, got %q", nextList.resource.Name())
	}
	if got := nextList.resource.Items(); len(got) != 1 || got[0].Name != "db-0" {
		t.Fatalf("expected pvc-mounted pod, got %#v", got)
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
