package logview

import "testing"

func TestWrapLine(t *testing.T) {
	tests := []struct {
		name  string
		line  string
		width int
		want  []string
	}{
		{name: "empty", line: "", width: 4, want: []string{""}},
		{name: "short", line: "abc", width: 4, want: []string{"abc"}},
		{name: "exact", line: "abcd", width: 4, want: []string{"abcd"}},
		{name: "long", line: "abcdefghij", width: 4, want: []string{"abcd", "efgh", "ij"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := wrapLine(tt.line, tt.width)
			if len(got) != len(tt.want) {
				t.Fatalf("len(got)=%d, want %d", len(got), len(tt.want))
			}
			for i := range tt.want {
				if got[i] != tt.want[i] {
					t.Fatalf("got[%d]=%q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestWrapLines(t *testing.T) {
	got := wrapLines([]string{"abcd", "123456"}, 4)
	want := "abcd\n1234\n56"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}
