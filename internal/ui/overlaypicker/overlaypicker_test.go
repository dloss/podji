package overlaypicker

import (
	"testing"

	bubbletea "github.com/charmbracelet/bubbletea"
	"github.com/dloss/podji/internal/ui/viewstate"
)

func keyRunes(r ...rune) bubbletea.KeyMsg {
	return bubbletea.KeyMsg{Type: bubbletea.KeyRunes, Runes: r}
}

func keyEsc() bubbletea.KeyMsg {
	return bubbletea.KeyMsg{Type: bubbletea.KeyEscape}
}

func keyEnter() bubbletea.KeyMsg {
	return bubbletea.KeyMsg{Type: bubbletea.KeyEnter}
}

func keyDown() bubbletea.KeyMsg {
	return bubbletea.KeyMsg{Type: bubbletea.KeyDown}
}

func keyUp() bubbletea.KeyMsg {
	return bubbletea.KeyMsg{Type: bubbletea.KeyUp}
}

func keyBackspace() bubbletea.KeyMsg {
	return bubbletea.KeyMsg{Type: bubbletea.KeyBackspace}
}

func TestFilterNarrowsItemList(t *testing.T) {
	p := New("namespace", []string{"default", "staging", "production"})

	p.Update(keyRunes('s'))

	got := p.filtered()
	for _, item := range got {
		found := false
		for _, c := range item {
			if c == 's' || c == 'S' {
				found = true
				break
			}
		}
		if !found {
			// check substring match
			if len(item) == 0 {
				t.Fatalf("unexpected empty item in filtered list")
			}
		}
	}
	if len(got) >= 3 {
		t.Fatalf("expected filter to narrow list, got %d items", len(got))
	}
}

func TestBackspaceRemovesLastFilterChar(t *testing.T) {
	p := New("namespace", []string{"default", "staging"})

	p.Update(keyRunes('s', 't'))
	if p.filter != "st" {
		t.Fatalf("expected filter=st, got %q", p.filter)
	}

	p.Update(keyBackspace())
	if p.filter != "s" {
		t.Fatalf("expected filter=s after backspace, got %q", p.filter)
	}
}

func TestEnterEmitsSelectedMsg(t *testing.T) {
	p := New("namespace", []string{"default", "staging"})
	p.SetSize(120, 40)

	update := p.Update(keyEnter())

	if update.Action != viewstate.Pop {
		t.Fatalf("expected Pop action on enter, got %v", update.Action)
	}
	if update.Cmd == nil {
		t.Fatal("expected Cmd to be non-nil on enter")
	}
	msg := update.Cmd()
	sel, ok := msg.(SelectedMsg)
	if !ok {
		t.Fatalf("expected SelectedMsg, got %T", msg)
	}
	if sel.Kind != "namespace" {
		t.Fatalf("expected Kind=namespace, got %q", sel.Kind)
	}
	if sel.Value != "default" {
		t.Fatalf("expected Value=default, got %q", sel.Value)
	}
}

func TestEscReturnsPop(t *testing.T) {
	p := New("context", []string{"prod", "staging"})

	update := p.Update(keyEsc())

	if update.Action != viewstate.Pop {
		t.Fatalf("expected Pop action on esc, got %v", update.Action)
	}
	if update.Cmd != nil {
		t.Fatal("expected nil Cmd on esc")
	}
}

func TestCursorClampsAtListBoundaries(t *testing.T) {
	p := New("namespace", []string{"a", "b", "c"})
	p.SetSize(120, 40)

	// Move up from start — cursor should stay at 0.
	p.Update(keyUp())
	if p.cursor != 0 {
		t.Fatalf("expected cursor=0 after up at start, got %d", p.cursor)
	}

	// Move down past end.
	p.Update(keyDown())
	p.Update(keyDown())
	p.Update(keyDown())
	p.Update(keyDown())
	if p.cursor != 2 {
		t.Fatalf("expected cursor=2 (last item) after excess down, got %d", p.cursor)
	}
}
