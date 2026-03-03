package data

import (
	"strings"
	"testing"
	"time"

	"github.com/dloss/podji/internal/resources"
	appsv1 "k8s.io/api/apps/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

func TestDetailFromObjectStatefulSetIncludesWorkloadFields(t *testing.T) {
	replicas := int32(3)
	obj := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "db",
			Labels: map[string]string{"app": "db"},
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas:    &replicas,
			ServiceName: "db-headless",
			Selector:    &metav1.LabelSelector{MatchLabels: map[string]string{"app": "db"}},
		},
		Status: appsv1.StatefulSetStatus{
			ReadyReplicas: 2,
		},
	}
	detail := detailFromObject(obj, "workloads", resources.ResourceItem{Name: "db", Kind: "STS"})
	if len(detail.Summary) == 0 {
		t.Fatal("expected non-empty summary")
	}
	flat := strings.Join(summaryValues(detail.Summary), " ")
	if !strings.Contains(flat, "StatefulSet") || !strings.Contains(flat, "2/3") || !strings.Contains(flat, "db-headless") {
		t.Fatalf("expected statefulset detail fields, got %#v", detail.Summary)
	}
}

func TestDetailFromObjectIngressIncludesHostsAndServices(t *testing.T) {
	pathType := networkingv1.PathTypePrefix
	obj := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "web",
			Labels: map[string]string{"app": "web"},
		},
		Spec: networkingv1.IngressSpec{
			Rules: []networkingv1.IngressRule{
				{
					Host: "web.example.com",
					IngressRuleValue: networkingv1.IngressRuleValue{
						HTTP: &networkingv1.HTTPIngressRuleValue{
							Paths: []networkingv1.HTTPIngressPath{
								{
									Path:     "/",
									PathType: &pathType,
									Backend: networkingv1.IngressBackend{
										Service: &networkingv1.IngressServiceBackend{Name: "web-svc"},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	detail := detailFromObject(obj, "ingresses", resources.ResourceItem{Name: "web", Kind: "Ingress"})
	flat := strings.Join(summaryValues(detail.Summary), " ")
	if !strings.Contains(flat, "web.example.com") || !strings.Contains(flat, "web-svc") {
		t.Fatalf("expected ingress host/service in detail summary, got %#v", detail.Summary)
	}
}

func summaryValues(summary []resources.SummaryField) []string {
	out := make([]string, 0, len(summary))
	for _, s := range summary {
		out = append(out, s.Value)
	}
	return out
}
