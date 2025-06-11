#!/bin/bash
set -e

NAMESPACE=myscaler
PG_POD=$(kubectl get pod -n "$NAMESPACE" -l app=postgresql -o jsonpath='{.items[0].metadata.name}')

# Apply ScaledObject with targetColumn
kubectl apply -f "$(dirname "$0")/resources/scaledobject-psql-targetcol.yaml"

# Insert test record for targetColumn scenario
NOW=$(date '+%Y-%m-%dT%H:%M:%S+09:00')
END=$(date -d "+2 min" '+%Y-%m-%dT%H:%M:%S+09:00')
echo "[INFO] Insert test record for targetColumn scenario ($NOW ~ $END)"
kubectl exec -n "$NAMESPACE" "$PG_POD" -- psql -U postgres -d calendar -c "INSERT INTO calendar_events VALUES ('$NOW', '$END', 3, '$NAMESPACE/scale-target-targetcol,${NAMESPACE}2/');"

# Observe scaling for 2 minutes
for i in {1..18}; do
  kubectl get deployment scale-target --no-headers -n "$NAMESPACE"
  sleep 10
done

# Delete test records
kubectl exec -n "$NAMESPACE" "$PG_POD" -- psql -U postgres -d calendar -c "TRUNCATE TABLE calendar_events;"
# Delete ScaledObject
kubectl delete -f "$(dirname "$0")/resources/scaledobject-psql-targetcol.yaml"

# Apply ScaledObject without targetColumn
kubectl apply -f "$(dirname "$0")/resources/scaledobject-psql-notargetcol.yaml"

# Insert test record for no targetColumn scenario
NOW2=$(date '+%Y-%m-%dT%H:%M:%S+09:00')
END2=$(date -d "+2 min" '+%Y-%m-%dT%H:%M:%S+09:00')
echo "[INFO] Insert test record for no targetColumn scenario ($NOW2 ~ $END2)"
kubectl exec -n "$NAMESPACE" "$PG_POD" -- psql -U postgres -d calendar -c "INSERT INTO calendar_events VALUES ('$NOW2', '$END2', 2, NULL);"

# Observe scaling for 2 minutes
for i in {1..18}; do
  kubectl get deployment scale-target --no-headers -n "$NAMESPACE"
  sleep 10
done

# Delete test records
kubectl exec -n "$NAMESPACE" "$PG_POD" -- psql -U postgres -d calendar -c "TRUNCATE TABLE calendar_events;"
# Delete ScaledObject
kubectl delete -f "$(dirname "$0")/resources/scaledobject-psql-notargetcol.yaml"

# Final cleanup
kubectl exec -n "$NAMESPACE" "$PG_POD" -- psql -U postgres -d calendar -c "TRUNCATE TABLE calendar_events;"

echo "[INFO] PostgreSQL table initialized."
