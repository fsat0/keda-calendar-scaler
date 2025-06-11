#!/bin/bash
# Script to delete DynamoDB test resources
set -e

SCRIPT_DIR=$(cd "$(dirname "$0")" && pwd)

kubectl delete -f "$SCRIPT_DIR/resources/deploy-dynamodb.yaml" || true

echo "[INFO] Deleted all DynamoDB related K8s resources."
