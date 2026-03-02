package resources

import "testing"

func TestObjectRefStableKeyPrefersUID(t *testing.T) {
	item := ResourceItem{
		UID:        "1234-abcd",
		APIVersion: "v1",
		Kind:       "Pod",
		Namespace:  "default",
		Name:       "api",
	}
	if got := item.Ref().StableKey(); got != "uid:1234-abcd" {
		t.Fatalf("expected uid key, got %q", got)
	}
}

func TestObjectRefStableKeyFallsBackToTuple(t *testing.T) {
	item := ResourceItem{
		APIVersion: "apps/v1",
		Kind:       "Deployment",
		Namespace:  "staging",
		Name:       "api",
	}
	if got := item.Ref().StableKey(); got != "apps/v1/Deployment/staging/api" {
		t.Fatalf("unexpected stable key %q", got)
	}
}
