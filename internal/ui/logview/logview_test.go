package logview

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	bubbletea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/dloss/podji/internal/resources"
	"github.com/dloss/podji/internal/ui/viewstate"
)

type optionsLogsResource struct {
	base      resources.ResourceType
	tailCalls []int
	follow    []bool
	previous  []bool
	container []string
}

func (o *optionsLogsResource) Name() string                        { return o.base.Name() }
func (o *optionsLogsResource) Key() rune                           { return o.base.Key() }
func (o *optionsLogsResource) Items() []resources.ResourceItem     { return o.base.Items() }
func (o *optionsLogsResource) Sort(items []resources.ResourceItem) { o.base.Sort(items) }
func (o *optionsLogsResource) Detail(item resources.ResourceItem) resources.DetailData {
	return o.base.Detail(item)
}
func (o *optionsLogsResource) Logs(item resources.ResourceItem) []string { return o.base.Logs(item) }
func (o *optionsLogsResource) Events(item resources.ResourceItem) []string {
	return o.base.Events(item)
}
func (o *optionsLogsResource) YAML(item resources.ResourceItem) string { return o.base.YAML(item) }
func (o *optionsLogsResource) Describe(item resources.ResourceItem) string {
	return o.base.Describe(item)
}

func (o *optionsLogsResource) LogsWithOptions(ctx context.Context, item resources.ResourceItem, opts resources.LogOptions) ([]string, error) {
	o.tailCalls = append(o.tailCalls, opts.Tail)
	o.follow = append(o.follow, opts.Follow)
	o.previous = append(o.previous, opts.Previous)
	o.container = append(o.container, opts.Container)
	return []string{"line-a", "line-b"}, nil
}

func TestWrapLine(t *testing.T) {
	tests := []struct {
		name  string
		line  string
		width int
		want  []string
	}{
		{name: "empty", line: "", width: 4, want: []string{""}},
		{name: "short", line: "abc", width: 4, want: []string{"abc"}},
		{name: "exact", line: "abcd", width: 4, want: []string{"abcd"}},
		{name: "long", line: "abcdefghij", width: 4, want: []string{"abcd", "efgh", "ij"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := wrapLine(tt.line, tt.width)
			if len(got) != len(tt.want) {
				t.Fatalf("len(got)=%d, want %d", len(got), len(tt.want))
			}
			for i := range tt.want {
				if got[i] != tt.want[i] {
					t.Fatalf("got[%d]=%q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestWrapLines(t *testing.T) {
	got := wrapLines([]string{"abcd", "123456"}, 4)
	want := "abcd\n1234\n56"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestWrapLineUsesPrintableWidthForANSI(t *testing.T) {
	red := "\x1b[31m"
	reset := "\x1b[0m"
	line := red + "abcdef" + reset
	parts := wrapLine(line, 4)
	if len(parts) != 2 {
		t.Fatalf("expected 2 wrapped lines, got %d", len(parts))
	}
	if got := ansi.Strip(parts[0]); got != "abcd" {
		t.Fatalf("expected first part to be abcd, got %q", got)
	}
	if got := ansi.Strip(parts[1]); got != "ef" {
		t.Fatalf("expected second part to be ef, got %q", got)
	}
}

func TestFooterShowsSinceAndMatchStatus(t *testing.T) {
	v := New(resources.ResourceItem{Name: "api"}, resources.NewPods())
	v.SetSize(80, 20)
	// "/" enters search mode; type + enter commits and computes matches.
	v.Update(bubbletea.KeyMsg{Type: bubbletea.KeyRunes, Runes: []rune{'/'}})
	v.Update(bubbletea.KeyMsg{Type: bubbletea.KeyRunes, Runes: []rune{'e'}})
	v.Update(bubbletea.KeyMsg{Type: bubbletea.KeyRunes, Runes: []rune{'r'}})
	v.Update(bubbletea.KeyMsg{Type: bubbletea.KeyRunes, Runes: []rune{'r'}})
	v.Update(bubbletea.KeyMsg{Type: bubbletea.KeyRunes, Runes: []rune{'o'}})
	v.Update(bubbletea.KeyMsg{Type: bubbletea.KeyRunes, Runes: []rune{'r'}})
	v.Update(bubbletea.KeyMsg{Type: bubbletea.KeyEnter})
	// "." switches since window away from default 5m.
	v.Update(bubbletea.KeyMsg{Type: bubbletea.KeyRunes, Runes: []rune{'.'}})

	footer := ansi.Strip(v.Footer())
	if !strings.Contains(footer, "since") {
		t.Fatalf("expected footer to include since indicator, got %q", footer)
	}
	if !strings.Contains(footer, "match") {
		t.Fatalf("expected footer to include match indicator, got %q", footer)
	}
}

func TestSinceWindowRefetchesWithTailOptions(t *testing.T) {
	res := &optionsLogsResource{base: resources.NewPods()}
	v := New(resources.ResourceItem{Name: "api"}, res)
	if len(res.tailCalls) != 1 || res.tailCalls[0] != 200 {
		t.Fatalf("expected initial tail=200 fetch, got %#v", res.tailCalls)
	}
	upd := v.Update(bubbletea.KeyMsg{Type: bubbletea.KeyRunes, Runes: []rune{'.'}})
	if upd.Cmd == nil {
		t.Fatal("expected reload cmd after since-window change")
	}
	_ = upd.Cmd()
	if len(res.tailCalls) != 2 || res.tailCalls[1] != 500 {
		t.Fatalf("expected second tail=500 fetch after . window switch, got %#v", res.tailCalls)
	}
}

func TestFollowToggleRefetchesWithUpdatedFollowOption(t *testing.T) {
	res := &optionsLogsResource{base: resources.NewPods()}
	v := New(resources.ResourceItem{Name: "api"}, res)
	if len(res.follow) != 1 || !res.follow[0] {
		t.Fatalf("expected initial follow=true fetch, got %#v", res.follow)
	}
	upd := v.Update(bubbletea.KeyMsg{Type: bubbletea.KeyRunes, Runes: []rune{'f'}})
	if upd.Cmd == nil {
		t.Fatal("expected reload cmd after follow toggle")
	}
	_ = upd.Cmd()
	if len(res.follow) != 2 || res.follow[1] {
		t.Fatalf("expected follow=false refetch after toggle, got %#v", res.follow)
	}
}

func TestPreviousToggleRefetchesWithUpdatedPreviousOption(t *testing.T) {
	res := &optionsLogsResource{base: resources.NewPods()}
	v := New(resources.ResourceItem{Name: "api"}, res)
	if len(res.previous) != 1 || res.previous[0] {
		t.Fatalf("expected initial previous=false fetch, got %#v", res.previous)
	}
	upd := v.Update(bubbletea.KeyMsg{Type: bubbletea.KeyRunes, Runes: []rune{'p'}})
	if upd.Cmd == nil {
		t.Fatal("expected reload cmd after previous toggle")
	}
	_ = upd.Cmd()
	if len(res.previous) != 2 || !res.previous[1] {
		t.Fatalf("expected previous=true refetch after toggle, got %#v", res.previous)
	}
}

func TestSinceWindowCommaDotRefetchWithTailOptions(t *testing.T) {
	res := &optionsLogsResource{base: resources.NewPods()}
	v := New(resources.ResourceItem{Name: "api"}, res)

	upd := v.Update(bubbletea.KeyMsg{Type: bubbletea.KeyRunes, Runes: []rune{'.'}})
	if upd.Cmd == nil {
		t.Fatal("expected reload cmd after since-window forward alias")
	}
	_ = upd.Cmd()
	if got := res.tailCalls[len(res.tailCalls)-1]; got != 500 {
		t.Fatalf("expected tail=500 after '.', got %d", got)
	}

	upd = v.Update(bubbletea.KeyMsg{Type: bubbletea.KeyRunes, Runes: []rune{','}})
	if upd.Cmd == nil {
		t.Fatal("expected reload cmd after since-window backward alias")
	}
	_ = upd.Cmd()
	if got := res.tailCalls[len(res.tailCalls)-1]; got != 200 {
		t.Fatalf("expected tail=200 after ',', got %d", got)
	}
}

func TestLegacyLogKeysDoNotTriggerModeOrSince(t *testing.T) {
	res := &optionsLogsResource{base: resources.NewPods()}
	v := New(resources.ResourceItem{Name: "api"}, res)

	upd := v.Update(bubbletea.KeyMsg{Type: bubbletea.KeyRunes, Runes: []rune{'t'}})
	if upd.Cmd != nil {
		t.Fatal("expected no reload cmd for legacy mode key t")
	}

	upd = v.Update(bubbletea.KeyMsg{Type: bubbletea.KeyRunes, Runes: []rune{']'}})
	if upd.Cmd != nil {
		t.Fatal("expected no reload cmd for legacy since key ]")
	}

	upd = v.Update(bubbletea.KeyMsg{Type: bubbletea.KeyRunes, Runes: []rune{'['}})
	if upd.Cmd != nil {
		t.Fatal("expected no reload cmd for legacy since key [")
	}
}

type blockingLogsResource struct {
	base      resources.ResourceType
	ctxSeen   chan context.Context
	cancelled chan struct{}
}

func (b *blockingLogsResource) Name() string                        { return b.base.Name() }
func (b *blockingLogsResource) Key() rune                           { return b.base.Key() }
func (b *blockingLogsResource) Items() []resources.ResourceItem     { return b.base.Items() }
func (b *blockingLogsResource) Sort(items []resources.ResourceItem) { b.base.Sort(items) }
func (b *blockingLogsResource) Detail(item resources.ResourceItem) resources.DetailData {
	return b.base.Detail(item)
}
func (b *blockingLogsResource) Logs(item resources.ResourceItem) []string { return b.base.Logs(item) }
func (b *blockingLogsResource) Events(item resources.ResourceItem) []string {
	return b.base.Events(item)
}
func (b *blockingLogsResource) YAML(item resources.ResourceItem) string { return b.base.YAML(item) }
func (b *blockingLogsResource) Describe(item resources.ResourceItem) string {
	return b.base.Describe(item)
}

func (b *blockingLogsResource) LogsWithOptions(ctx context.Context, item resources.ResourceItem, opts resources.LogOptions) ([]string, error) {
	select {
	case b.ctxSeen <- ctx:
	default:
	}
	<-ctx.Done()
	select {
	case b.cancelled <- struct{}{}:
	default:
	}
	return nil, ctx.Err()
}

type streamingLogsResource struct {
	base      resources.ResourceType
	streamed  []string
	streamCtx chan context.Context
	streamErr error
}

func (s *streamingLogsResource) Name() string                        { return s.base.Name() }
func (s *streamingLogsResource) Key() rune                           { return s.base.Key() }
func (s *streamingLogsResource) Items() []resources.ResourceItem     { return s.base.Items() }
func (s *streamingLogsResource) Sort(items []resources.ResourceItem) { s.base.Sort(items) }
func (s *streamingLogsResource) Detail(item resources.ResourceItem) resources.DetailData {
	return s.base.Detail(item)
}
func (s *streamingLogsResource) Logs(item resources.ResourceItem) []string { return []string{"seed"} }
func (s *streamingLogsResource) Events(item resources.ResourceItem) []string {
	return s.base.Events(item)
}
func (s *streamingLogsResource) YAML(item resources.ResourceItem) string { return s.base.YAML(item) }
func (s *streamingLogsResource) Describe(item resources.ResourceItem) string {
	return s.base.Describe(item)
}
func (s *streamingLogsResource) LogsWithOptions(ctx context.Context, item resources.ResourceItem, opts resources.LogOptions) ([]string, error) {
	return []string{"seed"}, nil
}
func (s *streamingLogsResource) LogsStream(ctx context.Context, item resources.ResourceItem, opts resources.LogOptions, onLine func(string)) error {
	select {
	case s.streamCtx <- ctx:
	default:
	}
	for _, line := range s.streamed {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		onLine(line)
	}
	if s.streamErr != nil {
		return s.streamErr
	}
	return nil
}

type blockingStreamingLogsResource struct {
	base      resources.ResourceType
	streamCtx chan context.Context
}

func (s *blockingStreamingLogsResource) Name() string                        { return s.base.Name() }
func (s *blockingStreamingLogsResource) Key() rune                           { return s.base.Key() }
func (s *blockingStreamingLogsResource) Items() []resources.ResourceItem     { return s.base.Items() }
func (s *blockingStreamingLogsResource) Sort(items []resources.ResourceItem) { s.base.Sort(items) }
func (s *blockingStreamingLogsResource) Detail(item resources.ResourceItem) resources.DetailData {
	return s.base.Detail(item)
}
func (s *blockingStreamingLogsResource) Logs(item resources.ResourceItem) []string {
	return []string{"seed"}
}
func (s *blockingStreamingLogsResource) Events(item resources.ResourceItem) []string {
	return s.base.Events(item)
}
func (s *blockingStreamingLogsResource) YAML(item resources.ResourceItem) string {
	return s.base.YAML(item)
}
func (s *blockingStreamingLogsResource) Describe(item resources.ResourceItem) string {
	return s.base.Describe(item)
}
func (s *blockingStreamingLogsResource) LogsWithOptions(ctx context.Context, item resources.ResourceItem, opts resources.LogOptions) ([]string, error) {
	return []string{"seed"}, nil
}
func (s *blockingStreamingLogsResource) LogsStream(ctx context.Context, item resources.ResourceItem, opts resources.LogOptions, onLine func(string)) error {
	select {
	case s.streamCtx <- ctx:
	default:
	}
	<-ctx.Done()
	return ctx.Err()
}

func TestDisposeCancelsInFlightReload(t *testing.T) {
	res := &blockingLogsResource{
		base:      resources.NewPods(),
		ctxSeen:   make(chan context.Context, 1),
		cancelled: make(chan struct{}, 2),
	}
	v := New(resources.ResourceItem{Name: "api"}, res)
	upd := v.Update(bubbletea.KeyMsg{Type: bubbletea.KeyRunes, Runes: []rune{'f'}})
	if upd.Cmd == nil {
		t.Fatal("expected reload cmd")
	}
	done := make(chan bubbletea.Msg, 1)
	go func() { done <- upd.Cmd() }()
	select {
	case <-res.ctxSeen:
	case <-time.After(1 * time.Second):
		t.Fatal("expected blocking LogsWithOptions call to start")
	}
	v.Dispose()
	select {
	case <-res.cancelled:
	case <-time.After(1 * time.Second):
		t.Fatal("expected dispose to cancel in-flight reload")
	}
	select {
	case msg := <-done:
		result, ok := msg.(logReloadResultMsg)
		if !ok {
			t.Fatalf("expected logReloadResultMsg, got %T", msg)
		}
		if !errors.Is(result.err, context.Canceled) && !errors.Is(result.err, context.DeadlineExceeded) {
			t.Fatalf("expected cancellation error, got %v", result.err)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("expected reload command to return after cancellation")
	}
}

func TestInitStartsFollowStreamAndAppendsLines(t *testing.T) {
	res := &streamingLogsResource{
		base:      resources.NewPods(),
		streamed:  []string{"stream-a", "stream-b"},
		streamCtx: make(chan context.Context, 1),
	}
	v := New(resources.ResourceItem{Name: "api"}, res)
	cmd := v.Init()
	if cmd == nil {
		t.Fatal("expected init stream command")
	}
	msg1 := cmd()
	upd := v.Update(msg1)
	if upd.Cmd == nil {
		t.Fatal("expected follow-up stream cmd after first append")
	}
	msg2 := upd.Cmd()
	upd = v.Update(msg2)
	if upd.Cmd == nil {
		t.Fatal("expected follow-up stream cmd after second append")
	}
	_ = upd.Cmd() // done message
	if got := strings.Join(v.lines, "\n"); !strings.Contains(got, "stream-a") || !strings.Contains(got, "stream-b") {
		t.Fatalf("expected streamed lines in viewport content, got %q", got)
	}
}

func TestDisposeCancelsInFlightStream(t *testing.T) {
	res := &blockingStreamingLogsResource{
		base:      resources.NewPods(),
		streamCtx: make(chan context.Context, 1),
	}
	v := New(resources.ResourceItem{Name: "api"}, res)
	cmd := v.Init()
	if cmd == nil {
		t.Fatal("expected init stream command")
	}
	done := make(chan bubbletea.Msg, 1)
	go func() { done <- cmd() }()
	select {
	case <-res.streamCtx:
	case <-time.After(1 * time.Second):
		t.Fatal("expected stream context")
	}
	v.Dispose()
	select {
	case <-done:
	case <-time.After(1 * time.Second):
		t.Fatal("expected stream command to return after dispose")
	}
}

func TestFooterShowsStreamErrorIndicator(t *testing.T) {
	res := &streamingLogsResource{
		base:      resources.NewPods(),
		streamCtx: make(chan context.Context, 1),
		streamErr: errors.New("backend stream failed"),
	}
	v := New(resources.ResourceItem{Name: "api"}, res)
	v.SetSize(80, 20)
	cmd := v.Init()
	if cmd == nil {
		t.Fatal("expected init stream command")
	}
	msg := cmd()
	v.Update(msg)
	footer := ansi.Strip(v.Footer())
	if !strings.Contains(footer, "stream") || !strings.Contains(footer, "backend stream failed") {
		t.Fatalf("expected stream error indicator in footer, got %q", footer)
	}
}

func TestSearchModeFooterIndicator(t *testing.T) {
	v := New(resources.ResourceItem{Name: "api"}, resources.NewPods())
	v.SetSize(80, 20)

	v.Update(bubbletea.KeyMsg{Type: bubbletea.KeyRunes, Runes: []rune{'/'}})
	footer := ansi.Strip(v.Footer())
	if !strings.Contains(footer, "search") {
		t.Fatalf("expected search mode indicator in footer, got: %q", footer)
	}
	if !strings.Contains(footer, "▌") {
		t.Fatalf("expected cursor in footer when search active, got: %q", footer)
	}
	if !strings.Contains(footer, "enter") {
		t.Fatalf("expected enter confirm hint in search footer, got: %q", footer)
	}
	if !strings.Contains(footer, "esc") {
		t.Fatalf("expected esc cancel hint in search footer, got: %q", footer)
	}
}

func TestFilterModeFooterIndicator(t *testing.T) {
	v := New(resources.ResourceItem{Name: "api"}, resources.NewPods())
	v.SetSize(80, 20)

	v.Update(bubbletea.KeyMsg{Type: bubbletea.KeyRunes, Runes: []rune{'&'}})
	v.Update(bubbletea.KeyMsg{Type: bubbletea.KeyRunes, Runes: []rune{'e'}})
	footer := ansi.Strip(v.Footer())
	if !strings.Contains(footer, "filter") {
		t.Fatalf("expected filter mode indicator in footer, got: %q", footer)
	}
	if !strings.Contains(footer, "& e") {
		t.Fatalf("expected filter prompt value in footer, got: %q", footer)
	}
	if !strings.Contains(footer, "enter") || !strings.Contains(footer, "esc") {
		t.Fatalf("expected confirm/cancel hints in filter footer, got: %q", footer)
	}
}

func TestFilterAppliesAndNarrowsLines(t *testing.T) {
	v := New(resources.ResourceItem{Name: "api"}, resources.NewPods())
	v.SetSize(80, 20)

	before := len(v.lines)
	v.Update(bubbletea.KeyMsg{Type: bubbletea.KeyRunes, Runes: []rune{'&'}})
	v.Update(bubbletea.KeyMsg{Type: bubbletea.KeyRunes, Runes: []rune{'e'}})
	v.Update(bubbletea.KeyMsg{Type: bubbletea.KeyRunes, Runes: []rune{'r'}})
	v.Update(bubbletea.KeyMsg{Type: bubbletea.KeyRunes, Runes: []rune{'r'}})
	v.Update(bubbletea.KeyMsg{Type: bubbletea.KeyRunes, Runes: []rune{'o'}})
	v.Update(bubbletea.KeyMsg{Type: bubbletea.KeyRunes, Runes: []rune{'r'}})
	v.Update(bubbletea.KeyMsg{Type: bubbletea.KeyEnter})

	if strings.TrimSpace(v.filterValue) != "error" {
		t.Fatalf("expected committed filter value to be error, got %q", v.filterValue)
	}
	if len(v.lines) == 0 {
		t.Fatal("expected filtered logs to retain matching lines")
	}
	if len(v.lines) >= before {
		t.Fatalf("expected filtered logs to narrow line set, got before=%d after=%d", before, len(v.lines))
	}
}

func TestSearchBackKeyMovesToPreviousMatch(t *testing.T) {
	v := New(resources.ResourceItem{Name: "api"}, resources.NewPods())
	v.SetSize(80, 20)

	v.Update(bubbletea.KeyMsg{Type: bubbletea.KeyRunes, Runes: []rune{'/'}})
	v.Update(bubbletea.KeyMsg{Type: bubbletea.KeyRunes, Runes: []rune{'e'}})
	v.Update(bubbletea.KeyMsg{Type: bubbletea.KeyEnter})
	if len(v.matchLines) < 2 {
		t.Fatalf("expected at least 2 matches for test, got %d", len(v.matchLines))
	}

	first := v.matchLines[0]
	v.Update(bubbletea.KeyMsg{Type: bubbletea.KeyRunes, Runes: []rune{'n'}})
	second := v.matchLines[v.matchIndex]
	if second == first {
		t.Fatal("expected n to advance to a different match")
	}

	v.Update(bubbletea.KeyMsg{Type: bubbletea.KeyRunes, Runes: []rune{'b'}})
	if got := v.matchLines[v.matchIndex]; got != first {
		t.Fatalf("expected b to return to first match line %d, got %d", first, got)
	}
}

func TestContainerKeyPopsWhenContainerSelected(t *testing.T) {
	v := NewWithContainer(resources.ResourceItem{Name: "api"}, resources.NewPods(), "api")
	update := v.Update(bubbletea.KeyMsg{Type: bubbletea.KeyRunes, Runes: []rune{'c'}})
	if update.Action != viewstate.Pop {
		t.Fatalf("expected c to pop from container logs, got %v", update.Action)
	}
}

func TestContainerKeyDoesNothingWithoutFactory(t *testing.T) {
	v := New(resources.ResourceItem{Name: "api"}, resources.NewPods())
	update := v.Update(bubbletea.KeyMsg{Type: bubbletea.KeyRunes, Runes: []rune{'c'}})
	if update.Action != viewstate.None {
		t.Fatalf("expected c to be a no-op without factory, got %v", update.Action)
	}
}

func TestContainerKeyPushesPickerWhenFactorySet(t *testing.T) {
	v := New(resources.ResourceItem{Name: "api"}, resources.NewPods())
	var factoryCalled bool
	v.ContainerViewFactory = func(item resources.ResourceItem, res resources.ResourceType) viewstate.View {
		factoryCalled = true
		return New(resources.ResourceItem{Name: "api"}, resources.NewPods())
	}
	update := v.Update(bubbletea.KeyMsg{Type: bubbletea.KeyRunes, Runes: []rune{'c'}})
	if update.Action != viewstate.Push {
		t.Fatalf("expected c to push container picker, got %v", update.Action)
	}
	if !factoryCalled {
		t.Fatal("expected ContainerViewFactory to be called")
	}
}

func TestFooterOmitsContainerKeyWithoutFactory(t *testing.T) {
	v := New(resources.ResourceItem{Name: "api"}, resources.NewPods())
	v.SetSize(80, 20)
	footer := ansi.Strip(v.Footer())
	if strings.Contains(footer, "container") {
		t.Fatalf("expected footer to omit container binding when no container context, got %q", footer)
	}
}

func TestFooterShowsContainerKeyWhenContainerSelected(t *testing.T) {
	v := NewWithContainer(resources.ResourceItem{Name: "api"}, resources.NewPods(), "api")
	v.SetSize(80, 20)
	footer := ansi.Strip(v.Footer())
	if !strings.Contains(footer, "container") {
		t.Fatalf("expected footer to show 'c container' when container is selected, got %q", footer)
	}
}

func TestFooterShowsContainerKeyWhenFactorySet(t *testing.T) {
	v := New(resources.ResourceItem{Name: "api"}, resources.NewPods())
	v.SetSize(80, 20)
	v.ContainerViewFactory = func(item resources.ResourceItem, res resources.ResourceType) viewstate.View {
		return New(resources.ResourceItem{Name: "api"}, resources.NewPods())
	}
	footer := ansi.Strip(v.Footer())
	if !strings.Contains(footer, "container") {
		t.Fatalf("expected footer to show 'c container' when factory is set, got %q", footer)
	}
}

func TestContainerSelectionPropagatesToLogOptions(t *testing.T) {
	res := &optionsLogsResource{base: resources.NewPods()}
	v := NewWithContainer(resources.ResourceItem{Name: "api"}, res, "sidecar")
	if len(res.container) != 1 || res.container[0] != "sidecar" {
		t.Fatalf("expected initial container option to be propagated, got %#v", res.container)
	}
	upd := v.Update(bubbletea.KeyMsg{Type: bubbletea.KeyRunes, Runes: []rune{'.'}})
	if upd.Cmd == nil {
		t.Fatal("expected reload cmd after since-window change")
	}
	_ = upd.Cmd()
	if len(res.container) < 2 || res.container[len(res.container)-1] != "sidecar" {
		t.Fatalf("expected refetch to keep container option, got %#v", res.container)
	}
}
