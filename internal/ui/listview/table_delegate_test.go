package listview

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
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

	listItems := makeListItems(items, rows, widths, columns)
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
