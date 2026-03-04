package data

import (
	"context"

	"github.com/dloss/podji/internal/resources"
)

// ReadModel defines resource read operations independent from concrete data source.
// Implementations may be mock-backed, informer-backed, or API-backed.
type ReadModel interface {
	List(resourceName string, scope Scope) ([]resources.ResourceItem, error)
	Detail(resourceName string, item resources.ResourceItem, scope Scope) (resources.DetailData, error)
	Logs(resourceName string, item resources.ResourceItem, scope Scope) ([]string, error)
	Events(resourceName string, item resources.ResourceItem, scope Scope) ([]string, error)
	YAML(resourceName string, item resources.ResourceItem, scope Scope) (string, error)
	Describe(resourceName string, item resources.ResourceItem, scope Scope) (string, error)
}

type LogOptions struct {
	Tail      int
	Follow    bool
	Previous  bool
	Container string
}

type EventOptions struct {
	Limit int
}

// StreamingReadModel optionally extends ReadModel with context-aware access for
// cancellation and future follow/tail behavior.
type StreamingReadModel interface {
	LogsWithContext(ctx context.Context, resourceName string, item resources.ResourceItem, scope Scope, opts LogOptions) ([]string, error)
	EventsWithContext(ctx context.Context, resourceName string, item resources.ResourceItem, scope Scope, opts EventOptions) ([]string, error)
}

// LogStreamReadModel optionally extends ReadModel with incremental log
// streaming support.
type LogStreamReadModel interface {
	StreamLogsWithContext(ctx context.Context, resourceName string, item resources.ResourceItem, scope Scope, opts LogOptions, onLine func(string)) error
}

func ReadLogs(ctx context.Context, read ReadModel, resourceName string, item resources.ResourceItem, scope Scope, opts LogOptions) ([]string, error) {
	if streaming, ok := read.(StreamingReadModel); ok {
		return streaming.LogsWithContext(ctx, resourceName, item, scope, opts)
	}
	return read.Logs(resourceName, item, scope)
}

func StreamLogs(ctx context.Context, read ReadModel, resourceName string, item resources.ResourceItem, scope Scope, opts LogOptions, onLine func(string)) error {
	if streaming, ok := read.(LogStreamReadModel); ok {
		return streaming.StreamLogsWithContext(ctx, resourceName, item, scope, opts, onLine)
	}
	lines, err := ReadLogs(ctx, read, resourceName, item, scope, opts)
	if err != nil {
		return err
	}
	for _, line := range lines {
		onLine(line)
	}
	return nil
}

func ReadEvents(ctx context.Context, read ReadModel, resourceName string, item resources.ResourceItem, scope Scope, opts EventOptions) ([]string, error) {
	if streaming, ok := read.(StreamingReadModel); ok {
		return streaming.EventsWithContext(ctx, resourceName, item, scope, opts)
	}
	return read.Events(resourceName, item, scope)
}
