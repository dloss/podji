package resourcebrowser

import (
	"strings"
	"testing"

	bubbletea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/dloss/podji/internal/resources"
)

func TestFooterShowsFilterAndFind(t *testing.T) {
	v := New(resources.DefaultRegistry(), resources.StubCRDs())
	v.SetSize(120, 40)

	footer := ansi.Strip(v.Footer())
	if !strings.Contains(footer, "/ filter") {
		t.Fatalf("expected / filter hint, got: %s", footer)
	}
	if !strings.Contains(footer, "f find") {
		t.Fatalf("expected f find hint, got: %s", footer)
	}
	if !strings.Contains(footer, "s sort") {
		t.Fatalf("expected s sort hint, got: %s", footer)
	}
}

func TestSortPickSelectsColumnByCharAndCount(t *testing.T) {
	v := New(resources.DefaultRegistry(), resources.StubCRDs())
	v.SetSize(120, 40)

	v.Update(keyRunes('s'))
	v.Update(keyRunes('g'))
	if v.sortCol != 1 {
		t.Fatalf("expected group column sort from char key, got %d", v.sortCol)
	}

	v.Update(keyRunes('s'))
	v.Update(keyRunes('3'))
	if v.sortCol != 2 {
		t.Fatalf("expected version column sort from numeric key, got %d", v.sortCol)
	}
}

func TestSortPickerHidesDuplicateLeadKeys(t *testing.T) {
	v := New(resources.DefaultRegistry(), resources.StubCRDs())
	v.SetSize(120, 40)

	v.Update(keyRunes('s'))
	footer := ansi.Strip(v.Footer())
	if got := strings.Count(footer, "s/S"); got != 1 {
		t.Fatalf("expected one s/S binding in sort picker for duplicated key, got %d: %s", got, footer)
	}
}

func keyRunes(r ...rune) bubbletea.KeyMsg {
	return bubbletea.KeyMsg{Type: bubbletea.KeyRunes, Runes: r}
}
