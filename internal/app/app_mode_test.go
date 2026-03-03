package app

import (
	"errors"
	"testing"

	"github.com/dloss/podji/internal/data"
)

func TestNewFromEnvUnknownModeReturnsError(t *testing.T) {
	prev := newStoreFromEnvFn
	newStoreFromEnvFn = func() (data.Store, error) {
		return nil, errors.New("unknown PODJI_MODE")
	}
	t.Cleanup(func() { newStoreFromEnvFn = prev })
	if _, err := NewFromEnv(); err == nil {
		t.Fatal("expected error")
	}
}

func TestNewFromEnvKubeErrorReturnsError(t *testing.T) {
	prev := newStoreFromEnvFn
	newStoreFromEnvFn = func() (data.Store, error) {
		return nil, errors.New("kube mode unavailable")
	}
	t.Cleanup(func() { newStoreFromEnvFn = prev })

	if _, err := NewFromEnv(); err == nil {
		t.Fatal("expected error")
	}
}

func TestNewFromEnvSuccess(t *testing.T) {
	prev := newStoreFromEnvFn
	newStoreFromEnvFn = func() (data.Store, error) {
		return data.NewMockStore(), nil
	}
	t.Cleanup(func() { newStoreFromEnvFn = prev })

	m, err := NewFromEnv()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if m.mode != data.ModeMock {
		t.Fatalf("expected mock mode label, got %q", m.mode)
	}
}

func TestNewUsesStoreScopeFromFactory(t *testing.T) {
	prev := newStoreFromEnvFn
	newStoreFromEnvFn = func() (data.Store, error) {
		store := data.NewMockStore()
		store.SetScope(data.Scope{Context: "prod-cluster", Namespace: "staging"})
		return store, nil
	}
	t.Cleanup(func() { newStoreFromEnvFn = prev })

	m, err := NewFromEnv()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
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
	newStoreFromEnvFn = func() (data.Store, error) {
		return data.NewMockStore(), nil
	}
	t.Cleanup(func() { newStoreFromEnvFn = prev })

	m, err := NewFromEnv()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(m.stack) == 0 {
		t.Fatal("expected non-empty startup stack")
	}
	if len(m.crumbs) == 0 || m.crumbs[0] != "workloads" {
		t.Fatalf("expected workloads root crumb at startup, got %#v", m.crumbs)
	}
}
