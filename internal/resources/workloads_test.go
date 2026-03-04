package resources

import (
	"strings"
	"testing"
)

func TestWorkloadsDefaultNameSort(t *testing.T) {
	w := NewWorkloads()
	items := w.Items()

	if len(items) < 3 {
		t.Fatalf("expected at least 3 workloads")
	}

	if items[0].Name != "api" {
		t.Fatalf("expected name sort starting at api, got %q", items[0].Name)
	}
	if items[1].Name != "cleanup-tmp" {
		t.Fatalf("expected cleanup-tmp second, got %q", items[1].Name)
	}
}

func TestWorkloadsSortByStatus(t *testing.T) {
	w := NewWorkloads()
	w.SetSort("status", false) // problem-first
	items := w.Items()
	if len(items) == 0 {
		t.Fatalf("expected mock workloads")
	}

	if items[0].Status != "Failed" {
		t.Fatalf("expected first status Failed, got %q", items[0].Status)
	}
	if items[len(items)-1].Status != "Suspended" {
		t.Fatalf("expected last status Suspended, got %q", items[len(items)-1].Status)
	}
}

func TestWorkloadsSortByStatusDesc(t *testing.T) {
	w := NewWorkloads()
	w.SetSort("status", true) // reversed: healthy-first
	items := w.Items()
	if len(items) == 0 {
		t.Fatalf("expected mock workloads")
	}

	if items[0].Status != "Suspended" {
		t.Fatalf("expected first status Suspended (reversed), got %q", items[0].Status)
	}
}

func TestWorkloadsSortByName(t *testing.T) {
	w := NewWorkloads()
	w.SetSort("name", true) // Z→A
	items := w.Items()
	if len(items) < 2 {
		t.Fatalf("expected at least 2 workloads")
	}
	if items[0].Name < items[1].Name {
		t.Fatalf("expected descending name sort, got %q before %q", items[0].Name, items[1].Name)
	}
}

func TestWorkloadsScenarioBanner(t *testing.T) {
	t.Setenv("PODJI_MOCK_SCENARIO", "forbidden")
	w := NewWorkloads()
	if !strings.Contains(w.Banner(), "Access denied") {
		t.Fatalf("expected access denied banner, got %q", w.Banner())
	}
}

func TestWorkloadsEmptyMessageGuidesNamespaceSwitch(t *testing.T) {
	w := NewWorkloads()
	msg := w.EmptyMessage(false, "")
	if !strings.Contains(msg, "Press N to switch namespace") {
		t.Fatalf("expected namespace switch hint, got %q", msg)
	}
}

func TestCronJobPodsNameAndEmptyState(t *testing.T) {
	pods := NewWorkloadPods(ResourceItem{Name: "sync-reports", Kind: "CJ"}, nil)

	if got := pods.Name(); got != "pods (CronJob: sync-reports, newest job: —)" {
		t.Fatalf("expected CronJob pods name with job context, got %q", got)
	}

	msg := pods.EmptyMessage(false, "")
	if msg != "No jobs have run for CronJob `sync-reports` yet." {
		t.Fatalf("expected CronJob-specific empty-state message, got %q", msg)
	}
}

func TestWorkloadPodsShowsNamespaceColumnInAllNamespacesMode(t *testing.T) {
	reg := DefaultRegistry()
	reg.SetNamespace(AllNamespaces)
	pods := NewWorkloadPods(ResourceItem{
		Name:     "api",
		Kind:     "DEP",
		Selector: map[string]string{"app": "api"},
	}, reg)
	pods.SetNamespace(AllNamespaces)

	cols := pods.TableColumns()
	if len(cols) == 0 || cols[0].ID != "namespace" {
		t.Fatalf("expected namespace column first in all-namespaces mode, got %#v", cols)
	}

	items := pods.Items()
	if len(items) == 0 {
		t.Fatal("expected workload pods")
	}
	row := pods.TableRow(items[0])
	if row["namespace"] == "" {
		t.Fatalf("expected namespace value in row, got %#v", row)
	}
}

func TestWorkloadPodsLiveWorkloadSkipsMockFallback(t *testing.T) {
	reg := DefaultRegistry()
	reg.SetNamespace(AllNamespaces)
	pods := NewWorkloadPods(ResourceItem{
		UID:       "live-uid-1",
		Name:      "coredns",
		Namespace: "kube-system",
		Kind:      "DEP",
		Selector:  map[string]string{"app": "coredns"},
	}, reg)
	pods.SetNamespace(AllNamespaces)

	items := pods.Items()
	if len(items) != 0 {
		t.Fatalf("expected no synthetic pods for live workload, got %#v", items)
	}
}

func TestWorkloadPodsMockFallbackUsesWorkloadNamespace(t *testing.T) {
	reg := DefaultRegistry()
	reg.SetNamespace(AllNamespaces)
	pods := NewWorkloadPods(ResourceItem{
		Name:      "coredns",
		Namespace: "kube-system",
		Kind:      "DEP",
		Selector:  map[string]string{"app": "coredns"},
	}, reg)
	pods.SetNamespace(AllNamespaces)

	items := pods.Items()
	if len(items) == 0 {
		t.Fatal("expected fallback mock pods")
	}
	for _, item := range items {
		if item.Namespace != "kube-system" {
			t.Fatalf("expected fallback namespace kube-system, got %#v", items)
		}
	}
}
