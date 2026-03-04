package resources

import "testing"

func TestMockScenarioPrefersExplicitMockScenario(t *testing.T) {
	t.Setenv("PODJI_MOCK_SCENARIO", "empty")
	if got := mockScenario(); got != "empty" {
		t.Fatalf("expected PODJI_MOCK_SCENARIO, got %q", got)
	}
}

func TestMockScenarioIgnoresLegacyScenario(t *testing.T) {
	t.Setenv("PODJI_SCENARIO", "partial")
	t.Setenv("PODJI_MOCK_SCENARIO", "")
	if got := mockScenario(); got != "" {
		t.Fatalf("expected empty scenario when PODJI_MOCK_SCENARIO is unset, got %q", got)
	}
}
