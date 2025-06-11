#!/bin/bash
set -e

SCRIPT_DIR=$(cd "$(dirname "$0")" && pwd)

kubectl apply -f "$SCRIPT_DIR/resources/components.yaml"
kubectl apply -f "$SCRIPT_DIR/resources/hpa-external-metrics-rbac.yaml"
kubectl apply -f "$SCRIPT_DIR/resources/deploy-dynamodb.yaml"

echo "[INFO] Applied KEDA/DB/ScaledObject resources to kind cluster."

sleep 10
