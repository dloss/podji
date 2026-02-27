package logview

import (
	"strings"
	"testing"

	bubbletea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/dloss/podji/internal/resources"
	"github.com/dloss/podji/internal/ui/viewstate"
)

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

func TestContainerKeyPopsWhenContainerSelected(t *testing.T) {
	v := NewWithContainer(resources.ResourceItem{Name: "api"}, resources.NewPods(), "api")
	update := v.Update(bubbletea.KeyMsg{Type: bubbletea.KeyRunes, Runes: []rune{'c'}})
	if update.Action != viewstate.Pop {
		t.Fatalf("expected c to pop from container logs, got %v", update.Action)
	}
}
