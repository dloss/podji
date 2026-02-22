package filterbar

import (
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/list"
)

func TestAppendPreservesInteriorBlankLines(t *testing.T) {
	model := filteringModel("zzz")
	view := "header\n\nNo items found"

	rendered := Append(view, model)
	lines := strings.Split(rendered, "\n")
	if len(lines) == 0 {
		t.Fatal("expected rendered output")
	}

	last := strings.TrimSpace(lines[len(lines)-1])
	if !strings.HasPrefix(last, "/") {
		t.Fatalf("expected filter bar on last line, got %q", lines[len(lines)-1])
	}

	if !strings.Contains(rendered, "\n\nNo items found\n") {
		t.Fatalf("expected interior blank line and message to remain, got %q", rendered)
	}
}

func TestAppendReplacesTrailingPaddingLine(t *testing.T) {
	model := filteringModel("zzz")
	view := "header\nrow\n"

	rendered := Append(view, model)
	lines := strings.Split(rendered, "\n")
	if len(lines) < 2 {
		t.Fatalf("expected at least two lines, got %q", rendered)
	}

	last := strings.TrimSpace(lines[len(lines)-1])
	if !strings.HasPrefix(last, "/") {
		t.Fatalf("expected filter bar on last line, got %q", lines[len(lines)-1])
	}
	if strings.Contains(rendered, "\n\n/") {
		t.Fatalf("expected trailing padding to be replaced, got %q", rendered)
	}
}

func filteringModel(value string) list.Model {
	model := list.New(nil, list.NewDefaultDelegate(), 80, 20)
	Setup(&model)
	model.SetFilteringEnabled(true)
	model.SetFilterState(list.Filtering)
	model.FilterInput.SetValue(value)
	return model
}
