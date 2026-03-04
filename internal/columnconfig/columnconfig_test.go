package columnconfig

import (
	"testing"

	"github.com/dloss/podji/internal/resources"
)

func TestSetGetAllowsEmptySelection(t *testing.T) {
	store := &Store{configs: make(map[string]ColumnConfig)}
	pool := []resources.TableColumn{
		{ID: "name", Name: "NAME", Default: true},
		{ID: "namespace", Name: "NAMESPACE", Default: true},
	}

	store.Set("pods", nil)
	got := store.Get("pods", pool)
	if len(got) != 0 {
		t.Fatalf("expected empty columns after empty selection, got %v", got)
	}
}

func TestSetGetPreservesExplicitOrderWithoutInjectingName(t *testing.T) {
	store := &Store{configs: make(map[string]ColumnConfig)}
	pool := []resources.TableColumn{
		{ID: "name", Name: "NAME", Default: true},
		{ID: "namespace", Name: "NAMESPACE", Default: true},
		{ID: "age", Name: "AGE", Default: true},
	}

	store.Set("pods", []string{"namespace", "age"})
	got := store.Get("pods", pool)
	want := []string{"namespace", "age"}
	if len(got) != len(want) {
		t.Fatalf("expected %d columns, got %d (%v)", len(want), len(got), got)
	}
	for i := range want {
		if got[i].ID != want[i] {
			t.Fatalf("expected column[%d]=%q, got %q", i, want[i], got[i].ID)
		}
	}
}
