#!/bin/bash
# KEDA e2e test: Apply kind resources and initialize PostgreSQL in one go
set -e

SCRIPT_DIR=$(cd "$(dirname "$0")" && pwd)
REPO_ROOT="$SCRIPT_DIR/.."

# 0. Build Docker image and load to kind
cd "$REPO_ROOT"
echo "[INFO] Building calendar-scaler:latest image..."
docker build -t calendar-scaler:latest .
echo "[INFO] Loading calendar-scaler:latest image to kind cluster..."
kind load docker-image calendar-scaler:latest
cd "$SCRIPT_DIR"

# 1. Apply resources to kind cluster
echo "[INFO] Applying KEDA/DB/ScaledObject resources to kind cluster..."
bash "$SCRIPT_DIR/apply_psql_res_to_kind.sh"

sleep 120

# 2. Initialize PostgreSQL and run e2e test
bash "$SCRIPT_DIR/setup_postgres.sh"

# 3. Run e2e test
bash "$SCRIPT_DIR/test_postgres_e2e.sh"

# 4. Cleanup
bash "$SCRIPT_DIR/cleanup_postgres.sh"

