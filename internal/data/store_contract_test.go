package data

import (
	"testing"

	"github.com/dloss/podji/internal/resources"
)

func TestStoreContractScopeSwitchAcrossAdapters(t *testing.T) {
	kube, err := newKubeStore(fakeKubeAPI{contexts: []string{"dev"}})
	if err != nil {
		t.Fatalf("unexpected kube store error: %v", err)
	}

	cases := []struct {
		name  string
		store Store
	}{
		{name: "mock", store: NewMockStore()},
		{name: "kube", store: kube},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tc.store.SetScope(Scope{Context: "dev", Namespace: "staging"})
			if got := tc.store.Scope().Namespace; got != "staging" {
				t.Fatalf("expected namespace staging, got %q", got)
			}
			if got := tc.store.Registry().Namespace(); got != "staging" {
				t.Fatalf("expected registry namespace staging, got %q", got)
			}

			tc.store.SetScope(Scope{Context: "dev", Namespace: ""})
			if got := tc.store.Scope().Namespace; got != resources.DefaultNamespace {
				t.Fatalf("expected default namespace fallback, got %q", got)
			}
		})
	}
}

func TestStoreContractCommandQueryConsistencyAcrossAdapters(t *testing.T) {
	mock := NewMockStore()
	kube, err := newKubeStore(fakeKubeAPI{contexts: []string{"dev"}})
	if err != nil {
		t.Fatalf("unexpected kube store error: %v", err)
	}

	scope := Scope{Context: "dev", Namespace: resources.DefaultNamespace}
	mock.SetScope(scope)
	kube.SetScope(scope)

	if got, want := itemNames(kube.UnhealthyItems()), itemNames(mock.UnhealthyItems()); !sameNames(got, want) {
		t.Fatalf("unhealthy query mismatch: kube=%v mock=%v", got, want)
	}
	if got, want := itemNames(kube.PodsByRestarts()), itemNames(mock.PodsByRestarts()); !sameNames(got, want) {
		t.Fatalf("restarts query mismatch: kube=%v mock=%v", got, want)
	}
}

func itemNames(items []resources.ResourceItem) []string {
	out := make([]string, 0, len(items))
	for _, it := range items {
		out = append(out, it.Name)
	}
	return out
}

func sameNames(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
