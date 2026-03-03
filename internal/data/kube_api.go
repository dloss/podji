package data

import (
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

// KubeObjectReader is an optional extension for typed object fetches used by
// live YAML/describe rendering paths.
type KubeObjectReader interface {
	ResourceDetail(context, namespace, resourceName string, item resources.ResourceItem) (resources.DetailData, error)
	ResourceYAML(context, namespace, resourceName string, item resources.ResourceItem) (string, error)
	ResourceDescribe(context, namespace, resourceName string, item resources.ResourceItem) (string, error)
}
