package data

import (
	"context"
	"strings"
	"testing"

	"github.com/dloss/podji/internal/resources"
)

type fallbackDetailReadModel struct {
	*MockReadModel
}

type fakeKubeAPIMeta struct {
	fakeKubeAPI
	cacheBacked bool
}

type fakeKubeAPIObjectReader struct {
	fakeKubeAPI
	detailByKey   map[string]resources.DetailData
	yamlByKey     map[string]string
	describeByKey map[string]string
}

type fakeKubeAPILogStreamer struct {
	fakeKubeAPI
	streamCalls int
	streamTail  int
	streamLines []string
}

type fakeKubeAPILogOptionsReader struct {
	fakeKubeAPI
	lastOpts LogOptions
}

func (f fakeKubeAPIMeta) ListResourcesMeta(contextName, namespace, resourceName string) ([]resources.ResourceItem, bool, error) {
	items, err := f.fakeKubeAPI.ListResources(contextName, namespace, resourceName)
	return items, f.cacheBacked, err
}

func (f fakeKubeAPIObjectReader) ResourceYAML(contextName, namespace, resourceName string, item resources.ResourceItem) (string, error) {
	key := contextName + "/" + namespace + "/" + resourceName + "/" + item.Name
	if out, ok := f.yamlByKey[key]; ok {
		return out, nil
	}
	return "", ErrObjectReadNotSupported
}

func (f fakeKubeAPIObjectReader) ResourceDetail(contextName, namespace, resourceName string, item resources.ResourceItem) (resources.DetailData, error) {
	key := contextName + "/" + namespace + "/" + resourceName + "/" + item.Name
	if out, ok := f.detailByKey[key]; ok {
		return out, nil
	}
	return resources.DetailData{}, ErrObjectReadNotSupported
}

func (f fakeKubeAPIObjectReader) ResourceDescribe(contextName, namespace, resourceName string, item resources.ResourceItem) (string, error) {
	key := contextName + "/" + namespace + "/" + resourceName + "/" + item.Name
	if out, ok := f.describeByKey[key]; ok {
		return out, nil
	}
	return "", ErrObjectReadNotSupported
}

func (f *fakeKubeAPILogStreamer) PodLogsStream(ctx context.Context, contextName, namespace, pod string, tail int, onLine func(string)) error {
	f.streamCalls++
	f.streamTail = tail
	for _, line := range f.streamLines {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		onLine(line)
	}
	return nil
}

func (f *fakeKubeAPILogOptionsReader) PodLogsWithOptions(ctx context.Context, contextName, namespace, pod string, opts LogOptions) ([]string, error) {
	f.lastOpts = opts
	return []string{"from-options-reader"}, nil
}

func (f fallbackDetailReadModel) Detail(resourceName string, item resources.ResourceItem, scope Scope) (resources.DetailData, error) {
	return resources.DetailData{
		Summary: []resources.SummaryField{{Key: "status", Value: "from-fallback"}},
	}, nil
}

func (f fallbackDetailReadModel) YAML(resourceName string, item resources.ResourceItem, scope Scope) (string, error) {
	return "from-fallback-yaml", nil
}

func (f fallbackDetailReadModel) Describe(resourceName string, item resources.ResourceItem, scope Scope) (string, error) {
	return "from-fallback-describe", nil
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

func TestKubeReadModelLogsWithContextUsesOptionsReader(t *testing.T) {
	api := &fakeKubeAPILogOptionsReader{}
	read := NewKubeReadModel(
		NewMockReadModel(resources.DefaultRegistry()),
		api,
		func() Scope { return Scope{Context: "dev", Namespace: "default"} },
		nil,
		nil,
		nil,
		nil,
	)
	lines, err := read.LogsWithContext(context.Background(), "pods", resources.ResourceItem{Name: "api-1"}, Scope{}, LogOptions{
		Tail:       123,
		Follow:     false,
		Previous:   true,
		Container:  "sidecar",
		Timestamps: true,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(lines) != 1 || lines[0] != "from-options-reader" {
		t.Fatalf("expected options reader result, got %#v", lines)
	}
	if api.lastOpts.Tail != 123 || api.lastOpts.Follow || !api.lastOpts.Previous || api.lastOpts.Container != "sidecar" || !api.lastOpts.Timestamps {
		t.Fatalf("expected options to propagate, got %#v", api.lastOpts)
	}
}

func TestKubeReadModelStreamLogsUsesAPIStreamerForPodFollow(t *testing.T) {
	api := &fakeKubeAPILogStreamer{
		streamLines: []string{"live-a", "live-b"},
	}
	read := NewKubeReadModel(
		NewMockReadModel(resources.DefaultRegistry()),
		api,
		func() Scope { return Scope{Context: "dev", Namespace: "default"} },
		nil,
		nil,
		nil,
		nil,
	)

	var got []string
	err := read.StreamLogsWithContext(context.Background(), "pods", resources.ResourceItem{Name: "api-1"}, Scope{}, LogOptions{
		Tail:   25,
		Follow: true,
	}, func(line string) {
		got = append(got, line)
	})
	if err != nil {
		t.Fatalf("expected no stream error, got %v", err)
	}
	if api.streamCalls != 1 || api.streamTail != 25 {
		t.Fatalf("expected one streamer call with tail=25, got calls=%d tail=%d", api.streamCalls, api.streamTail)
	}
	if len(got) != 2 || got[0] != "live-a" || got[1] != "live-b" {
		t.Fatalf("expected streamed lines, got %#v", got)
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

func TestKubeReadModelPrefersAPIObjectReaderForDetail(t *testing.T) {
	reg := resources.DefaultRegistry()
	read := NewKubeReadModel(
		fallbackDetailReadModel{MockReadModel: NewMockReadModel(reg)},
		fakeKubeAPIObjectReader{
			detailByKey: map[string]resources.DetailData{
				"dev/default/pods/api-1": {
					Summary: []resources.SummaryField{{Key: "status", Value: "from-object-reader"}},
				},
			},
		},
		func() Scope { return Scope{Context: "dev", Namespace: "default"} },
		nil,
		nil,
		nil,
		nil,
	)
	got, err := read.Detail("pods", resources.ResourceItem{Name: "api-1"}, Scope{})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(got.Summary) == 0 || got.Summary[0].Value != "from-object-reader" {
		t.Fatalf("expected detail from object-reader path, got %#v", got.Summary)
	}
}

func TestKubeReadModelMarksWarmingWhenListIsDirectAPIBacked(t *testing.T) {
	called := false
	read := NewKubeReadModel(
		NewMockReadModel(resources.DefaultRegistry()),
		fakeKubeAPIMeta{
			fakeKubeAPI: fakeKubeAPI{
				listsByKey: map[string][]resources.ResourceItem{
					"dev/default/pods": {{Name: "api-1"}},
				},
			},
			cacheBacked: false,
		},
		func() Scope { return Scope{Context: "dev", Namespace: "default"} },
		nil,
		nil,
		func(resourceName string) {
			if resourceName == "pods" {
				called = true
			}
		},
		nil,
	)
	_, err := read.List("pods", Scope{})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !called {
		t.Fatal("expected warming callback for direct API list path")
	}
}

func TestKubeReadModelUsesLiveYAMLForPods(t *testing.T) {
	reg := resources.DefaultRegistry()
	read := NewKubeReadModel(
		fallbackDetailReadModel{MockReadModel: NewMockReadModel(reg)},
		fakeKubeAPI{},
		func() Scope { return Scope{Context: "dev", Namespace: "default"} },
		nil,
		nil,
		nil,
		nil,
	)
	yaml, err := read.YAML("pods", resources.ResourceItem{
		Name:      "api-1",
		Namespace: "default",
		Status:    "Running",
		Ready:     "1/1",
		Labels:    map[string]string{"app": "api"},
	}, Scope{})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if yaml == "from-fallback-yaml" {
		t.Fatalf("expected live yaml renderer, got fallback output %q", yaml)
	}
	if !containsAll(yaml, "kind: Pod", "name: api-1", "phase: Running") {
		t.Fatalf("expected live yaml content, got %q", yaml)
	}
}

func TestKubeReadModelUsesLiveDescribeForPods(t *testing.T) {
	reg := resources.DefaultRegistry()
	read := NewKubeReadModel(
		fallbackDetailReadModel{MockReadModel: NewMockReadModel(reg)},
		fakeKubeAPI{},
		func() Scope { return Scope{Context: "dev", Namespace: "default"} },
		nil,
		nil,
		nil,
		nil,
	)
	describe, err := read.Describe("pods", resources.ResourceItem{
		Name:      "api-1",
		Namespace: "default",
		Status:    "Running",
		Ready:     "1/1",
		Extra: map[string]string{
			"node": "worker-1",
			"ip":   "10.0.0.1",
		},
	}, Scope{})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if describe == "from-fallback-describe" {
		t.Fatalf("expected live describe renderer, got fallback output %q", describe)
	}
	if !containsAll(describe, "Name:        api-1", "Status:      Running", "Node:        worker-1") {
		t.Fatalf("expected live describe content, got %q", describe)
	}
}

func TestKubeReadModelUsesLiveDetailForConfigMaps(t *testing.T) {
	reg := resources.DefaultRegistry()
	read := NewKubeReadModel(
		fallbackDetailReadModel{MockReadModel: NewMockReadModel(reg)},
		fakeKubeAPI{},
		func() Scope { return Scope{Context: "dev", Namespace: "default"} },
		nil,
		nil,
		nil,
		nil,
	)
	detail, err := read.Detail("configmaps", resources.ResourceItem{
		Name:   "app-config",
		Kind:   "ConfigMap",
		Status: "Available",
		Ready:  "3 keys",
		Age:    "2d",
	}, Scope{})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(detail.Summary) == 0 || detail.Summary[0].Value != "ConfigMap" {
		t.Fatalf("expected live generic detail for configmap, got %#v", detail.Summary)
	}
}

func TestKubeReadModelUsesLiveYAMLForConfigMaps(t *testing.T) {
	reg := resources.DefaultRegistry()
	read := NewKubeReadModel(
		fallbackDetailReadModel{MockReadModel: NewMockReadModel(reg)},
		fakeKubeAPI{},
		func() Scope { return Scope{Context: "dev", Namespace: "default"} },
		nil,
		nil,
		nil,
		nil,
	)
	yaml, err := read.YAML("configmaps", resources.ResourceItem{
		Name: "app-config",
		Kind: "ConfigMap",
	}, Scope{Namespace: "default"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if yaml == "from-fallback-yaml" {
		t.Fatalf("expected live yaml renderer, got fallback output %q", yaml)
	}
	if !containsAll(yaml, "kind: ConfigMap", "name: app-config") {
		t.Fatalf("expected live yaml content, got %q", yaml)
	}
}

func TestKubeReadModelPrefersAPIObjectReaderForYAML(t *testing.T) {
	reg := resources.DefaultRegistry()
	read := NewKubeReadModel(
		fallbackDetailReadModel{MockReadModel: NewMockReadModel(reg)},
		fakeKubeAPIObjectReader{
			yamlByKey: map[string]string{
				"dev/default/pods/api-1": "apiVersion: v1\nkind: Pod\nmetadata:\n  name: api-1\n",
			},
		},
		func() Scope { return Scope{Context: "dev", Namespace: "default"} },
		nil,
		nil,
		nil,
		nil,
	)
	got, err := read.YAML("pods", resources.ResourceItem{Name: "api-1"}, Scope{})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !containsAll(got, "kind: Pod", "name: api-1") {
		t.Fatalf("expected object-reader yaml, got %q", got)
	}
}

func TestKubeReadModelUsesLiveDescribeForConfigMaps(t *testing.T) {
	reg := resources.DefaultRegistry()
	read := NewKubeReadModel(
		fallbackDetailReadModel{MockReadModel: NewMockReadModel(reg)},
		fakeKubeAPI{},
		func() Scope { return Scope{Context: "dev", Namespace: "default"} },
		nil,
		nil,
		nil,
		nil,
	)
	describe, err := read.Describe("configmaps", resources.ResourceItem{
		Name:   "app-config",
		Kind:   "ConfigMap",
		Status: "Available",
	}, Scope{Namespace: "default"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if describe == "from-fallback-describe" {
		t.Fatalf("expected live describe renderer, got fallback output %q", describe)
	}
	if !containsAll(describe, "Kind:        ConfigMap", "Name:        app-config") {
		t.Fatalf("expected live describe content, got %q", describe)
	}
}

func TestKubeReadModelPrefersAPIObjectReaderForDescribe(t *testing.T) {
	reg := resources.DefaultRegistry()
	read := NewKubeReadModel(
		fallbackDetailReadModel{MockReadModel: NewMockReadModel(reg)},
		fakeKubeAPIObjectReader{
			describeByKey: map[string]string{
				"dev/default/services/api": "Name:        api\nKind:        Service\n",
			},
		},
		func() Scope { return Scope{Context: "dev", Namespace: "default"} },
		nil,
		nil,
		nil,
		nil,
	)
	got, err := read.Describe("services", resources.ResourceItem{Name: "api"}, Scope{})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !containsAll(got, "Name:        api", "Kind:        Service") {
		t.Fatalf("expected object-reader describe, got %q", got)
	}
}

func containsAll(text string, parts ...string) bool {
	for _, p := range parts {
		if !strings.Contains(text, p) {
			return false
		}
	}
	return true
}
