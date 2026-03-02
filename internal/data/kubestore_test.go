package data

import (
	"errors"
	"strings"
	"testing"

	"github.com/dloss/podji/internal/resources"
)

type fakeRunner struct {
	out map[string]string
	err map[string]error
}

func (r fakeRunner) Run(name string, args ...string) (string, error) {
	key := strings.Join(append([]string{name}, args...), " ")
	if err := r.err[key]; err != nil {
		return "", err
	}
	return r.out[key], nil
}

func TestNewKubeStoreUsesFirstSortedContext(t *testing.T) {
	store, err := newKubeStore(fakeRunner{
		out: map[string]string{
			"kubectl config get-contexts -o name": "prod\nstaging\ndev\n",
		},
	})
	if err != nil {
		t.Fatalf("expected kube store creation to succeed, got %v", err)
	}
	if got := store.Scope().Context; got != "dev" {
		t.Fatalf("expected first sorted context dev, got %q", got)
	}
}

func TestKubeStoreNamespaceNamesFallbackOnError(t *testing.T) {
	store, err := newKubeStore(fakeRunner{
		out: map[string]string{
			"kubectl config get-contexts -o name": "dev\n",
		},
		err: map[string]error{
			"kubectl --context dev get namespaces -o jsonpath={range .items[*]}{.metadata.name}{\"\\n\"}{end}": errors.New("boom"),
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
	store, err := newKubeStore(fakeRunner{
		out: map[string]string{
			"kubectl config get-contexts -o name": "dev\nprod\n",
			"kubectl --context dev get namespaces -o jsonpath={range .items[*]}{.metadata.name}{\"\\n\"}{end}": "kube-system\ndefault\n",
		},
	})
	if err != nil {
		t.Fatalf("unexpected error creating kube store: %v", err)
	}
	got := store.NamespaceNames()
	want := []string{resources.AllNamespaces, "default", "kube-system"}
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
	store, err := newKubeStore(fakeRunner{
		out: map[string]string{
			"kubectl config get-contexts -o name": "dev\n",
		},
	})
	if err != nil {
		t.Fatalf("unexpected error creating kube store: %v", err)
	}
	store.SetScope(Scope{Context: "dev", Namespace: "staging"})
	if got := store.Registry().Namespace(); got != "staging" {
		t.Fatalf("expected registry namespace staging, got %q", got)
	}
}

func TestKubeStorePodLogsFetcherWired(t *testing.T) {
	runner := fakeRunner{
		out: map[string]string{
			"kubectl config get-contexts -o name":                  "dev\n",
			"kubectl --context dev -n default logs api --tail=200": "line-a\nline-b\n",
		},
	}
	store, err := newKubeStore(runner)
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
	runner := fakeRunner{
		out: map[string]string{
			"kubectl config get-contexts -o name": "dev\n",
			`kubectl --context dev -n default get events --field-selector involvedObject.name=api -o jsonpath={range .items[*]}{.lastTimestamp}{"   "}{.type}{"   "}{.reason}{"   "}{.message}{"\n"}{end}`: "2026-03-01T12:00:00Z   Warning   BackOff   Back-off restarting failed container\n",
		},
	}
	store, err := newKubeStore(runner)
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
