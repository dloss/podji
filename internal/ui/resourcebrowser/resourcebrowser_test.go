package resourcebrowser

import (
	"strings"
	"testing"

	bubbletea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/dloss/podji/internal/resources"
	"github.com/dloss/podji/internal/ui/listview"
	"github.com/dloss/podji/internal/ui/viewstate"
)

func TestFooterShowsFilterAndFind(t *testing.T) {
	v := New(resources.DefaultRegistry(), resources.StubCRDs())
	v.SetSize(120, 40)

	footer := ansi.Strip(v.Footer())
	if !strings.Contains(footer, "& filter") {
		t.Fatalf("expected & filter hint, got: %s", footer)
	}
	if !strings.Contains(footer, "f find") {
		t.Fatalf("expected f find hint, got: %s", footer)
	}
	if !strings.Contains(footer, "s sort") {
		t.Fatalf("expected s sort hint, got: %s", footer)
	}
}

func TestFooterDoesNotShowRelatedHint(t *testing.T) {
	v := New(resources.DefaultRegistry(), resources.StubCRDs())
	v.SetSize(120, 40)

	footer := ansi.Strip(v.Footer())
	if strings.Contains(footer, "r related") {
		t.Fatalf("expected no related hint in resource browser footer, got: %s", footer)
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

func TestPVCKindUsesExpectedCasing(t *testing.T) {
	entries := buildEntries(resources.DefaultRegistry(), nil)
	for _, e := range entries {
		if strings.EqualFold(e.resource.Name(), "persistentvolumeclaims") {
			if e.kind != "PersistentVolumeClaims" {
				t.Fatalf("expected pvc kind casing to be PersistentVolumeClaims, got %q", e.kind)
			}
			return
		}
	}
	t.Fatal("expected persistentvolumeclaims entry in resource browser")
}

func TestIngressKindUsesExpectedCasing(t *testing.T) {
	entries := buildEntries(resources.DefaultRegistry(), nil)
	for _, e := range entries {
		if strings.EqualFold(e.resource.Name(), "ingresses") {
			if e.kind != "Ingress" {
				t.Fatalf("expected ingress kind casing to be Ingress, got %q", e.kind)
			}
			return
		}
	}
	t.Fatal("expected ingresses entry in resource browser")
}

func TestEnterUsesAdaptedResource(t *testing.T) {
	v := New(resources.DefaultRegistry(), resources.StubCRDs())
	v.SetSize(120, 40)

	adapted := resources.NewQueryResource("pods", nil, resources.NewPods())
	v.SetResourceAdapter(func(resources.ResourceType) resources.ResourceType {
		return adapted
	})

	update := v.Update(bubbletea.KeyMsg{Type: bubbletea.KeyEnter})
	if update.Action != viewstate.Push {
		t.Fatalf("expected push action, got %v", update.Action)
	}
	next, ok := update.Next.(*listview.View)
	if !ok {
		t.Fatalf("expected next view type *listview.View, got %T", update.Next)
	}
	if next.Resource() != adapted {
		t.Fatalf("expected adapted resource to be used, got %T", next.Resource())
	}
}

func keyRunes(r ...rune) bubbletea.KeyMsg {
	return bubbletea.KeyMsg{Type: bubbletea.KeyRunes, Runes: r}
}
