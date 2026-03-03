package data

import (
	"strings"
	"testing"
	"time"

	"github.com/dloss/podji/internal/resources"
)

func TestClientGoNamespaceCacheHitAndExpire(t *testing.T) {
	api := &clientGoAPI{
		nsTTL: 25 * time.Millisecond,
		ns:    map[string]namespaceCacheEntry{},
	}
	api.namespaceCacheSet("dev", []string{"default", "kube-system"})

	got, ok := api.namespaceCacheGet("dev")
	if !ok {
		t.Fatal("expected namespace cache hit")
	}
	if len(got) != 2 || got[0] != "default" || got[1] != "kube-system" {
		t.Fatalf("unexpected cached namespaces: %#v", got)
	}

	time.Sleep(35 * time.Millisecond)
	if _, ok := api.namespaceCacheGet("dev"); ok {
		t.Fatal("expected namespace cache entry to expire")
	}
}

func TestClientGoListCacheHitAndExpire(t *testing.T) {
	api := &clientGoAPI{
		listTTL: 25 * time.Millisecond,
		list:    map[string]listCacheEntry{},
	}
	api.listCacheSet("dev|default|pods", []resources.ResourceItem{{Name: "api-a"}})

	got, ok := api.listCacheGet("dev|default|pods")
	if !ok {
		t.Fatal("expected list cache hit")
	}
	if len(got) != 1 || got[0].Name != "api-a" {
		t.Fatalf("unexpected cached list entries: %#v", got)
	}

	time.Sleep(35 * time.Millisecond)
	if _, ok := api.listCacheGet("dev|default|pods"); ok {
		t.Fatal("expected list cache entry to expire")
	}
}

func TestBoundedNonEmptyLinesTrimsAndBounds(t *testing.T) {
	in := strings.NewReader("\n  a  \n\nb\nc\n")
	got, err := boundedNonEmptyLines(in, 2)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(got) != 2 || got[0] != "b" || got[1] != "c" {
		t.Fatalf("expected bounded tail lines [b c], got %#v", got)
	}
}

func TestBoundedNonEmptyLinesHandlesZeroMax(t *testing.T) {
	in := strings.NewReader("a\nb\n")
	got, err := boundedNonEmptyLines(in, 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(got) != 1 || got[0] != "b" {
		t.Fatalf("expected single most recent line, got %#v", got)
	}
}
