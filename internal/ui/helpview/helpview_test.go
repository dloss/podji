package helpview

import (
	"strings"
	"testing"
)

func TestHelpTextReflectsCurrentNavigation(t *testing.T) {
	if strings.Contains(helpText, "tab / shift+tab") {
		t.Fatalf("help text still references old tab column cycling")
	}
	if strings.Contains(helpText, "  tab ") {
		t.Fatalf("help text still references tab key")
	}
	if !strings.Contains(helpText, "c                    Copy mode") {
		t.Fatalf("help text missing copy mode entry")
	}
	if !strings.Contains(helpText, "GLOBAL (app navigation)") {
		t.Fatalf("help text missing updated global section heading")
	}
	if !strings.Contains(helpText, "TABLE (filterable lists, including A)") {
		t.Fatalf("help text missing updated table section heading")
	}
	if !strings.Contains(helpText, ":                    Command bar (from lists)") {
		t.Fatalf("help text missing command bar entry")
	}
}
