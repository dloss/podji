#!/usr/bin/env bash
set -euo pipefail

CLUSTER="dev"
NS="tui-e2e"

create_cluster() {
  echo ">> Creating k3d cluster: $CLUSTER"
  k3d cluster create "$CLUSTER" \
    --servers 1 \
    --agents 0 \
    --no-lb \
    --k3s-arg "--disable=traefik@server:0" \
    --k3s-arg "--disable=metrics-server@server:0"

  kubectl config use-context "k3d-$CLUSTER"
}

delete_cluster() {
  echo ">> Deleting cluster: $CLUSTER"
  k3d cluster delete "$CLUSTER" || true
}

create_ns() {
  echo ">> Creating namespace + fixtures: $NS"
  kubectl create ns "$NS" 2>/dev/null || true

  kubectl -n "$NS" apply -f - <<'YAML'
apiVersion: apps/v1
kind: Deployment
metadata:
  name: web
spec:
  replicas: 1
  selector:
    matchLabels:
      app: web
  template:
    metadata:
      labels:
        app: web
    spec:
      containers:
      - name: nginx
        image: nginx:alpine
        ports:
        - containerPort: 80
---
apiVersion: v1
kind: Pod
metadata:
  name: crashloop
spec:
  restartPolicy: Always
  containers:
  - name: boom
    image: busybox:1.36
    command: ["sh","-c","echo boom; sleep 1; exit 1"]
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: ticks
data:
  v: "0"
---
apiVersion: batch/v1
kind: Job
metadata:
  name: oneshot
spec:
  template:
    spec:
      restartPolicy: Never
      containers:
      - name: curl
        image: curlimages/curl:8.10.1
        command: ["sh","-c","echo hello; sleep 2;"]
YAML
}

delete_ns() {
  echo ">> Deleting namespace: $NS"
  kubectl delete ns "$NS" --wait=false 2>/dev/null || true
}

case "${1:-}" in
  up-cluster)
    create_cluster
    create_ns
    ;;
  down-cluster)
    delete_cluster
    ;;
  up-ns)
    create_ns
    ;;
  down-ns)
    delete_ns
    ;;
  reset-ns)
    delete_ns
    sleep 2
    create_ns
    ;;
  *)
    echo "Usage:"
    echo "  $0 up-cluster"
    echo "  $0 down-cluster"
    echo "  $0 up-ns"
    echo "  $0 down-ns"
    echo "  $0 reset-ns"
    exit 1
    ;;
esac
