#!/usr/bin/env bash
# Deploy to the CURRENT kubectl context.
#   ./deploy/deploy.sh                 # uses images as tagged in kustomization
#   ./deploy/deploy.sh minikube        # after `minikube image load ...`
set -euo pipefail
cd "$(dirname "$0")/.."

CONTEXT="${1:-}"
ARGS=()
[[ -n "$CONTEXT" ]] && ARGS=(--context "$CONTEXT")

echo "==> Applying manifests (kustomize)"
kubectl "${ARGS[@]}" apply -k deploy/k8s

echo "==> Waiting for rollouts"
kubectl "${ARGS[@]}" -n ckad-simulator rollout status deploy/backend  --timeout=180s
kubectl "${ARGS[@]}" -n ckad-simulator rollout status deploy/frontend --timeout=180s

echo
echo "==> Access:"
if [[ "$CONTEXT" == "minikube" || -z "$CONTEXT" ]]; then
  echo "    minikube service frontend -n ckad-simulator"
  echo "    # or: kubectl port-forward svc/frontend 8081:80 -n ckad-simulator"
else
  NODE_IP=$(kubectl "${ARGS[@]}" get nodes -o jsonpath='{.items[0].status.addresses[?(@.type=="ExternalIP")].address}')
  echo "    http://${NODE_IP:-<node-ip>}:30080"
fi
