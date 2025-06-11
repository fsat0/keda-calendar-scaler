#!/bin/bash
# Script to delete PostgreSQL test resources
set -e

SCRIPT_DIR=$(cd "$(dirname "$0")" && pwd)

kubectl delete -f "$SCRIPT_DIR/resources/deploy-postgresql.yaml" || true

echo "[INFO] Deleted all PostgreSQL related K8s resources."
