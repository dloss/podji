package filterbar

import (
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/list"
)

func TestFilterInputView(t *testing.T) {
	model := filteringModel("zzz")
	
	filterInput := FilterInputView(model)
	if filterInput == "" {
		t.Fatal("expected filter input view, got empty string")
	}
	
	if !strings.HasPrefix(filterInput, "/") {
		t.Fatalf("expected filter input to start with '/', got %q", filterInput)
	}
	
	if !strings.Contains(filterInput, "zzz") {
		t.Fatalf("expected filter input to contain 'zzz', got %q", filterInput)
	}
}

func TestFilterInputViewNotFiltering(t *testing.T) {
	model := list.New(nil, list.NewDefaultDelegate(), 80, 20)
	Setup(&model)
	model.SetFilteringEnabled(true)
	
	filterInput := FilterInputView(model)
	if filterInput != "" {
		t.Fatalf("expected empty string when not filtering, got %q", filterInput)
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
