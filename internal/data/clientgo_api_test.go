package data

import (
	"testing"
	"time"
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
