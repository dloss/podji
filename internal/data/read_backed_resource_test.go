package data

import (
	"context"
	"errors"
	"testing"

	"github.com/dloss/podji/internal/resources"
)

type fakeReadModel struct {
	items []resources.ResourceItem
	err   error
}

func (f fakeReadModel) List(resourceName string, scope Scope) ([]resources.ResourceItem, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.items, nil
}

func (f fakeReadModel) Detail(resourceName string, item resources.ResourceItem, scope Scope) (resources.DetailData, error) {
	if f.err != nil {
		return resources.DetailData{}, f.err
	}
	return resources.DetailData{Summary: []resources.SummaryField{{Key: "status", Value: "from-read-model"}}}, nil
}

func (f fakeReadModel) Logs(resourceName string, item resources.ResourceItem, scope Scope) ([]string, error) {
	if f.err != nil {
		return nil, f.err
	}
	return []string{"read-model-log"}, nil
}

func (f fakeReadModel) Events(resourceName string, item resources.ResourceItem, scope Scope) ([]string, error) {
	if f.err != nil {
		return nil, f.err
	}
	return []string{"read-model-event"}, nil
}

func (f fakeReadModel) YAML(resourceName string, item resources.ResourceItem, scope Scope) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	return "read-model-yaml", nil
}

func (f fakeReadModel) Describe(resourceName string, item resources.ResourceItem, scope Scope) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	return "read-model-describe", nil
}

func TestReadBackedResourceDelegatesToReadModel(t *testing.T) {
	base := resources.NewPods()
	base.SetNamespace("default")
	adapter := NewReadBackedResource(base, fakeReadModel{
		items: []resources.ResourceItem{{Name: "from-read-model"}},
	}, func() Scope { return Scope{Context: "default", Namespace: "default"} })

	items := adapter.Items()
	if len(items) != 1 || items[0].Name != "from-read-model" {
		t.Fatalf("expected read-model items, got %#v", items)
	}
	if got := adapter.Logs(resources.ResourceItem{Name: "x"}); len(got) == 0 || got[0] != "read-model-log" {
		t.Fatalf("expected read-model logs, got %#v", got)
	}
}

func TestReadBackedResourceFallsBackToBaseOnError(t *testing.T) {
	base := resources.NewPods()
	base.SetNamespace("default")
	adapter := NewReadBackedResource(base, fakeReadModel{err: errors.New("boom")}, func() Scope {
		return Scope{Context: "default", Namespace: "default"}
	})
	items := adapter.Items()
	if len(items) == 0 {
		t.Fatal("expected fallback items from base resource")
	}
}

type fakeStreamingReadModel struct {
	fakeReadModel
	lastLogOptions   LogOptions
	lastEventOptions EventOptions
}

func (f *fakeStreamingReadModel) LogsWithContext(ctx context.Context, resourceName string, item resources.ResourceItem, scope Scope, opts LogOptions) ([]string, error) {
	f.lastLogOptions = opts
	return []string{"streaming-log"}, nil
}

func (f *fakeStreamingReadModel) EventsWithContext(ctx context.Context, resourceName string, item resources.ResourceItem, scope Scope, opts EventOptions) ([]string, error) {
	f.lastEventOptions = opts
	return []string{"streaming-event"}, nil
}

func TestReadBackedResourceOptionReadersPropagateOptions(t *testing.T) {
	base := resources.NewPods()
	base.SetNamespace("default")
	streaming := &fakeStreamingReadModel{}
	adapter := NewReadBackedResource(base, streaming, func() Scope {
		return Scope{Context: "dev", Namespace: "default"}
	})
	logReader, ok := adapter.(resources.LogOptionsReader)
	if !ok {
		t.Fatalf("expected adapted resource to implement LogOptionsReader, got %T", adapter)
	}
	eventReader, ok := adapter.(resources.EventOptionsReader)
	if !ok {
		t.Fatalf("expected adapted resource to implement EventOptionsReader, got %T", adapter)
	}
	lines, err := logReader.LogsWithOptions(context.Background(), resources.ResourceItem{Name: "api"}, resources.LogOptions{
		Tail:   42,
		Follow: true,
	})
	if err != nil {
		t.Fatalf("expected no log error, got %v", err)
	}
	if len(lines) != 1 || lines[0] != "streaming-log" {
		t.Fatalf("expected streaming log result, got %#v", lines)
	}
	if streaming.lastLogOptions.Tail != 42 || !streaming.lastLogOptions.Follow {
		t.Fatalf("expected propagated log options, got %#v", streaming.lastLogOptions)
	}
	events, err := eventReader.EventsWithOptions(context.Background(), resources.ResourceItem{Name: "api"}, resources.EventOptions{Limit: 7})
	if err != nil {
		t.Fatalf("expected no event error, got %v", err)
	}
	if len(events) != 1 || events[0] != "streaming-event" {
		t.Fatalf("expected streaming event result, got %#v", events)
	}
	if streaming.lastEventOptions.Limit != 7 {
		t.Fatalf("expected propagated event options, got %#v", streaming.lastEventOptions)
	}
}
