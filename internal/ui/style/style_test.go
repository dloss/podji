package style

import "testing"

func TestStatusStylesBySeverity(t *testing.T) {
	if got := classifyStatus("CrashLoopBackOff"); got != statusError {
		t.Fatalf("expected error status, got %v", got)
	}

	if got := classifyStatus("Unknown"); got != statusWarning {
		t.Fatalf("expected warning status, got %v", got)
	}

	if got := classifyStatus("Suspended"); got != statusNeutral {
		t.Fatalf("expected neutral status, got %v", got)
	}

	if got := classifyStatus("Running"); got != statusHealthy {
		t.Fatalf("expected healthy status, got %v", got)
	}
}
