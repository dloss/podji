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

func TestWorkloadsScenarioCycleAndBanner(t *testing.T) {
	w := NewWorkloads()

	w.CycleScenario() // empty
	if w.Scenario() != "empty" {
		t.Fatalf("expected empty scenario, got %q", w.Scenario())
	}

	w.CycleScenario() // forbidden
	if w.Scenario() != "forbidden" {
		t.Fatalf("expected forbidden scenario, got %q", w.Scenario())
	}
	if !strings.Contains(w.Banner(), "Access denied") {
		t.Fatalf("expected access denied banner, got %q", w.Banner())
	}
}

func TestCronJobPodsNameAndEmptyState(t *testing.T) {
	pods := NewWorkloadPods(ResourceItem{Name: "sync-reports", Kind: "CJ"}, nil)

	if got := pods.Name(); got != "pods (sync-reports)" {
		t.Fatalf("expected concise pods name, got %q", got)
	}

	msg := pods.EmptyMessage(false, "")
	if msg != "No pods found for workload `sync-reports`." {
		t.Fatalf("expected simplified empty-state message, got %q", msg)
	}
}
