#!/bin/bash
set -e

NAMESPACE=myscaler
PG_POD=$(kubectl get pod -n "$NAMESPACE" -l app=postgresql -o jsonpath='{.items[0].metadata.name}')

# Check if calendar DB exists
DB_EXISTS=$(kubectl exec -n "$NAMESPACE" "$PG_POD" -- psql -U postgres -tAc "SELECT 1 FROM pg_database WHERE datname='calendar';")
if [ "$DB_EXISTS" != "1" ]; then
  echo "[INFO] calendar database does not exist. Creating."
  kubectl exec -n "$NAMESPACE" "$PG_POD" -- psql -U postgres -c "CREATE DATABASE calendar;"
fi

# Check if table exists
TABLE_EXISTS=$(kubectl exec -n "$NAMESPACE" "$PG_POD" -- psql -U postgres -d calendar -tAc "SELECT to_regclass('public.calendar_events') IS NOT NULL;")

if [ "$TABLE_EXISTS" = "t" ]; then
  echo "[INFO] calendar_events table already exists. Only TRUNCATE."
  kubectl exec -n "$NAMESPACE" "$PG_POD" -- psql -U postgres -d calendar -c "TRUNCATE TABLE calendar_events;"
else
  echo "[INFO] Creating calendar_events table."
  kubectl exec -n "$NAMESPACE" "$PG_POD" -- psql -U postgres -d calendar -c "CREATE TABLE IF NOT EXISTS calendar_events (startEvent TIMESTAMP NOT NULL, endEvent TIMESTAMP NOT NULL, desiredReplicas INTEGER NOT NULL, targetWorkload TEXT);"
  kubectl exec -n "$NAMESPACE" "$PG_POD" -- psql -U postgres -d calendar -c "TRUNCATE TABLE calendar_events;"
fi

echo "[INFO] PostgreSQL table initialization complete"
