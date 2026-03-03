package logview

import (
	"context"
	"strings"
	"testing"

	bubbletea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/dloss/podji/internal/resources"
	"github.com/dloss/podji/internal/ui/viewstate"
)

type optionsLogsResource struct {
	base      resources.ResourceType
	tailCalls []int
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
	// "]" switches since window away from default 5m.
	v.Update(bubbletea.KeyMsg{Type: bubbletea.KeyRunes, Runes: []rune{']'}})

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
	v.Update(bubbletea.KeyMsg{Type: bubbletea.KeyRunes, Runes: []rune{']'}})
	if len(res.tailCalls) != 2 || res.tailCalls[1] != 500 {
		t.Fatalf("expected second tail=500 fetch after ] window switch, got %#v", res.tailCalls)
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
