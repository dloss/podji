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

type fakeKubeAPIWithMeta struct {
	fakeKubeAPI
	cacheBacked bool
}

func (f fakeKubeAPIWithMeta) ListResourcesMeta(context, namespace, resourceName string) ([]resources.ResourceItem, bool, error) {
	items, err := f.fakeKubeAPI.ListResources(context, namespace, resourceName)
	return items, f.cacheBacked, err
}

type fakeKubeAPIWithMetaSequence struct {
	fakeKubeAPI
	sequence []bool
	calls    int
}

func (f *fakeKubeAPIWithMetaSequence) ListResourcesMeta(context, namespace, resourceName string) ([]resources.ResourceItem, bool, error) {
	items, err := f.fakeKubeAPI.ListResources(context, namespace, resourceName)
	cacheBacked := false
	if len(f.sequence) == 0 {
		cacheBacked = false
	} else if f.calls < len(f.sequence) {
		cacheBacked = f.sequence[f.calls]
	} else {
		cacheBacked = f.sequence[len(f.sequence)-1]
	}
	f.calls++
	return items, cacheBacked, err
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
	items, ok := f.listsByKey[key]
	if !ok {
		return nil, ErrListNotSupported
	}
	out := make([]resources.ResourceItem, len(items))
	copy(out, items)
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
	if status := store.Status(); status.State != StoreStateLoading {
		t.Fatalf("expected initial loading status, got %#v", status)
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
	if status := store.Status(); status.State != StoreStateLoading {
		t.Fatalf("expected loading status after scope change, got %#v", status)
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

func TestKubeStoreStatusPartialWhenListUnsupported(t *testing.T) {
	store, err := newKubeStore(fakeKubeAPI{
		contexts: []string{"dev"},
	})
	if err != nil {
		t.Fatalf("unexpected error creating kube store: %v", err)
	}
	_, err = store.ReadModel().List("configmaps", store.Scope())
	if !errors.Is(err, ErrListNotSupported) {
		t.Fatalf("expected unsupported list error, got %v", err)
	}
	status := store.Status()
	if status.State != StoreStatePartial {
		t.Fatalf("expected partial status after unsupported live list, got %#v", status)
	}
}

func TestKubeStoreStatusTransitionsToReadyAfterLiveListSuccess(t *testing.T) {
	store, err := newKubeStore(fakeKubeAPI{
		contexts: []string{"dev"},
		listsByKey: map[string][]resources.ResourceItem{
			"dev/default/workloads": {
				{Name: "api", Kind: "DEP", Status: "Healthy", Ready: "1/1"},
			},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error creating kube store: %v", err)
	}
	if status := store.Status(); status.State != StoreStateLoading {
		t.Fatalf("expected initial loading status, got %#v", status)
	}
	_, err = store.ReadModel().List("workloads", store.Scope())
	if err != nil {
		t.Fatalf("expected live list success, got %v", err)
	}
	if status := store.Status(); status.State != StoreStateReady {
		t.Fatalf("expected ready status after live list, got %#v", status)
	}
	if status := store.Status(); !strings.Contains(status.Message, "cache ready for workloads") {
		t.Fatalf("expected cache-ready message, got %#v", status)
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

func TestKubeStoreUnhealthyItemsUsesLiveListsWhenAvailable(t *testing.T) {
	store, err := newKubeStore(fakeKubeAPI{
		contexts: []string{"dev"},
		listsByKey: map[string][]resources.ResourceItem{
			"dev/default/pods": {
				{Name: "pod-ok", Status: "Running", Age: "5m"},
				{Name: "pod-bad", Status: "CrashLoop", Age: "2m"},
			},
			"dev/default/deployments": {
				{Name: "dep-bad", Status: "Degraded", Age: "10m"},
			},
			"dev/default/persistentvolumeclaims": {
				{Name: "pvc-ok", Status: "Bound", Age: "1d"},
				{Name: "pvc-pending", Status: "Pending", Age: "3m"},
			},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error creating kube store: %v", err)
	}
	got := store.UnhealthyItems()
	if len(got) != 3 {
		t.Fatalf("expected 3 unhealthy items from live lists, got %#v", got)
	}
}

func TestKubeStorePodsByRestartsUsesLiveListWhenAvailable(t *testing.T) {
	store, err := newKubeStore(fakeKubeAPI{
		contexts: []string{"dev"},
		listsByKey: map[string][]resources.ResourceItem{
			"dev/default/pods": {
				{Name: "pod-a", Restarts: "0"},
				{Name: "pod-b", Restarts: "12"},
				{Name: "pod-c", Restarts: "3"},
			},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error creating kube store: %v", err)
	}
	got := store.PodsByRestarts()
	if len(got) != 2 || got[0].Name != "pod-b" || got[1].Name != "pod-c" {
		t.Fatalf("expected live restart ordering, got %#v", got)
	}
}

func TestKubeStoreUnhealthyItemsSetsPartialOnUnsupportedLiveQuery(t *testing.T) {
	store, err := newKubeStore(fakeKubeAPI{
		contexts: []string{"dev"},
	})
	if err != nil {
		t.Fatalf("unexpected error creating kube store: %v", err)
	}
	_ = store.UnhealthyItems()
	status := store.Status()
	if status.State != StoreStatePartial {
		t.Fatalf("expected partial status on unsupported unhealthy query fallback, got %#v", status)
	}
	if !strings.Contains(status.Message, "unhealthy") {
		t.Fatalf("expected unhealthy query fallback message, got %#v", status)
	}
}

func TestKubeStorePodsByRestartsSetsStatusOnLiveQueryError(t *testing.T) {
	store, err := newKubeStore(fakeKubeAPI{
		contexts: []string{"dev"},
		listErrByKey: map[string]error{
			"dev/default/pods": errors.New("connection refused"),
		},
	})
	if err != nil {
		t.Fatalf("unexpected error creating kube store: %v", err)
	}
	_ = store.PodsByRestarts()
	status := store.Status()
	if status.State != StoreStateUnreachable {
		t.Fatalf("expected unreachable status on restarts live query error, got %#v", status)
	}
}

func TestKubeStoreStatusShowsCacheWarmingOnDirectListPath(t *testing.T) {
	store, err := newKubeStore(fakeKubeAPIWithMeta{
		fakeKubeAPI: fakeKubeAPI{
			contexts: []string{"dev"},
			listsByKey: map[string][]resources.ResourceItem{
				"dev/default/pods": {{Name: "pod-a", Status: "Running"}},
			},
		},
		cacheBacked: false,
	})
	if err != nil {
		t.Fatalf("unexpected error creating kube store: %v", err)
	}
	_, err = store.ReadModel().List("pods", store.Scope())
	if err != nil {
		t.Fatalf("expected list success, got %v", err)
	}
	status := store.Status()
	if status.State != StoreStateLoading {
		t.Fatalf("expected loading status for cache warming path, got %#v", status)
	}
	if !strings.Contains(status.Message, "warming cache for pods") {
		t.Fatalf("expected cache warming message, got %#v", status)
	}
}

func TestKubeStoreStatusTransitionsFromWarmingToReadyOnCacheBackedRead(t *testing.T) {
	api := &fakeKubeAPIWithMetaSequence{
		fakeKubeAPI: fakeKubeAPI{
			contexts: []string{"dev"},
			listsByKey: map[string][]resources.ResourceItem{
				"dev/default/pods": {{Name: "pod-a", Status: "Running"}},
			},
		},
		sequence: []bool{false, true},
	}
	store, err := newKubeStore(api)
	if err != nil {
		t.Fatalf("unexpected error creating kube store: %v", err)
	}
	_, err = store.ReadModel().List("pods", store.Scope())
	if err != nil {
		t.Fatalf("expected first list success, got %v", err)
	}
	if status := store.Status(); status.State != StoreStateLoading {
		t.Fatalf("expected loading status after direct list path, got %#v", status)
	}
	_, err = store.ReadModel().List("pods", store.Scope())
	if err != nil {
		t.Fatalf("expected second list success, got %v", err)
	}
	if status := store.Status(); status.State != StoreStateReady {
		t.Fatalf("expected ready status after cache-backed list path, got %#v", status)
	}
}
