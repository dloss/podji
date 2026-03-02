package data

import (
	"bytes"
	"fmt"
	"os/exec"
	"sort"
	"strings"

	"github.com/dloss/podji/internal/resources"
)

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

type KubeStore struct {
	registry *resources.Registry
	scope    Scope
	runner   commandRunner
}

func NewKubeStore() (*KubeStore, error) {
	return newKubeStore(execRunner{})
}

func newKubeStore(runner commandRunner) (*KubeStore, error) {
	if runner == nil {
		runner = execRunner{}
	}
	contexts, err := kubeContexts(runner)
	if err != nil {
		return nil, err
	}

	scope := Scope{
		Context:   "default",
		Namespace: resources.DefaultNamespace,
	}
	if len(contexts) > 0 {
		scope.Context = contexts[0]
	}

	registry := resources.DefaultRegistry()
	registry.SetNamespace(scope.Namespace)

	store := &KubeStore{
		registry: registry,
		scope:    scope,
		runner:   runner,
	}
	store.configurePodFetchers()
	return store, nil
}

func (s *KubeStore) Registry() *resources.Registry {
	return s.registry
}

func (s *KubeStore) Scope() Scope {
	return s.scope
}

func (s *KubeStore) SetScope(scope Scope) {
	if scope.Context == "" {
		scope.Context = s.scope.Context
	}
	scope.Namespace = normalizeScopeNamespace(scope.Namespace)
	s.scope = scope
	s.registry.SetNamespace(s.scope.Namespace)
}

func (s *KubeStore) NamespaceNames() []string {
	namespaces, err := kubeNamespaces(s.runner, s.scope.Context)
	if err != nil || len(namespaces) == 0 {
		return []string{resources.AllNamespaces, resources.DefaultNamespace}
	}
	out := make([]string, 0, len(namespaces)+1)
	out = append(out, resources.AllNamespaces)
	out = append(out, namespaces...)
	return out
}

func (s *KubeStore) ContextNames() []string {
	contexts, err := kubeContexts(s.runner)
	if err != nil || len(contexts) == 0 {
		return []string{s.scope.Context}
	}
	return contexts
}

func (s *KubeStore) UnhealthyItems() []resources.ResourceItem {
	// Query aggregation still uses the shared resource query path for now.
	return resources.UnhealthyItems(s.scope.Namespace)
}

func (s *KubeStore) PodsByRestarts() []resources.ResourceItem {
	// Query aggregation still uses the shared resource query path for now.
	return resources.PodsByRestarts(s.scope.Namespace)
}

func (s *KubeStore) configurePodFetchers() {
	pods, ok := s.registry.ByName("pods").(*resources.Pods)
	if !ok {
		return
	}
	pods.SetLiveFetchers(s.podLogs, s.podEvents)
}

func (s *KubeStore) podLogs(namespace, pod string) ([]string, error) {
	args := []string{
		"--context", s.scope.Context,
		"-n", namespace,
		"logs", pod,
		"--tail=200",
	}
	out, err := s.runner.Run("kubectl", args...)
	if err != nil {
		return nil, err
	}
	lines := splitNonEmptyLines(out)
	if len(lines) == 0 {
		return []string{"No log lines returned."}, nil
	}
	return lines, nil
}

func (s *KubeStore) podEvents(namespace, pod string) ([]string, error) {
	args := []string{
		"--context", s.scope.Context,
		"-n", namespace,
		"get", "events",
		"--field-selector", "involvedObject.name=" + pod,
		"-o", `jsonpath={range .items[*]}{.lastTimestamp}{"   "}{.type}{"   "}{.reason}{"   "}{.message}{"\n"}{end}`,
	}
	out, err := s.runner.Run("kubectl", args...)
	if err != nil {
		return nil, err
	}
	lines := splitNonEmptyLines(out)
	if len(lines) == 0 {
		return []string{"—   No recent events"}, nil
	}
	return lines, nil
}

func kubeContexts(runner commandRunner) ([]string, error) {
	out, err := runner.Run("kubectl", "config", "get-contexts", "-o", "name")
	if err != nil {
		return nil, fmt.Errorf("failed to list kube contexts: %w", err)
	}
	lines := splitNonEmptyLines(out)
	sort.Strings(lines)
	return lines, nil
}

func kubeNamespaces(runner commandRunner, context string) ([]string, error) {
	args := []string{"get", "namespaces", "-o", "jsonpath={range .items[*]}{.metadata.name}{\"\\n\"}{end}"}
	if context != "" {
		args = append([]string{"--context", context}, args...)
	}
	out, err := runner.Run("kubectl", args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list namespaces for context %q: %w", context, err)
	}
	lines := splitNonEmptyLines(out)
	sort.Strings(lines)
	return lines, nil
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
