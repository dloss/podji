package data

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"sort"
	"strings"

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

type commandRunner interface {
	Run(name string, args ...string) (string, error)
}

type execRunner struct{}

func (r execRunner) Run(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	var out bytes.Buffer
	var errOut bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errOut
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(errOut.String())
		if msg == "" {
			msg = err.Error()
		}
		return "", fmt.Errorf("%s: %w", msg, err)
	}
	return out.String(), nil
}

type kubectlAPI struct {
	runner commandRunner
}

func newKubectlAPI(runner commandRunner) KubeAPI {
	if runner == nil {
		runner = execRunner{}
	}
	return &kubectlAPI{runner: runner}
}

func (k *kubectlAPI) Contexts() ([]string, error) {
	out, err := k.runner.Run("kubectl", "config", "get-contexts", "-o", "name")
	if err != nil {
		return nil, fmt.Errorf("failed to list kube contexts: %w", err)
	}
	lines := splitNonEmptyLines(out)
	sort.Strings(lines)
	return lines, nil
}

func (k *kubectlAPI) Namespaces(context string) ([]string, error) {
	args := []string{"get", "namespaces", "-o", "jsonpath={range .items[*]}{.metadata.name}{\"\\n\"}{end}"}
	if context != "" {
		args = append([]string{"--context", context}, args...)
	}
	out, err := k.runner.Run("kubectl", args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list namespaces for context %q: %w", context, err)
	}
	lines := splitNonEmptyLines(out)
	sort.Strings(lines)
	return lines, nil
}

func (k *kubectlAPI) PodLogs(context, namespace, pod string, tail int) ([]string, error) {
	if tail <= 0 {
		tail = 200
	}
	args := []string{
		"--context", context,
		"-n", namespace,
		"logs", pod,
		fmt.Sprintf("--tail=%d", tail),
	}
	out, err := k.runner.Run("kubectl", args...)
	if err != nil {
		return nil, err
	}
	lines := splitNonEmptyLines(out)
	if len(lines) == 0 {
		return []string{"No log lines returned."}, nil
	}
	return lines, nil
}

func (k *kubectlAPI) PodEvents(context, namespace, pod string) ([]string, error) {
	args := []string{
		"--context", context,
		"-n", namespace,
		"get", "events",
		"--field-selector", "involvedObject.name=" + pod,
		"-o", `jsonpath={range .items[*]}{.lastTimestamp}{"   "}{.type}{"   "}{.reason}{"   "}{.message}{"\n"}{end}`,
	}
	out, err := k.runner.Run("kubectl", args...)
	if err != nil {
		return nil, err
	}
	lines := splitNonEmptyLines(out)
	if len(lines) == 0 {
		return []string{"—   No recent events"}, nil
	}
	return lines, nil
}

func (k *kubectlAPI) ListResources(context, namespace, resourceName string) ([]resources.ResourceItem, error) {
	return nil, fmt.Errorf("%w: %s", ErrListNotSupported, resourceName)
}

func splitNonEmptyLines(raw string) []string {
	lines := strings.Split(raw, "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		out = append(out, line)
	}
	return out
}
