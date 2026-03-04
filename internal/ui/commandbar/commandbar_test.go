package commandbar

import (
	"strings"
	"testing"

	bubbletea "github.com/charmbracelet/bubbletea"
)

func TestEnterSubmitsTrimmedValueAndStoresHistory(t *testing.T) {
	m := New()
	_, _, _ = m.Update(bubbletea.KeyMsg{Type: bubbletea.KeyRunes, Runes: []rune(" ")})
	_, _, _ = m.Update(bubbletea.KeyMsg{Type: bubbletea.KeyRunes, Runes: []rune("p")})
	_, _, _ = m.Update(bubbletea.KeyMsg{Type: bubbletea.KeyRunes, Runes: []rune("o")})
	_, _, _ = m.Update(bubbletea.KeyMsg{Type: bubbletea.KeyRunes, Runes: []rune("d")})
	_, _, _ = m.Update(bubbletea.KeyMsg{Type: bubbletea.KeyRunes, Runes: []rune(" ")})

	_, cmd, closed := m.Update(bubbletea.KeyMsg{Type: bubbletea.KeyEnter})
	if !closed {
		t.Fatal("expected command bar to close on enter")
	}
	if cmd == nil {
		t.Fatal("expected submit command on enter")
	}
	msg, ok := cmd().(SubmitMsg)
	if !ok {
		t.Fatalf("expected SubmitMsg, got %T", cmd())
	}
	if msg.Value != "pod" {
		t.Fatalf("expected trimmed value 'pod', got %q", msg.Value)
	}
	if got := m.Input(); got != "" {
		t.Fatalf("expected input reset after submit, got %q", got)
	}
	if len(m.history) != 1 || m.history[0] != "pod" {
		t.Fatalf("expected history to contain submitted command, got %#v", m.history)
	}
}

func TestHistoryNavigationUpAndDown(t *testing.T) {
	m := New()
	m.history = []string{"po api", "deploy api-gateway"}

	_, _, _ = m.Update(bubbletea.KeyMsg{Type: bubbletea.KeyUp})
	if got := m.Input(); got != "deploy api-gateway" {
		t.Fatalf("expected most recent history entry, got %q", got)
	}

	_, _, _ = m.Update(bubbletea.KeyMsg{Type: bubbletea.KeyUp})
	if got := m.Input(); got != "po api" {
		t.Fatalf("expected previous history entry, got %q", got)
	}

	_, _, _ = m.Update(bubbletea.KeyMsg{Type: bubbletea.KeyDown})
	if got := m.Input(); got != "deploy api-gateway" {
		t.Fatalf("expected next history entry, got %q", got)
	}

	_, _, _ = m.Update(bubbletea.KeyMsg{Type: bubbletea.KeyDown})
	if got := m.Input(); got != "" {
		t.Fatalf("expected empty input when leaving history, got %q", got)
	}
}

func TestEscClosesWithoutSubmit(t *testing.T) {
	m := New()
	_, _, _ = m.Update(bubbletea.KeyMsg{Type: bubbletea.KeyRunes, Runes: []rune("x")})

	_, cmd, closed := m.Update(bubbletea.KeyMsg{Type: bubbletea.KeyEsc})
	if !closed {
		t.Fatal("expected command bar to close on esc")
	}
	if cmd != nil {
		t.Fatal("expected no submit command on esc")
	}
}

func TestViewShowsSuggestionAndError(t *testing.T) {
	m := New()
	m.SetSize(120)
	m.SetError("unknown command")
	_, _, _ = m.Update(bubbletea.KeyMsg{Type: bubbletea.KeyRunes, Runes: []rune("po")})

	view := m.View("ds")
	if !strings.Contains(view, ": po") {
		t.Fatalf("expected prompt and input in view, got %q", view)
	}
	if !strings.Contains(view, "unknown command") {
		t.Fatalf("expected error in view, got %q", view)
	}
	if !strings.Contains(view, "█") {
		t.Fatalf("expected cursor block in view, got %q", view)
	}
}
