package app

import (
	"strings"
	"testing"
)

func TestNewUnknownModeSetsStatusMessage(t *testing.T) {
	t.Setenv("PODJI_MODE", "unknown")
	m := New()
	if !strings.Contains(m.statusMsg, "unknown PODJI_MODE") {
		t.Fatalf("expected unknown mode status message, got %q", m.statusMsg)
	}
}
