package style

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestActionFooterIncludesScopeBindings(t *testing.T) {
	footer := ansi.Strip(ActionFooter(nil, 0))
	if !strings.Contains(footer, "X ctx") {
		t.Fatalf("expected ctx binding in action footer, got %q", footer)
	}
	if !strings.Contains(footer, "N ns") {
		t.Fatalf("expected ns binding in action footer, got %q", footer)
	}
}
