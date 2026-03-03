package app

import (
	"strings"
	"testing"

	bubbletea "github.com/charmbracelet/bubbletea"
	"github.com/dloss/podji/internal/ui/commandbar"
	"github.com/dloss/podji/internal/ui/describeview"
	"github.com/dloss/podji/internal/ui/detailview"
	"github.com/dloss/podji/internal/ui/listview"
	"github.com/dloss/podji/internal/ui/overlaypicker"
	"github.com/dloss/podji/internal/ui/viewstate"
)

type overflowView struct{}
type shortView struct{}
type keySpyView struct {
	lastKey  string
	suppress bool
}
type disposableSpyView struct {
	disposed bool
}
type pushInitMsg struct{}
type initPushView struct{}
type initTargetView struct{}

func (overflowView) Init() bubbletea.Cmd { return nil }

func (overflowView) Update(msg bubbletea.Msg) viewstate.Update {
	return viewstate.Update{Action: viewstate.None, Next: overflowView{}}
}

func (overflowView) View() string {
	return strings.Repeat("row\n", 20)
}

func (overflowView) Breadcrumb() string { return "workloads" }

func (overflowView) Footer() string { return "q quit" }

func (overflowView) SetSize(width, height int) {}

func (shortView) Init() bubbletea.Cmd { return nil }

func (shortView) Update(msg bubbletea.Msg) viewstate.Update {
	return viewstate.Update{Action: viewstate.None, Next: shortView{}}
}

func (shortView) View() string { return "line 1\nline 2" }

func (shortView) Breadcrumb() string { return "workloads" }

func (shortView) Footer() string { return "status\nq quit" }

func (shortView) SetSize(width, height int) {}

func (v *keySpyView) Init() bubbletea.Cmd { return nil }

func (v *keySpyView) Update(msg bubbletea.Msg) viewstate.Update {
	if key, ok := msg.(bubbletea.KeyMsg); ok {
		v.lastKey = key.String()
	}
	return viewstate.Update{Action: viewstate.None, Next: v}
}

func (*keySpyView) View() string              { return "" }
func (*keySpyView) Breadcrumb() string        { return "workloads" }
func (*keySpyView) Footer() string            { return "status\nq quit" }
func (*keySpyView) SetSize(width, height int) {}
func (v *keySpyView) SuppressGlobalKeys() bool {
	return v.suppress
}

func (v *disposableSpyView) Init() bubbletea.Cmd { return nil }
func (v *disposableSpyView) Update(msg bubbletea.Msg) viewstate.Update {
	return viewstate.Update{Action: viewstate.None, Next: v}
}
func (v *disposableSpyView) View() string              { return "" }
func (v *disposableSpyView) Breadcrumb() string        { return "logs" }
func (v *disposableSpyView) Footer() string            { return "" }
func (v *disposableSpyView) SetSize(width, height int) {}
func (v *disposableSpyView) Dispose()                  { v.disposed = true }

func (initPushView) Init() bubbletea.Cmd { return nil }
func (initPushView) Update(msg bubbletea.Msg) viewstate.Update {
	if key, ok := msg.(bubbletea.KeyMsg); ok && key.String() == "enter" {
		return viewstate.Update{Action: viewstate.Push, Next: initTargetView{}}
	}
	return viewstate.Update{Action: viewstate.None, Next: initPushView{}}
}
func (initPushView) View() string              { return "" }
func (initPushView) Breadcrumb() string        { return "workloads" }
func (initPushView) Footer() string            { return "" }
func (initPushView) SetSize(width, height int) {}

func (initTargetView) Init() bubbletea.Cmd {
	return func() bubbletea.Msg { return pushInitMsg{} }
}
func (initTargetView) Update(msg bubbletea.Msg) viewstate.Update {
	return viewstate.Update{Action: viewstate.None, Next: initTargetView{}}
}
func (initTargetView) View() string              { return "" }
func (initTargetView) Breadcrumb() string        { return "events" }
func (initTargetView) Footer() string            { return "" }
func (initTargetView) SetSize(width, height int) {}

func TestViewClampsBodyToWindowHeight(t *testing.T) {
	m := Model{
		stack:     []viewstate.View{overflowView{}},
		crumbs:    []string{"workloads"},
		context:   "default",
		namespace: "default",
		height:    6,
	}

	rendered := m.View()
	lines := strings.Split(rendered, "\n")
	if len(lines) > m.height {
		t.Fatalf("expected <= %d lines, got %d", m.height, len(lines))
	}
	if !strings.Contains(lines[0], "Context:") || !strings.Contains(lines[0], "Namespace:") {
		t.Fatalf("expected scope line with context and namespace, got %q", lines[0])
	}
	if !strings.Contains(lines[0], "Mode:") {
		t.Fatalf("expected scope line with mode label, got %q", lines[0])
	}
	if !strings.Contains(lines[1], "Workload") {
		t.Fatalf("expected breadcrumb line with root resource tag, got %q", lines[1])
	}
}

func TestViewPadsBodyToKeepFooterAtBottom(t *testing.T) {
	m := Model{
		stack:     []viewstate.View{shortView{}},
		crumbs:    []string{"workloads"},
		context:   "default",
		namespace: "default",
		height:    8,
	}

	rendered := m.View()
	lines := strings.Split(rendered, "\n")
	if len(lines) != m.height {
		t.Fatalf("expected %d lines, got %d", m.height, len(lines))
	}
	if lines[len(lines)-2] != "status" {
		t.Fatalf("expected footer status on second-to-last line, got %q", lines[len(lines)-2])
	}
	if lines[len(lines)-1] != "q quit" {
		t.Fatalf("expected footer action on last line, got %q", lines[len(lines)-1])
	}
}

func TestSpaceMapsToPageDownWhenGlobalsAllowed(t *testing.T) {
	spy := &keySpyView{}
	m := Model{
		stack:     []viewstate.View{spy},
		crumbs:    []string{"workloads"},
		context:   "default",
		namespace: "default",
	}

	updated, _ := m.Update(bubbletea.KeyMsg{Type: bubbletea.KeySpace})
	got := updated.(Model)
	nextSpy := got.top().(*keySpyView)
	if nextSpy.lastKey != "pgdown" {
		t.Fatalf("expected space to map to pgdown, got %q", nextSpy.lastKey)
	}
}

func TestSpaceDoesNotMapWhenGlobalsSuppressed(t *testing.T) {
	spy := &keySpyView{suppress: true}
	m := Model{
		stack:     []viewstate.View{spy},
		crumbs:    []string{"workloads"},
		context:   "default",
		namespace: "default",
	}

	updated, _ := m.Update(bubbletea.KeyMsg{Type: bubbletea.KeySpace})
	got := updated.(Model)
	nextSpy := got.top().(*keySpyView)
	if nextSpy.lastKey != " " {
		t.Fatalf("expected raw space when globals suppressed, got %q", nextSpy.lastKey)
	}
}

func TestNKeyOpensNamespaceOverlay(t *testing.T) {
	m := New()

	updated, _ := m.Update(bubbletea.KeyMsg{Type: bubbletea.KeyRunes, Runes: []rune{'N'}})
	got := updated.(Model)

	if got.overlay == nil {
		t.Fatal("expected overlay to be non-nil after pressing N")
	}
}

func TestXKeyOpensContextOverlay(t *testing.T) {
	m := New()

	updated, _ := m.Update(bubbletea.KeyMsg{Type: bubbletea.KeyRunes, Runes: []rune{'X'}})
	got := updated.(Model)

	if got.overlay == nil {
		t.Fatal("expected overlay to be non-nil after pressing X")
	}
}

func TestSelectedNamespaceMsgUpdatesNamespaceAndReloads(t *testing.T) {
	m := New()

	updated, _ := m.Update(overlaypicker.SelectedMsg{Kind: "namespace", Value: "staging"})
	got := updated.(Model)

	if got.namespace != "staging" {
		t.Fatalf("expected namespace=staging, got %q", got.namespace)
	}
	if got.crumbs[0] != "workloads" {
		t.Fatalf("expected workloads crumb after namespace switch, got %q", got.crumbs[0])
	}
}

func TestInputRoutedToOverlayWhenActive(t *testing.T) {
	spy := &keySpyView{}
	m := Model{
		stack:     []viewstate.View{spy},
		crumbs:    []string{"workloads"},
		overlay:   overlaypicker.New("namespace", []string{"default", "staging"}),
		context:   "default",
		namespace: "default",
	}
	m.overlay.SetSize(120, 40)

	// Send a key — should be consumed by overlay, not reach spy.
	updated, _ := m.Update(bubbletea.KeyMsg{Type: bubbletea.KeyRunes, Runes: []rune{'a'}})
	got := updated.(Model)
	nextSpy := got.top().(*keySpyView)
	if nextSpy.lastKey != "" {
		t.Fatalf("expected spy not to receive key when overlay is active, but got %q", nextSpy.lastKey)
	}
}

func TestBackspaceWithSingleStackIsNoop(t *testing.T) {
	m := New()

	updated, _ := m.Update(bubbletea.KeyMsg{Type: bubbletea.KeyBackspace})
	got := updated.(Model)

	if len(got.stack) != 1 {
		t.Fatalf("expected stack len 1 after backspace at root, got %d", len(got.stack))
	}
}

func TestBackNavigationDisposesPoppedView(t *testing.T) {
	disposable := &disposableSpyView{}
	m := New()
	m.stack = append(m.stack, disposable)
	m.crumbs = append(m.crumbs, "logs")

	updated, _ := m.Update(bubbletea.KeyMsg{Type: bubbletea.KeyBackspace})
	got := updated.(Model)
	if len(got.stack) != 1 {
		t.Fatalf("expected stack len 1 after backspace pop, got %d", len(got.stack))
	}
	if !disposable.disposed {
		t.Fatal("expected popped disposable view to be disposed")
	}
}

func TestPushBatchesNextInitCommand(t *testing.T) {
	m := Model{
		stack:     []viewstate.View{initPushView{}},
		crumbs:    []string{"workloads"},
		context:   "default",
		namespace: "default",
		width:     120,
		height:    40,
	}
	updated, cmd := m.Update(bubbletea.KeyMsg{Type: bubbletea.KeyEnter})
	got := updated.(Model)
	if len(got.stack) != 2 {
		t.Fatalf("expected pushed stack depth 2, got %d", len(got.stack))
	}
	if cmd == nil {
		t.Fatal("expected command containing pushed view init cmd")
	}
	msg := cmd()
	if _, ok := msg.(pushInitMsg); !ok {
		t.Fatalf("expected pushInitMsg from batched init cmd, got %T", msg)
	}
}

func TestScopeSwitchDisposesPreviousStackViews(t *testing.T) {
	disposable := &disposableSpyView{}
	m := New()
	m.stack = append(m.stack, disposable)
	m.crumbs = append(m.crumbs, "logs")

	updated, _ := m.Update(overlaypicker.SelectedMsg{Kind: "namespace", Value: "staging"})
	got := updated.(Model)
	if len(got.stack) != 1 {
		t.Fatalf("expected scope switch to reset stack to one view, got %d", len(got.stack))
	}
	if !disposable.disposed {
		t.Fatal("expected previous stack views to be disposed on scope switch")
	}
}

func TestRapidScopeSwitchesDisposeStackViews(t *testing.T) {
	m := New()
	disposables := make([]*disposableSpyView, 0, 20)

	for i := 0; i < 20; i++ {
		disposable := &disposableSpyView{}
		disposables = append(disposables, disposable)
		m.stack = append(m.stack, disposable)
		m.crumbs = append(m.crumbs, "logs")

		msg := overlaypicker.SelectedMsg{Kind: "namespace", Value: "staging"}
		if i%2 == 1 {
			msg = overlaypicker.SelectedMsg{Kind: "context", Value: "prod"}
		}

		updated, _ := m.Update(msg)
		m = updated.(Model)
		if len(m.stack) != 1 {
			t.Fatalf("expected single root view after switch %d, got %d", i, len(m.stack))
		}
	}

	for i, disposable := range disposables {
		if !disposable.disposed {
			t.Fatalf("expected disposable %d to be disposed", i)
		}
	}
}

func TestResourceHotkeyDisposesPreviousStackViews(t *testing.T) {
	disposable := &disposableSpyView{}
	m := New()
	m.stack = append(m.stack, disposable)
	m.crumbs = append(m.crumbs, "logs")

	updated, _ := m.Update(bubbletea.KeyMsg{Type: bubbletea.KeyRunes, Runes: []rune{'P'}})
	got := updated.(Model)
	if len(got.stack) != 1 {
		t.Fatalf("expected resource hotkey to reset stack, got %d", len(got.stack))
	}
	if !disposable.disposed {
		t.Fatal("expected previous stack views to be disposed on resource switch")
	}
}

func TestRKeyOpensRelatedPickerOverlay(t *testing.T) {
	m := New()

	updated, _ := m.Update(bubbletea.KeyMsg{Type: bubbletea.KeyRunes, Runes: []rune{'r'}})
	got := updated.(Model)

	if got.relatedPicker == nil {
		t.Fatal("expected relatedPicker to be non-nil after pressing r")
	}
	if got.overlay != nil {
		t.Fatal("expected overlay to remain nil after pressing r")
	}
}

func TestRelatedPickerEscClosesOverlay(t *testing.T) {
	m := New()

	opened, _ := m.Update(bubbletea.KeyMsg{Type: bubbletea.KeyRunes, Runes: []rune{'r'}})
	withPicker := opened.(Model)
	if withPicker.relatedPicker == nil {
		t.Fatal("expected relatedPicker to be open after r")
	}

	updated, _ := withPicker.Update(bubbletea.KeyMsg{Type: bubbletea.KeyEsc})
	got := updated.(Model)
	if got.relatedPicker != nil {
		t.Fatal("expected relatedPicker to be nil after Esc")
	}
}

func TestScopeSwitchPreservesCurrentListViewResource(t *testing.T) {
	m := New()

	// Navigate to pods (press 'P').
	m1, _ := m.Update(bubbletea.KeyMsg{Type: bubbletea.KeyRunes, Runes: []rune{'P'}})
	onPods := m1.(Model)
	if onPods.crumbs[0] != "pods" {
		t.Fatalf("expected pods crumb after pressing P, got %q", onPods.crumbs[0])
	}

	// Switch namespace — should stay on pods, not jump to workloads.
	m2, _ := onPods.Update(overlaypicker.SelectedMsg{Kind: "namespace", Value: "staging"})
	got := m2.(Model)
	if got.namespace != "staging" {
		t.Fatalf("expected namespace=staging, got %q", got.namespace)
	}
	if got.crumbs[0] != "pods" {
		t.Fatalf("expected pods crumb preserved after namespace switch, got %q", got.crumbs[0])
	}
}

func TestScopeSwitchFallsBackToParentListViewResource(t *testing.T) {
	m := New() // starts on workloads listview

	// Push a non-listview view on top to simulate a detail/related view.
	m.stack = append(m.stack, shortView{})
	m.crumbs = append(m.crumbs, "detail")

	// Switch namespace — top is not a listview, should fall back to the
	// workloads listview below it.
	updated, _ := m.Update(overlaypicker.SelectedMsg{Kind: "namespace", Value: "staging"})
	got := updated.(Model)
	if got.crumbs[0] != "workloads" {
		t.Fatalf("expected workloads crumb after fallback to parent, got %q", got.crumbs[0])
	}
	if len(got.stack) != 1 {
		t.Fatalf("expected stack reset to 1 entry, got %d", len(got.stack))
	}
}

func TestContextSwitchPreservesCurrentListViewResource(t *testing.T) {
	m := New()

	// Navigate to services (press 'S').
	m1, _ := m.Update(bubbletea.KeyMsg{Type: bubbletea.KeyRunes, Runes: []rune{'S'}})
	onServices := m1.(Model)

	// Switch context — should stay on services.
	m2, _ := onServices.Update(overlaypicker.SelectedMsg{Kind: "context", Value: "prod-cluster"})
	got := m2.(Model)
	if got.context != "prod-cluster" {
		t.Fatalf("expected context=prod-cluster, got %q", got.context)
	}
	if got.crumbs[0] != "services" {
		t.Fatalf("expected services crumb preserved after context switch, got %q", got.crumbs[0])
	}
}

func TestCommandBarSingleMatchPushesListAndDetail(t *testing.T) {
	m := New()
	m.width = 120
	m.height = 40
	err := m.runCommand("deploy api-gateway")
	if err != "" {
		t.Fatalf("expected no error running command, got %q", err)
	}
	model := m

	if len(model.stack) != 3 {
		t.Fatalf("expected stack len 3 (workloads->deployments->detail), got %d", len(model.stack))
	}
	if _, ok := model.stack[1].(*listview.View); !ok {
		t.Fatalf("expected second stack view to be listview, got %T", model.stack[1])
	}
	if _, ok := model.stack[2].(*detailview.View); !ok {
		t.Fatalf("expected third stack view to be detailview, got %T", model.stack[2])
	}
}

func TestCommandBarDescribeSubviewPushesDetailAndDescribe(t *testing.T) {
	m := New()
	m.width = 120
	m.height = 40

	err := m.runCommand("deploy api-gateway describe")
	if err != "" {
		t.Fatalf("expected no error running command, got %q", err)
	}
	model := m

	if len(model.stack) != 4 {
		t.Fatalf("expected stack depth 4 for describe path, got %d", len(model.stack))
	}
	if _, ok := model.stack[2].(*detailview.View); !ok {
		t.Fatalf("expected detail view before describe, got %T", model.stack[2])
	}
	if _, ok := model.stack[len(model.stack)-1].(*describeview.View); !ok {
		t.Fatalf("expected top view to be describeview, got %T", model.stack[len(model.stack)-1])
	}
}

func TestCommandBarLogsSubviewPushesDetailAndLogs(t *testing.T) {
	m := New()
	m.width = 120
	m.height = 40

	err := m.runCommand("deploy api-gateway logs")
	if err != "" {
		t.Fatalf("expected no error running command, got %q", err)
	}
	model := m

	if len(model.stack) != 4 {
		t.Fatalf("expected stack depth 4 for logs path, got %d", len(model.stack))
	}
	if _, ok := model.stack[2].(*detailview.View); !ok {
		t.Fatalf("expected detail view before logs, got %T", model.stack[2])
	}
	if _, ok := model.stack[len(model.stack)-1].(*listview.View); !ok {
		t.Fatalf("expected top view to be listview for deployment log target, got %T", model.stack[len(model.stack)-1])
	}
}

func TestCommandBarQSubmitQuits(t *testing.T) {
	m := New()
	m.cmdBar = commandbar.New()

	updated, cmd := m.Update(commandbar.SubmitMsg{Value: "q"})
	got := updated.(Model)
	if got.cmdBar != nil {
		t.Fatal("expected command bar to close on :q")
	}
	if cmd == nil {
		t.Fatal("expected quit command for :q")
	}
	msg := cmd()
	if _, ok := msg.(bubbletea.QuitMsg); !ok {
		t.Fatalf("expected bubbletea.QuitMsg for :q, got %T", msg)
	}
}

func TestCommandBarQuitSubmitQuits(t *testing.T) {
	m := New()
	m.cmdBar = commandbar.New()

	updated, cmd := m.Update(commandbar.SubmitMsg{Value: "  QuIt  "})
	got := updated.(Model)
	if got.cmdBar != nil {
		t.Fatal("expected command bar to close on :quit")
	}
	if cmd == nil {
		t.Fatal("expected quit command for :quit")
	}
	msg := cmd()
	if _, ok := msg.(bubbletea.QuitMsg); !ok {
		t.Fatalf("expected bubbletea.QuitMsg for :quit, got %T", msg)
	}
}

func TestBookmarkSetAndJump(t *testing.T) {
	m := New()

	// Start on workloads — set bookmark 1 via m+1.
	m1, _ := m.Update(bubbletea.KeyMsg{Type: bubbletea.KeyRunes, Runes: []rune{'m'}})
	afterM := m1.(Model)
	if !afterM.bookmarkMode {
		t.Fatal("expected bookmarkMode after pressing m")
	}
	m2, _ := afterM.Update(bubbletea.KeyMsg{Type: bubbletea.KeyRunes, Runes: []rune{'1'}})
	afterSet := m2.(Model)
	if afterSet.bookmarks[0] == nil {
		t.Fatal("expected bookmark 0 to be set")
	}
	if len(afterSet.bookmarks[0].stack) == 0 {
		t.Fatal("expected bookmark to capture the view stack")
	}

	// Navigate to Pods.
	m3, _ := afterSet.Update(bubbletea.KeyMsg{Type: bubbletea.KeyRunes, Runes: []rune{'P'}})
	onPods := m3.(Model)
	if lv, ok := onPods.top().(*listview.View); !ok || lv.Resource().Name() != "pods" {
		t.Fatal("expected to be on pods after pressing P")
	}

	// Jump back to bookmark 1 — should restore the workloads stack.
	m4, _ := onPods.Update(bubbletea.KeyMsg{Type: bubbletea.KeyRunes, Runes: []rune{'1'}})
	afterJump := m4.(Model)
	lv, ok := afterJump.top().(*listview.View)
	if !ok {
		t.Fatal("expected top view to be a listview after jump")
	}
	if lv.Resource().Name() != "workloads" {
		t.Fatalf("expected workloads after jump, got %q", lv.Resource().Name())
	}
}

func TestMergeLinePadsToAnchorWhenBackgroundIsShort(t *testing.T) {
	got := mergeLine("short", "BOX", 10)
	if got != "short     BOX" {
		t.Fatalf("expected overlay to start at anchor column, got %q", got)
	}
}

func TestCompositeOverlayKeepsAnchorOnShortHeaderLines(t *testing.T) {
	bg := "h1\nh2\nthis line is long enough"
	got := compositeOverlay(bg, "A\nB", 8, 0)
	lines := strings.Split(got, "\n")
	if lines[0] != "h1      A" {
		t.Fatalf("expected first overlaid line to honor anchor, got %q", lines[0])
	}
	if lines[1] != "h2      B" {
		t.Fatalf("expected second overlaid line to honor anchor, got %q", lines[1])
	}
}
