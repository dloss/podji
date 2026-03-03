package data

import (
	"testing"

	"github.com/dloss/podji/internal/resources"
)

func TestReadRelationIndexWorkloadResolvesPodsAndServices(t *testing.T) {
	store, err := newKubeStore(fakeKubeAPI{
		contexts: []string{"dev"},
		listsByKey: map[string][]resources.ResourceItem{
			"dev/default/pods": {
				{Name: "api-1", Labels: map[string]string{"app": "api"}},
				{Name: "other-1", Labels: map[string]string{"app": "other"}},
			},
			"dev/default/services": {
				{Name: "api-svc", Selector: map[string]string{"app": "api"}},
			},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	item := resources.ResourceItem{Name: "api", Selector: map[string]string{"app": "api"}}
	got := store.RelationIndex().Related(store.Scope(), "workloads", item)
	if len(got["pods"]) != 1 || got["pods"][0].Name != "api-1" {
		t.Fatalf("expected workload related pod from live list, got %#v", got["pods"])
	}
	if len(got["services"]) != 1 || got["services"][0].Name != "api-svc" {
		t.Fatalf("expected workload related service from live list, got %#v", got["services"])
	}
}

func TestReadRelationIndexServiceResolvesBackends(t *testing.T) {
	store, err := newKubeStore(fakeKubeAPI{
		contexts: []string{"dev"},
		listsByKey: map[string][]resources.ResourceItem{
			"dev/default/pods": {
				{Name: "api-1", Labels: map[string]string{"app": "api"}},
				{Name: "api-2", Labels: map[string]string{"app": "api"}},
				{Name: "other-1", Labels: map[string]string{"app": "other"}},
			},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	item := resources.ResourceItem{Name: "api-svc", Selector: map[string]string{"app": "api"}}
	got := store.RelationIndex().Related(store.Scope(), "services", item)
	if len(got["backends"]) != 2 {
		t.Fatalf("expected 2 backend pods from live list, got %#v", got["backends"])
	}
}

func TestReadRelationIndexIngressServiceMappingUsesBackendList(t *testing.T) {
	store, err := newKubeStore(fakeKubeAPI{
		contexts: []string{"dev"},
		listsByKey: map[string][]resources.ResourceItem{
			"dev/default/services": {
				{Name: "api-svc"},
				{Name: "web-svc"},
			},
			"dev/default/ingresses": {
				{Name: "main", Extra: map[string]string{"services": "api-svc,web-svc"}},
			},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ing := resources.ResourceItem{Name: "main", Extra: map[string]string{"services": "api-svc,web-svc"}}
	gotIngress := store.RelationIndex().Related(store.Scope(), "ingresses", ing)
	if len(gotIngress["services"]) != 2 {
		t.Fatalf("expected ingress to resolve 2 services, got %#v", gotIngress["services"])
	}

	svc := resources.ResourceItem{Name: "api-svc"}
	gotService := store.RelationIndex().Related(store.Scope(), "services", svc)
	if len(gotService["ingresses"]) != 1 || gotService["ingresses"][0].Name != "main" {
		t.Fatalf("expected service to resolve ingress by backend map, got %#v", gotService["ingresses"])
	}
}

func TestReadRelationIndexPodConfigAndStorageFromRefs(t *testing.T) {
	store, err := newKubeStore(fakeKubeAPI{contexts: []string{"dev"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	pod := resources.ResourceItem{
		Name: "api-1",
		Extra: map[string]string{
			"config-refs": "app-config,feature-flags",
			"secret-refs": "api-secret",
			"pvc-refs":    "data-pvc",
		},
	}
	got := store.RelationIndex().Related(store.Scope(), "pods", pod)
	if len(got["config"]) != 3 {
		t.Fatalf("expected 3 config/secret refs, got %#v", got["config"])
	}
	if len(got["storage"]) != 1 || got["storage"][0].Name != "data-pvc" {
		t.Fatalf("expected pvc relation, got %#v", got["storage"])
	}
}

func TestReadRelationIndexPVCMountedByFromPodRefs(t *testing.T) {
	store, err := newKubeStore(fakeKubeAPI{
		contexts: []string{"dev"},
		listsByKey: map[string][]resources.ResourceItem{
			"dev/default/pods": {
				{Name: "api-1", Extra: map[string]string{"pvc-refs": "data-pvc"}},
				{Name: "worker-1", Extra: map[string]string{"pvc-refs": "other-pvc"}},
			},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	pvc := resources.ResourceItem{Name: "data-pvc"}
	got := store.RelationIndex().Related(store.Scope(), "persistentvolumeclaims", pvc)
	if len(got["mounted-by"]) != 1 || got["mounted-by"][0].Name != "api-1" {
		t.Fatalf("expected mounted-by relation from pod refs, got %#v", got["mounted-by"])
	}
}
