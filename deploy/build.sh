#!/usr/bin/env bash
# Build both container images.
#   ./deploy/build.sh                       -> tags :local (for minikube/kind)
#   REGISTRY=ghcr.io/me TAG=v1 ./deploy/build.sh [--push]
set -euo pipefail
cd "$(dirname "$0")/.."

REGISTRY="${REGISTRY:-ghcr.io/manaf}"
TAG="${TAG:-local}"
PUSH=0
[[ "${1:-}" == "--push" ]] && PUSH=1

BACKEND_IMG="${REGISTRY}/ckad-backend:${TAG}"
FRONTEND_IMG="${REGISTRY}/ckad-frontend:${TAG}"

echo "==> Building backend  ${BACKEND_IMG}"
docker build -f deploy/Dockerfile.backend -t "${BACKEND_IMG}" backend

echo "==> Building frontend ${FRONTEND_IMG}"
docker build -f deploy/Dockerfile.frontend -t "${FRONTEND_IMG}" .

if [[ $PUSH -eq 1 ]]; then
  echo "==> Pushing images"
  docker push "${BACKEND_IMG}"
  docker push "${FRONTEND_IMG}"
fi

echo "==> Done: ${BACKEND_IMG} , ${FRONTEND_IMG}"
