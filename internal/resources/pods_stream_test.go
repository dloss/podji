package resources

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestPodsLogsStreamFollowFalseReturnsSnapshot(t *testing.T) {
	pods := NewPods()
	item := ResourceItem{Name: "api"}
	opts := LogOptions{Follow: false, Timestamps: true}

	want, err := pods.LogsWithOptions(context.Background(), item, opts)
	if err != nil {
		t.Fatalf("LogsWithOptions() error = %v", err)
	}

	var got []string
	err = pods.LogsStream(context.Background(), item, opts, func(line string) {
		got = append(got, line)
	})
	if err != nil {
		t.Fatalf("LogsStream() error = %v", err)
	}
	if len(got) != len(want) {
		t.Fatalf("expected snapshot length %d, got %d", len(want), len(got))
	}
}

func TestPodsLogsStreamFollowAppendsUntilContextDone(t *testing.T) {
	pods := NewPods()
	item := ResourceItem{Name: "api"}
	opts := LogOptions{Follow: true, Timestamps: true}

	ctx, cancel := context.WithTimeout(context.Background(), 900*time.Millisecond)
	defer cancel()

	var got []string
	err := pods.LogsStream(ctx, item, opts, func(line string) {
		got = append(got, line)
	})
	if !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context cancellation, got %v", err)
	}
	if len(got) < 1 {
		t.Fatalf("expected at least one follow line, got %d", len(got))
	}
	if !strings.Contains(got[0], "T") {
		t.Fatalf("expected RFC3339 timestamp prefix in streamed line, got %q", got[0])
	}
}
