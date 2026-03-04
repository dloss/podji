package style

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestActionFooterIncludesScopeBindings(t *testing.T) {
	footer := ansi.Strip(ActionFooter(nil, 0))
	if !strings.Contains(footer, "X context") {
		t.Fatalf("expected context binding in action footer, got %q", footer)
	}
	if !strings.Contains(footer, "N namespace") {
		t.Fatalf("expected namespace binding in action footer, got %q", footer)
	}
}

