package columnpicker

import (
	"strings"
	"testing"

	bubbletea "github.com/charmbracelet/bubbletea"
	"github.com/dloss/podji/internal/resources"
)

func keyRunes(r ...rune) bubbletea.KeyMsg {
	return bubbletea.KeyMsg{Type: bubbletea.KeyRunes, Runes: r}
}

func TestResetReturnsToDefaultColumns(t *testing.T) {
	pool := []resources.TableColumn{
		{ID: "name", Name: "NAME", Default: true},
		{ID: "age", Name: "AGE", Default: true},
		{ID: "status", Name: "STATUS", Default: false},
	}
	labelPool := []resources.TableColumn{
		{ID: "label:app", Name: "APP", Default: false},
	}
	current := []string{"name", "status", "label:app"}

	p := New("pods", pool, labelPool, current)
	p.Update(keyRunes('d'))

	got := p.visibleIDs()
	want := []string{"name", "age"}

	if len(got) != len(want) {
		t.Fatalf("expected %d visible columns, got %d (%v)", len(want), len(got), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("expected visible[%d]=%q, got %q (%v)", i, want[i], got[i], got)
		}
	}
}

func TestViewTitleUsesColumn(t *testing.T) {
	p := New("pods", []resources.TableColumn{{ID: "name", Name: "NAME", Default: true}}, nil, []string{"name"})
	p.SetSize(120, 40)

	view := p.View()
	if !strings.Contains(view, "  Column  ") {
		t.Fatalf("expected title to contain Column, got %q", view)
	}
	if strings.Contains(view, "  columns  ") {
		t.Fatalf("expected old columns title to be absent, got %q", view)
	}
}

func TestViewShowsDiscoverableDefaultHintAndBuiltInSection(t *testing.T) {
	pool := []resources.TableColumn{
		{ID: "name", Name: "NAME", Default: true},
		{ID: "age", Name: "AGE", Default: true},
	}
	p := New("pods", pool, nil, []string{"name", "age"})
	p.SetSize(80, 20)

	view := p.View()
	if !strings.Contains(view, "built-in") {
		t.Fatalf("expected section label built-in, got %q", view)
	}
	if strings.Contains(view, "standard") {
		t.Fatalf("expected old section label standard to be absent, got %q", view)
	}
	if !strings.Contains(view, "d default") {
		t.Fatalf("expected default hint in footer, got %q", view)
	}
}
