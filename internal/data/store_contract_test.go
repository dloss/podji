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

func TestStoreContractKubeQueriesDoNotFallbackToMockData(t *testing.T) {
	kube, err := newKubeStore(fakeKubeAPI{contexts: []string{"dev"}})
	if err != nil {
		t.Fatalf("unexpected kube store error: %v", err)
	}

	scope := Scope{Context: "dev", Namespace: resources.DefaultNamespace}
	kube.SetScope(scope)

	if got := kube.UnhealthyItems(); len(got) != 0 {
		t.Fatalf("expected no unhealthy items without live list support, got %#v", got)
	}
	if got := kube.PodsByRestarts(); len(got) != 0 {
		t.Fatalf("expected no restart items without live list support, got %#v", got)
	}
}
