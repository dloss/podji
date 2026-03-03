package data

import (
	"testing"

	"github.com/dloss/podji/internal/resources"
)

func TestKubeReadModelUsesAPIForPodLogs(t *testing.T) {
	api := fakeKubeAPI{
		logsByKey: map[string][]string{
			"dev/default/api-1": {"live-a", "live-b"},
		},
	}
	read := NewKubeReadModel(NewMockReadModel(resources.DefaultRegistry()), api, func() Scope {
		return Scope{Context: "dev", Namespace: "default"}
	}, nil)

	got, err := read.Logs("pods", resources.ResourceItem{Name: "api-1"}, Scope{})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(got) < 2 || got[0] != "live-a" || got[1] != "live-b" {
		t.Fatalf("expected live pod logs from api, got %#v", got)
	}
}

func TestKubeReadModelFallsBackForNonPodLogs(t *testing.T) {
	reg := resources.DefaultRegistry()
	read := NewKubeReadModel(NewMockReadModel(reg), fakeKubeAPI{}, func() Scope {
		return Scope{Context: "dev", Namespace: "default"}
	}, nil)

	got, err := read.Logs("workloads", resources.ResourceItem{Name: "api-gateway"}, Scope{})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(got) == 0 {
		t.Fatalf("expected fallback logs for workload, got %#v", got)
	}
}

func TestKubeStoreAdaptedPodUsesKubeReadModelForLogs(t *testing.T) {
	store, err := newKubeStore(fakeKubeAPI{
		contexts: []string{"dev"},
		logsByKey: map[string][]string{
			"dev/default/api": {"line-a", "line-b"},
		},
	})
	if err != nil {
		t.Fatalf("unexpected kube store error: %v", err)
	}

	pods := store.AdaptResource(store.Registry().ByName("pods"))
	got := pods.Logs(resources.ResourceItem{Name: "api"})
	if len(got) < 2 || got[0] != "line-a" || got[1] != "line-b" {
		t.Fatalf("expected adapted resource to use kube read model logs, got %#v", got)
	}
}
