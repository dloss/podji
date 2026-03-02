package detailview

import (
	"strings"
	"testing"

	"github.com/dloss/podji/internal/resources"
)

func TestUseTwoColumnLayout(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		width  int
		detail resources.DetailData
		want   bool
	}{
		{
			name:   "narrow view stays single column",
			width:  100,
			detail: resources.DetailData{Containers: []resources.ContainerRow{{Name: "api"}}},
			want:   false,
		},
		{
			name:   "wide with containers uses two column",
			width:  140,
			detail: resources.DetailData{Containers: []resources.ContainerRow{{Name: "api"}}},
			want:   true,
		},
		{
			name:   "wide with conditions uses two column",
			width:  140,
			detail: resources.DetailData{Conditions: []string{"Ready=True"}},
			want:   true,
		},
		{
			name:  "wide labels and events only stays single column",
			width: 140,
			detail: resources.DetailData{
				Labels: []string{"app=api"},
				Events: []string{"—   No recent events"},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := useTwoColumnLayout(tt.width, tt.detail)
			if got != tt.want {
				t.Fatalf("useTwoColumnLayout(%d, detail) = %v, want %v", tt.width, got, tt.want)
			}
		})
	}
}

func TestRenderSummary(t *testing.T) {
	t.Parallel()

	line := renderSummary([]resources.SummaryField{
		{Key: "status", Label: "Status", Value: "Healthy"},
		{Key: "age", Label: "Age", Value: "14d"},
		{Key: "related", Label: "Related", Value: "2", Tone: resources.SummaryToneNeutral},
	})

	for _, want := range []string{"Status:", "Healthy", "Age:", "14d", "Related:", "2"} {
		if !strings.Contains(line, want) {
			t.Fatalf("renderSummary(...) missing %q in %q", want, line)
		}
	}
}
