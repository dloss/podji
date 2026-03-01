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
	// Suspended is intentional: it should sort after Pending but before Healthy.
	lastSuspendedIdx := -1
	firstHealthyIdx := -1
	for i, it := range items {
		if it.Status == "Suspended" {
			lastSuspendedIdx = i
		}
		if it.Status == "Healthy" && firstHealthyIdx == -1 {
			firstHealthyIdx = i
		}
	}
	if lastSuspendedIdx == -1 {
		t.Fatal("expected at least one Suspended item")
	}
	if firstHealthyIdx == -1 {
		t.Fatal("expected at least one Healthy item")
	}
	if firstHealthyIdx < lastSuspendedIdx {
		t.Fatalf("Healthy items should appear after Suspended in problem-first sort (firstHealthyIdx=%d, lastSuspendedIdx=%d)", firstHealthyIdx, lastSuspendedIdx)
	}
}

func TestWorkloadsSortByStatusDesc(t *testing.T) {
	w := NewWorkloads()
	w.SetSort("status", true) // reversed: healthy-first
	items := w.Items()
	if len(items) == 0 {
		t.Fatalf("expected mock workloads")
	}

	if items[0].Status != "Healthy" {
		t.Fatalf("expected first status Healthy (reversed), got %q", items[0].Status)
	}
	// Suspended should appear after all Healthy items but before Pending/Degraded/Failed.
	lastHealthyIdx := -1
	firstSuspendedIdx := -1
	for i, it := range items {
		if it.Status == "Healthy" {
			lastHealthyIdx = i
		}
		if it.Status == "Suspended" && firstSuspendedIdx == -1 {
			firstSuspendedIdx = i
		}
	}
	if firstSuspendedIdx == -1 {
		t.Fatal("expected at least one Suspended item")
	}
	if firstSuspendedIdx < lastHealthyIdx {
		t.Fatalf("Suspended items should appear after Healthy in reversed sort (firstSuspendedIdx=%d, lastHealthyIdx=%d)", firstSuspendedIdx, lastHealthyIdx)
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
	t.Setenv("PODJI_SCENARIO", "forbidden")
	w := NewWorkloads()
	if !strings.Contains(w.Banner(), "Access denied") {
		t.Fatalf("expected access denied banner, got %q", w.Banner())
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
