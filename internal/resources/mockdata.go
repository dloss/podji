package resources

import (
	"fmt"
	"hash/fnv"
	"math/rand"
	"strings"
	"time"
)

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

// expandMockLogs appends deterministic, flog-style synthetic lines so log
// views have enough depth to exercise paging, search, and scrolling.
func expandMockLogs(lines []string, extra int) []string {
	if len(lines) == 0 || extra <= 0 {
		return lines
	}

	out := make([]string, 0, len(lines)+extra)
	out = append(out, lines...)
	out = append(out, mockFlogLines(lines, extra)...)
	return out
}

func mockFlogLines(seedLines []string, n int) []string {
	seedHasher := fnv.New64a()
	_, _ = seedHasher.Write([]byte(strings.Join(seedLines, "|")))
	rng := rand.New(rand.NewSource(int64(seedHasher.Sum64())))

	ips := []string{"10.244.1.35", "10.244.2.15", "10.0.1.11", "10.0.1.13", "172.16.4.21"}
	methods := []string{"GET", "POST", "PUT", "PATCH", "DELETE"}
	paths := []string{"/healthz", "/readyz", "/api/v1/orders", "/api/v1/users/42", "/metrics", "/auth/login"}
	agents := []string{"kube-probe/1.30", "curl/8.6.0", "Go-http-client/2.0", "okhttp/4.12.0", "prometheus/2.53.1"}
	levels := []string{"INFO", "WARN", "ERROR"}
	services := []string{"api", "worker", "gateway", "scheduler", "sidecar"}
	statuses := []int{200, 201, 204, 400, 401, 403, 404, 429, 500, 502}
	base := time.Date(2026, 2, 20, 15, 0, 0, 0, time.UTC)
	format := rng.Intn(4)

	out := make([]string, 0, n)
	for i := 0; i < n; i++ {
		ts := base.Add(time.Duration(i*2) * time.Second)
		if i > 0 && i%37 == 0 {
			// Occasional unstructured output mixed into otherwise consistent logs.
			out = append(out, fmt.Sprintf("%s panic: runtime error: invalid memory address or nil pointer dereference", ts.Format(time.RFC3339)))
			out = append(out, "goroutine 1 [running]:")
			out = append(out, "main.(*server).handleRequest(0x0, 0xc00019e1c0)")
			out = append(out, "\t/workspace/server.go:184 +0x1a2")
			continue
		}

		switch format {
		case 0:
			out = append(out, fmt.Sprintf(
				`%s - - [%s] "%s %s HTTP/1.1" %d %d`,
				ips[rng.Intn(len(ips))],
				ts.Format("02/Jan/2006:15:04:05 -0700"),
				methods[rng.Intn(len(methods))],
				paths[rng.Intn(len(paths))],
				statuses[rng.Intn(len(statuses))],
				200+rng.Intn(3800),
			))
		case 1:
			out = append(out, fmt.Sprintf(
				`%s [%s] [%s] service=%s trace=%08x msg="%s %s"`,
				ts.Format(time.RFC3339),
				levels[rng.Intn(len(levels))],
				ips[rng.Intn(len(ips))],
				services[rng.Intn(len(services))],
				rng.Uint32(),
				methods[rng.Intn(len(methods))],
				paths[rng.Intn(len(paths))],
			))
		case 2:
			out = append(out, fmt.Sprintf(
				`<%d>1 %s node-%02d podji %d - [meta trace=%08x userAgent="%s"] request completed`,
				10+rng.Intn(20),
				ts.Format(time.RFC3339),
				1+rng.Intn(9),
				1000+rng.Intn(9000),
				rng.Uint32(),
				agents[rng.Intn(len(agents))],
			))
		default:
			out = append(out, fmt.Sprintf(
				`{"time":"%s","level":"%s","service":"%s","path":"%s","status":%d,"latency_ms":%d}`,
				ts.Format(time.RFC3339Nano),
				strings.ToLower(levels[rng.Intn(len(levels))]),
				services[rng.Intn(len(services))],
				paths[rng.Intn(len(paths))],
				statuses[rng.Intn(len(statuses))],
				2+rng.Intn(1800),
			))
		}
	}
	return out
}
