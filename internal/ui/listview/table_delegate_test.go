package listview

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/dloss/podji/internal/resources"
)

func TestMakeListItemsUsesNameColumnForMatching(t *testing.T) {
	items := []resources.ResourceItem{{Name: "api"}}
	rows := [][]string{{"default", "api"}}
	widths := []int{10, 10}
	columns := []resources.TableColumn{
		{ID: "namespace", Name: "NAMESPACE"},
		{ID: "name", Name: "NAME"},
	}

	listItems := makeListItems(resources.NewPods(), items, rows, widths, columns)
	got, ok := listItems[0].(item)
	if !ok {
		t.Fatalf("expected list item type item, got %T", listItems[0])
	}
	if got.matchColumn != 1 {
		t.Fatalf("expected matchColumn=1 for NAME, got %d", got.matchColumn)
	}
}

func TestRenderRowWithNameMatchUnderlinesNameColumn(t *testing.T) {
	it := item{
		data:        resources.ResourceItem{Name: "api"},
		row:         []string{"default", "api"},
		widths:      []int{10, 10},
		matchColumn: 1,
	}

	row := renderRowWithNameMatch(it, false, nil, lipgloss.NewStyle(), lipgloss.NewStyle(), true)
	if !strings.Contains(row, "\x1b[4;1;97ma") {
		t.Fatalf("expected find underline on name column, got %q", row)
	}
	if strings.Contains(row, "\x1b[4;1;97md") {
		t.Fatalf("expected namespace column to remain unmarked, got %q", row)
	}
}

func TestPodGeneratedSuffixRange(t *testing.T) {
	tests := []struct {
		name  string
		want  string
		valid bool
	}{
		{name: "zz-api-7c6c8d5f7d-x8p2k-01", want: "-7c6c8d5f7d-x8p2k", valid: true},
		{name: "api-7c6c8d5f7d-x8p2k", want: "-7c6c8d5f7d-x8p2k", valid: true},
		{name: "db-0", valid: false},
		{name: "api-canary", valid: false},
	}

	for _, tc := range tests {
		start, end, ok := podGeneratedSuffixRange(tc.name)
		if ok != tc.valid {
			t.Fatalf("podGeneratedSuffixRange(%q) valid=%v, want %v", tc.name, ok, tc.valid)
		}
		if !tc.valid {
			continue
		}
		got := string([]rune(tc.name)[start:end])
		if got != tc.want {
			t.Fatalf("podGeneratedSuffixRange(%q)=%q, want %q", tc.name, got, tc.want)
		}
	}
}

func TestRenderRowDimsGeneratedPodSuffixButKeepsOrdinalTail(t *testing.T) {
	it := item{
		data:       resources.ResourceItem{Name: "zz-api-7c6c8d5f7d-x8p2k-01"},
		row:        []string{"zz-api-7c6c8d5f7d-x8p2k-01"},
		widths:     []int{40},
		dimPodName: true,
	}

	row := renderRowWithNameMatch(it, false, nil, lipgloss.NewStyle(), lipgloss.NewStyle(), false)
	if !strings.Contains(row, "\x1b[38;5;247m-7c6c8d5f7d-x8p2k\x1b[39m") {
		t.Fatalf("expected dimmed generated suffix, got %q", row)
	}
	if !strings.Contains(ansi.Strip(row), "zz-api-7c6c8d5f7d-x8p2k-01") {
		t.Fatalf("expected full visible name, got %q", ansi.Strip(row))
	}
	if strings.Contains(row, "\x1b[38;5;241m-01") {
		t.Fatalf("expected ordinal tail to remain undimmed, got %q", row)
	}
}
