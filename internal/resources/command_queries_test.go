package resources

import "testing"

func TestParseRestartCount(t *testing.T) {
	tests := []struct {
		raw  string
		want int
	}{
		{raw: "", want: 0},
		{raw: "-", want: 0},
		{raw: "0", want: 0},
		{raw: "3", want: 3},
		{raw: "12 (10m ago)", want: 12},
	}
	for _, tt := range tests {
		if got := parseRestartCount(tt.raw); got != tt.want {
			t.Fatalf("parseRestartCount(%q)=%d want %d", tt.raw, got, tt.want)
		}
	}
}

func TestMatchesLabelSelector(t *testing.T) {
	item := ResourceItem{
		Name: "api",
		Labels: map[string]string{
			"app": "api",
			"env": "prod",
		},
	}

	if !MatchesLabelSelector(item, "app=api") {
		t.Fatal("expected single label selector to match")
	}
	if !MatchesLabelSelector(item, "app=api,env=prod") {
		t.Fatal("expected multi-label selector to match")
	}
	if MatchesLabelSelector(item, "app=worker") {
		t.Fatal("expected mismatched label selector to fail")
	}
	if MatchesLabelSelector(item, "app") {
		t.Fatal("expected malformed selector to fail")
	}
}

func TestPodsByRestartsSortedAndPositiveOnly(t *testing.T) {
	items := PodsByRestarts(DefaultNamespace)
	if len(items) == 0 {
		t.Fatal("expected at least one restarted pod in stub data")
	}
	prev := parseRestartCount(items[0].Restarts)
	if prev <= 0 {
		t.Fatalf("expected first restart count > 0, got %d", prev)
	}
	for i := 1; i < len(items); i++ {
		cur := parseRestartCount(items[i].Restarts)
		if cur <= 0 {
			t.Fatalf("expected only pods with restarts > 0, got %d at index %d", cur, i)
		}
		if cur > prev {
			t.Fatalf("expected descending restart sort, index %d has %d after %d", i, cur, prev)
		}
		prev = cur
	}
}

func TestUnhealthyItemsExcludeHealthyStatuses(t *testing.T) {
	items := UnhealthyItems(DefaultNamespace)
	if len(items) == 0 {
		t.Fatal("expected unhealthy items in stub data")
	}
	for _, it := range items {
		switch it.Status {
		case "", "Healthy", "Running", "Bound":
			t.Fatalf("unexpected healthy item in unhealthy query: %#v", it)
		}
		if it.Kind == "" {
			t.Fatalf("expected kind to be set for unhealthy item: %#v", it)
		}
	}
}
