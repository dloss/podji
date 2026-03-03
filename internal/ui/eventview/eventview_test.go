package eventview

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	bubbletea "github.com/charmbracelet/bubbletea"
	"github.com/dloss/podji/internal/resources"
)

type eventOptionsResource struct {
	base resources.ResourceType
}

func (e *eventOptionsResource) Name() string                        { return e.base.Name() }
func (e *eventOptionsResource) Key() rune                           { return e.base.Key() }
func (e *eventOptionsResource) Items() []resources.ResourceItem     { return e.base.Items() }
func (e *eventOptionsResource) Sort(items []resources.ResourceItem) { e.base.Sort(items) }
func (e *eventOptionsResource) Detail(item resources.ResourceItem) resources.DetailData {
	return e.base.Detail(item)
}
func (e *eventOptionsResource) Logs(item resources.ResourceItem) []string { return e.base.Logs(item) }
func (e *eventOptionsResource) Events(item resources.ResourceItem) []string {
	return e.base.Events(item)
}
func (e *eventOptionsResource) YAML(item resources.ResourceItem) string { return e.base.YAML(item) }
func (e *eventOptionsResource) Describe(item resources.ResourceItem) string {
	return e.base.Describe(item)
}
func (e *eventOptionsResource) EventsWithOptions(ctx context.Context, item resources.ResourceItem, opts resources.EventOptions) ([]string, error) {
	return []string{"2026-03-03T12:00:00Z   Warning   BackOff   restart loop"}, nil
}

func TestInitLoadsEventsFromOptionReader(t *testing.T) {
	res := &eventOptionsResource{base: resources.NewPods()}
	v := New(resources.ResourceItem{Name: "api"}, res)
	cmd := v.Init()
	if cmd == nil {
		t.Fatal("expected init command")
	}
	msg := cmd()
	update := v.Update(msg)
	_ = update
	v.SetSize(80, 20)
	if !strings.Contains(v.View(), "BackOff") {
		t.Fatalf("expected loaded event content, got %q", v.View())
	}
}

type blockingEventResource struct {
	base      resources.ResourceType
	ctxSeen   chan context.Context
	cancelled chan struct{}
}

func (b *blockingEventResource) Name() string                        { return b.base.Name() }
func (b *blockingEventResource) Key() rune                           { return b.base.Key() }
func (b *blockingEventResource) Items() []resources.ResourceItem     { return b.base.Items() }
func (b *blockingEventResource) Sort(items []resources.ResourceItem) { b.base.Sort(items) }
func (b *blockingEventResource) Detail(item resources.ResourceItem) resources.DetailData {
	return b.base.Detail(item)
}
func (b *blockingEventResource) Logs(item resources.ResourceItem) []string { return b.base.Logs(item) }
func (b *blockingEventResource) Events(item resources.ResourceItem) []string {
	return b.base.Events(item)
}
func (b *blockingEventResource) YAML(item resources.ResourceItem) string { return b.base.YAML(item) }
func (b *blockingEventResource) Describe(item resources.ResourceItem) string {
	return b.base.Describe(item)
}
func (b *blockingEventResource) EventsWithOptions(ctx context.Context, item resources.ResourceItem, opts resources.EventOptions) ([]string, error) {
	select {
	case b.ctxSeen <- ctx:
	default:
	}
	<-ctx.Done()
	select {
	case b.cancelled <- struct{}{}:
	default:
	}
	return nil, ctx.Err()
}

func TestDisposeCancelsInFlightEventLoad(t *testing.T) {
	res := &blockingEventResource{
		base:      resources.NewPods(),
		ctxSeen:   make(chan context.Context, 1),
		cancelled: make(chan struct{}, 1),
	}
	v := New(resources.ResourceItem{Name: "api"}, res)
	cmd := v.Init()
	if cmd == nil {
		t.Fatal("expected init command")
	}
	done := make(chan bubbletea.Msg, 1)
	go func() { done <- cmd() }()
	select {
	case <-res.ctxSeen:
	case <-time.After(1 * time.Second):
		t.Fatal("expected in-flight event request to start")
	}
	v.Dispose()
	select {
	case <-res.cancelled:
	case <-time.After(1 * time.Second):
		t.Fatal("expected dispose to cancel in-flight event request")
	}
	select {
	case msg := <-done:
		result, ok := msg.(eventReloadResultMsg)
		if !ok {
			t.Fatalf("expected eventReloadResultMsg, got %T", msg)
		}
		if !errors.Is(result.err, context.Canceled) && !errors.Is(result.err, context.DeadlineExceeded) {
			t.Fatalf("expected cancellation error, got %v", result.err)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("expected event command to return after cancellation")
	}
}
