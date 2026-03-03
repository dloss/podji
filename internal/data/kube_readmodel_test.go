package data

import (
	"context"
	"testing"

	"github.com/dloss/podji/internal/resources"
)

type fallbackDetailReadModel struct {
	*MockReadModel
}

func (f fallbackDetailReadModel) Detail(resourceName string, item resources.ResourceItem, scope Scope) (resources.DetailData, error) {
	return resources.DetailData{
		Summary: []resources.SummaryField{{Key: "status", Value: "from-fallback"}},
	}, nil
}

func TestKubeReadModelUsesAPIForPodLogs(t *testing.T) {
	api := fakeKubeAPI{
		logsByKey: map[string][]string{
			"dev/default/api-1": {"live-a", "live-b"},
		},
	}
	read := NewKubeReadModel(
		NewMockReadModel(resources.DefaultRegistry()),
		api,
		func() Scope { return Scope{Context: "dev", Namespace: "default"} },
		nil,
		nil,
		nil,
	)

	got, err := read.Logs("pods", resources.ResourceItem{Name: "api-1"}, Scope{})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(got) < 2 || got[0] != "live-a" || got[1] != "live-b" {
		t.Fatalf("expected live pod logs from api, got %#v", got)
	}
}

func TestKubeReadModelLogsWithContextCancelled(t *testing.T) {
	api := fakeKubeAPI{
		logsByKey: map[string][]string{
			"dev/default/api-1": {"live-a", "live-b"},
		},
	}
	read := NewKubeReadModel(
		NewMockReadModel(resources.DefaultRegistry()),
		api,
		func() Scope { return Scope{Context: "dev", Namespace: "default"} },
		nil,
		nil,
		nil,
	)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := read.LogsWithContext(ctx, "pods", resources.ResourceItem{Name: "api-1"}, Scope{}, LogOptions{Tail: 10})
	if err == nil {
		t.Fatal("expected cancellation error, got nil")
	}
}

func TestKubeReadModelFallsBackForNonPodLogs(t *testing.T) {
	reg := resources.DefaultRegistry()
	read := NewKubeReadModel(
		NewMockReadModel(reg),
		fakeKubeAPI{},
		func() Scope { return Scope{Context: "dev", Namespace: "default"} },
		nil,
		nil,
		nil,
	)

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

func TestKubeReadModelUsesAPIForPodList(t *testing.T) {
	read := NewKubeReadModel(
		NewMockReadModel(resources.DefaultRegistry()),
		fakeKubeAPI{
			listsByKey: map[string][]resources.ResourceItem{
				"dev/default/pods": {{Name: "live-pod-a"}},
			},
		},
		func() Scope { return Scope{Context: "dev", Namespace: "default"} },
		nil,
		nil,
		nil,
	)

	got, err := read.List("pods", Scope{})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(got) != 1 || got[0].Name != "live-pod-a" {
		t.Fatalf("expected live pod list, got %#v", got)
	}
}

func TestKubeReadModelFallsBackWhenListUnsupported(t *testing.T) {
	reg := resources.DefaultRegistry()
	read := NewKubeReadModel(
		NewMockReadModel(reg),
		fakeKubeAPI{
			listErrByKey: map[string]error{
				"dev/default/configmaps": ErrListNotSupported,
			},
		},
		func() Scope { return Scope{Context: "dev", Namespace: "default"} },
		nil,
		nil,
		nil,
	)

	got, err := read.List("configmaps", Scope{})
	if err != nil {
		t.Fatalf("expected fallback list success, got %v", err)
	}
	if len(got) == 0 {
		t.Fatalf("expected fallback items, got %#v", got)
	}
}

func TestKubeReadModelUsesLiveDetailForPods(t *testing.T) {
	reg := resources.DefaultRegistry()
	read := NewKubeReadModel(
		fallbackDetailReadModel{MockReadModel: NewMockReadModel(reg)},
		fakeKubeAPI{},
		func() Scope { return Scope{Context: "dev", Namespace: "default"} },
		nil,
		nil,
		nil,
	)
	detail, err := read.Detail("pods", resources.ResourceItem{
		Name:   "api-1",
		Status: "Running",
		Ready:  "1/1",
		Labels: map[string]string{"app": "api"},
		Extra: map[string]string{
			"node":       "worker-1",
			"ip":         "10.0.0.1",
			"qos":        "Burstable",
			"containers": "api,sidecar",
			"images":     "myco/api:v1,envoy:v1",
		},
	}, Scope{})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(detail.Summary) == 0 || detail.Summary[0].Value != "Running" {
		t.Fatalf("expected live detail summary from item status, got %#v", detail.Summary)
	}
	if len(detail.Containers) != 2 {
		t.Fatalf("expected container rows from live item metadata, got %#v", detail.Containers)
	}
}
