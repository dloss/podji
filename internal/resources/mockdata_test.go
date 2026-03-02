package resources

import "testing"

func TestMockScenarioPrefersExplicitMockScenario(t *testing.T) {
	t.Setenv("PODJI_SCENARIO", "forbidden")
	t.Setenv("PODJI_MOCK_SCENARIO", "empty")
	if got := mockScenario(); got != "empty" {
		t.Fatalf("expected PODJI_MOCK_SCENARIO to win, got %q", got)
	}
}

func TestMockScenarioFallsBackToLegacyScenario(t *testing.T) {
	t.Setenv("PODJI_SCENARIO", "partial")
	t.Setenv("PODJI_MOCK_SCENARIO", "")
	if got := mockScenario(); got != "partial" {
		t.Fatalf("expected fallback to PODJI_SCENARIO, got %q", got)
	}
}
