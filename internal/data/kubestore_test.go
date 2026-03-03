package data

import (
	"errors"
	"strings"
	"testing"

	"github.com/dloss/podji/internal/resources"
)

type fakeKubeAPI struct {
	contexts        []string
	contextErr      error
	namespacesByCtx map[string][]string
	namespaceErr    map[string]error
	listsByKey      map[string][]resources.ResourceItem
	listErrByKey    map[string]error
	logsByKey       map[string][]string
	logErrByKey     map[string]error
	eventsByKey     map[string][]string
	eventErrByKey   map[string]error
}

func (f fakeKubeAPI) Contexts() ([]string, error) {
	if f.contextErr != nil {
		return nil, f.contextErr
	}
	out := make([]string, len(f.contexts))
	copy(out, f.contexts)
	return out, nil
}

func (f fakeKubeAPI) Namespaces(context string) ([]string, error) {
	if err := f.namespaceErr[context]; err != nil {
		return nil, err
	}
	out := make([]string, len(f.namespacesByCtx[context]))
	copy(out, f.namespacesByCtx[context])
	return out, nil
}

func (f fakeKubeAPI) ListResources(context, namespace, resourceName string) ([]resources.ResourceItem, error) {
	key := context + "/" + namespace + "/" + resourceName
	if err := f.listErrByKey[key]; err != nil {
		return nil, err
	}
	out := make([]resources.ResourceItem, len(f.listsByKey[key]))
	copy(out, f.listsByKey[key])
	return out, nil
}

func (f fakeKubeAPI) PodLogs(context, namespace, pod string, tail int) ([]string, error) {
	key := context + "/" + namespace + "/" + pod
	if err := f.logErrByKey[key]; err != nil {
		return nil, err
	}
	out := make([]string, len(f.logsByKey[key]))
	copy(out, f.logsByKey[key])
	return out, nil
}

func (f fakeKubeAPI) PodEvents(context, namespace, pod string) ([]string, error) {
	key := context + "/" + namespace + "/" + pod
	if err := f.eventErrByKey[key]; err != nil {
		return nil, err
	}
	out := make([]string, len(f.eventsByKey[key]))
	copy(out, f.eventsByKey[key])
	return out, nil
}

func TestNewKubeStoreUsesFirstSortedContext(t *testing.T) {
	store, err := newKubeStore(fakeKubeAPI{
		contexts: []string{"prod", "staging", "dev"},
	})
	if err != nil {
		t.Fatalf("expected kube store creation to succeed, got %v", err)
	}
	if got := store.Scope().Context; got != "prod" {
		t.Fatalf("expected first context prod, got %q", got)
	}
}

func TestKubeStoreNamespaceNamesFallbackOnError(t *testing.T) {
	store, err := newKubeStore(fakeKubeAPI{
		contexts: []string{"dev"},
		namespaceErr: map[string]error{
			"dev": errors.New("boom"),
		},
	})
	if err != nil {
		t.Fatalf("unexpected error creating kube store: %v", err)
	}
	got := store.NamespaceNames()
	want := []string{resources.AllNamespaces, resources.DefaultNamespace}
	if len(got) != len(want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("expected %v, got %v", want, got)
		}
	}
}

func TestKubeStoreNamespaceNamesUsesContext(t *testing.T) {
	store, err := newKubeStore(fakeKubeAPI{
		contexts: []string{"dev", "prod"},
		namespacesByCtx: map[string][]string{
			"dev": {"kube-system", "default"},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error creating kube store: %v", err)
	}
	got := store.NamespaceNames()
	want := []string{resources.AllNamespaces, "kube-system", "default"}
	if len(got) != len(want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("expected %v, got %v", want, got)
		}
	}
}

func TestKubeStoreSetScopeUpdatesRegistryNamespace(t *testing.T) {
	store, err := newKubeStore(fakeKubeAPI{
		contexts: []string{"dev"},
	})
	if err != nil {
		t.Fatalf("unexpected error creating kube store: %v", err)
	}
	store.SetScope(Scope{Context: "dev", Namespace: "staging"})
	if got := store.Registry().Namespace(); got != "staging" {
		t.Fatalf("expected registry namespace staging, got %q", got)
	}
}

func TestKubeStoreStatusDegradedAfterDiscoveryError(t *testing.T) {
	store, err := newKubeStore(fakeKubeAPI{
		contexts: []string{"dev"},
		namespaceErr: map[string]error{
			"dev": errors.New("connection refused"),
		},
	})
	if err != nil {
		t.Fatalf("unexpected error creating kube store: %v", err)
	}
	_ = store.NamespaceNames()
	status := store.Status()
	if status.State != StoreStateUnreachable {
		t.Fatalf("expected degraded status, got %#v", status)
	}
	if !strings.Contains(status.Message, "connection refused") {
		t.Fatalf("expected discovery error in status message, got %#v", status)
	}
}

func TestKubeStoreStatusForbiddenOnPermissionError(t *testing.T) {
	store, err := newKubeStore(fakeKubeAPI{
		contexts: []string{"dev"},
		namespaceErr: map[string]error{
			"dev": errors.New("forbidden: User cannot list namespaces"),
		},
	})
	if err != nil {
		t.Fatalf("unexpected error creating kube store: %v", err)
	}
	_ = store.NamespaceNames()
	status := store.Status()
	if status.State != StoreStateForbidden {
		t.Fatalf("expected forbidden status, got %#v", status)
	}
}

func TestKubeStorePodLogsFetcherWired(t *testing.T) {
	store, err := newKubeStore(fakeKubeAPI{
		contexts: []string{"dev"},
		logsByKey: map[string][]string{
			"dev/default/api": {"line-a", "line-b"},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error creating kube store: %v", err)
	}
	pods, ok := store.Registry().ByName("pods").(*resources.Pods)
	if !ok {
		t.Fatalf("expected pods resource type, got %T", store.Registry().ByName("pods"))
	}
	lines := pods.Logs(resources.ResourceItem{Name: "api"})
	if len(lines) < 2 || lines[0] != "line-a" || lines[1] != "line-b" {
		t.Fatalf("expected live log lines, got %#v", lines)
	}
}

func TestKubeStorePodEventsFetcherWired(t *testing.T) {
	store, err := newKubeStore(fakeKubeAPI{
		contexts: []string{"dev"},
		eventsByKey: map[string][]string{
			"dev/default/api": {"2026-03-01T12:00:00Z   Warning   BackOff   Back-off restarting failed container"},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error creating kube store: %v", err)
	}
	pods, ok := store.Registry().ByName("pods").(*resources.Pods)
	if !ok {
		t.Fatalf("expected pods resource type, got %T", store.Registry().ByName("pods"))
	}
	lines := pods.Events(resources.ResourceItem{Name: "api"})
	if len(lines) == 0 || !strings.Contains(lines[0], "BackOff") {
		t.Fatalf("expected live event line, got %#v", lines)
	}
}
