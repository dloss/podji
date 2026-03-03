package data

import (
	"context"
	"errors"

	"github.com/dloss/podji/internal/resources"
)

var ErrListNotSupported = errors.New("list not supported")
var ErrObjectReadNotSupported = errors.New("object read not supported")

type KubeAPI interface {
	Contexts() ([]string, error)
	Namespaces(context string) ([]string, error)
	ListResources(context, namespace, resourceName string) ([]resources.ResourceItem, error)
	PodLogs(context, namespace, pod string, tail int) ([]string, error)
	PodEvents(context, namespace, pod string) ([]string, error)
}

// KubeAPILogStreamer is an optional extension for incremental pod log
// streaming used by follow mode.
type KubeAPILogStreamer interface {
	PodLogsStream(ctx context.Context, contextName, namespace, pod string, tail int, onLine func(string)) error
}

// KubeAPILogOptionsReader is an optional extension for full log option support
// such as previous-container logs.
type KubeAPILogOptionsReader interface {
	PodLogsWithOptions(ctx context.Context, contextName, namespace, pod string, opts LogOptions) ([]string, error)
}

// KubeAPILogOptionsStreamer is an optional extension for full option-aware
// incremental pod log streaming.
type KubeAPILogOptionsStreamer interface {
	PodLogsStreamWithOptions(ctx context.Context, contextName, namespace, pod string, opts LogOptions, onLine func(string)) error
}

// KubeObjectReader is an optional extension for typed object fetches used by
// live YAML/describe rendering paths.
type KubeObjectReader interface {
	ResourceDetail(context, namespace, resourceName string, item resources.ResourceItem) (resources.DetailData, error)
	ResourceYAML(context, namespace, resourceName string, item resources.ResourceItem) (string, error)
	ResourceDescribe(context, namespace, resourceName string, item resources.ResourceItem) (string, error)
}
