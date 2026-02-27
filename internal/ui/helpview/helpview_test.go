package helpview

import (
	"strings"
	"testing"
)

func TestHelpTextReflectsCurrentNavigation(t *testing.T) {
	if strings.Contains(helpText, "tab / shift+tab") {
		t.Fatalf("help text still references old tab column cycling")
	}
	if !strings.Contains(helpText, "c                    Copy mode") {
		t.Fatalf("help text missing copy mode entry")
	}
	if !strings.Contains(helpText, "RESOURCE BROWSER (A)") {
		t.Fatalf("help text missing resource browser section")
	}
}
