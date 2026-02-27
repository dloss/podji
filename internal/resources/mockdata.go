package resources

import "fmt"

// expandMockItems pads a stub item list to a target length by cloning entries
// with deterministic "zz-" names so existing sort expectations stay stable.
func expandMockItems(items []ResourceItem, target int) []ResourceItem {
	if len(items) == 0 || len(items) >= target {
		return items
	}

	out := make([]ResourceItem, len(items), target)
	copy(out, items)

	cloneRound := 1
	for len(out) < target {
		for i := 0; i < len(items) && len(out) < target; i++ {
			base := items[i]
			clone := base
			clone.Name = fmt.Sprintf("zz-%s-%02d", base.Name, cloneRound)
			out = append(out, clone)
		}
		cloneRound++
	}
	return out
}

// expandMockLogs appends synthetic lines so log views have enough depth to
// exercise paging, search, and scrolling interactions.
func expandMockLogs(lines []string, extra int) []string {
	if len(lines) == 0 || extra <= 0 {
		return lines
	}

	out := make([]string, 0, len(lines)+extra)
	out = append(out, lines...)
	for i := 1; i <= extra; i++ {
		out = append(out, fmt.Sprintf("2026-02-20T15:%02d:%02dZ  trace=mock-%03d  heartbeat ok",
			(i/60)%60, i%60, i))
	}
	return out
}
