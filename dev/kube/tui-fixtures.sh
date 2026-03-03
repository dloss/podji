#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
NS="tui-e2e"

create() {
  echo ">> Creating namespace $NS"
  kubectl create ns "$NS" 2>/dev/null || true

  echo ">> Applying fixtures"
  kubectl -n "$NS" apply -f "$SCRIPT_DIR/fixtures.yaml"

  echo ">> Waiting for web deployment"
  kubectl -n "$NS" rollout status deploy/web --timeout=30s || true
}

delete_ns() {
  echo ">> Deleting namespace"
  kubectl delete ns "$NS" --wait=false 2>/dev/null || true
}

storm() {
  echo ">> Generating continuous events (Ctrl+C to stop)"
  while true; do
    kubectl -n "$NS" scale deploy/web --replicas=3 >/dev/null
    sleep 2
    kubectl -n "$NS" scale deploy/web --replicas=1 >/dev/null
    sleep 2
    kubectl -n "$NS" patch configmap ticks \
      -p "{\"data\":{\"v\":\"$(date +%s)\"}}" >/dev/null
    sleep 2
  done
}

case "${1:-}" in
  up)
    create
    ;;
  down)
    delete_ns
    ;;
  reset)
    delete_ns
    sleep 2
    create
    ;;
  storm)
    storm
    ;;
  *)
    echo "Usage:"
    echo "  $0 up"
    echo "  $0 down"
    echo "  $0 reset"
    echo "  $0 storm"
    exit 1
    ;;
esac
