package app

import (
	"strings"
	"testing"

	"github.com/dloss/podji/internal/data"
)

func TestNewUnknownModeSetsStatusMessage(t *testing.T) {
	prev := newStoreFromEnvFn
	newStoreFromEnvFn = func() (data.Store, string) {
		return data.NewMockStore(), "unknown PODJI_MODE=\"unknown\" (using mock mode)"
	}
	t.Cleanup(func() { newStoreFromEnvFn = prev })
	m := New()
	if !strings.Contains(m.statusMsg, "unknown PODJI_MODE") {
		t.Fatalf("expected unknown mode status message, got %q", m.statusMsg)
	}
}

func TestNewKubeFallbackWarningSetsStatusMessage(t *testing.T) {
	prev := newStoreFromEnvFn
	newStoreFromEnvFn = func() (data.Store, string) {
		return data.NewMockStore(), "kube mode unavailable: boom (using mock mode)"
	}
	t.Cleanup(func() { newStoreFromEnvFn = prev })

	m := New()
	if !strings.Contains(m.statusMsg, "kube mode unavailable") {
		t.Fatalf("expected kube fallback warning in statusMsg, got %q", m.statusMsg)
	}
}

func TestNewKubeModeWithoutWarningKeepsStatusMessageEmpty(t *testing.T) {
	prev := newStoreFromEnvFn
	newStoreFromEnvFn = func() (data.Store, string) {
		return data.NewMockStore(), ""
	}
	t.Cleanup(func() { newStoreFromEnvFn = prev })

	m := New()
	if m.statusMsg != "" {
		t.Fatalf("expected empty statusMsg for warning-free startup, got %q", m.statusMsg)
	}
}

func TestNewUsesStoreScopeFromFactory(t *testing.T) {
	prev := newStoreFromEnvFn
	newStoreFromEnvFn = func() (data.Store, string) {
		store := data.NewMockStore()
		store.SetScope(data.Scope{Context: "prod-cluster", Namespace: "staging"})
		return store, ""
	}
	t.Cleanup(func() { newStoreFromEnvFn = prev })

	m := New()
	if m.context != "prod-cluster" {
		t.Fatalf("expected context from startup store, got %q", m.context)
	}
	if m.namespace != "staging" {
		t.Fatalf("expected namespace from startup store, got %q", m.namespace)
	}
	if m.registry.Namespace() != "staging" {
		t.Fatalf("expected registry namespace to follow startup scope, got %q", m.registry.Namespace())
	}
}

func TestNewStartupStackUsesWorkloadsRoot(t *testing.T) {
	prev := newStoreFromEnvFn
	newStoreFromEnvFn = func() (data.Store, string) {
		return data.NewMockStore(), ""
	}
	t.Cleanup(func() { newStoreFromEnvFn = prev })

	m := New()
	if len(m.stack) == 0 {
		t.Fatal("expected non-empty startup stack")
	}
	if len(m.crumbs) == 0 || m.crumbs[0] != "workloads" {
		t.Fatalf("expected workloads root crumb at startup, got %#v", m.crumbs)
	}
}
