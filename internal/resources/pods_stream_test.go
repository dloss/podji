package resources

import (
	"context"
	"errors"
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

	initial, err := pods.LogsWithOptions(context.Background(), item, opts)
	if err != nil {
		t.Fatalf("LogsWithOptions() error = %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 900*time.Millisecond)
	defer cancel()

	var got []string
	err = pods.LogsStream(ctx, item, opts, func(line string) {
		got = append(got, line)
	})
	if !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context cancellation, got %v", err)
	}
	if len(got) <= len(initial) {
		t.Fatalf("expected streamed follow lines beyond initial snapshot (%d), got %d", len(initial), len(got))
	}
}
