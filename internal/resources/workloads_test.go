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

func TestWorkloadsToggleSortByProblem(t *testing.T) {
	w := NewWorkloads()
	w.ToggleSort()
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
	pods := NewWorkloadPods(ResourceItem{Name: "sync-reports", Kind: "CJ"})

	if got := pods.Name(); got != "pods (sync-reports)" {
		t.Fatalf("expected concise pods name, got %q", got)
	}

	msg := pods.EmptyMessage(false, "")
	if !strings.Contains(msg, "press r") {
		t.Fatalf("expected related hint, got %q", msg)
	}
}
