#!/bin/bash
set -e

# Port-forward DynamoDB Local (if running in Kubernetes)
# This assumes a service named dynamodb in the myscaler namespace (see deploy-dynamodb.yaml)
kubectl port-forward svc/dynamodb-local 8000:8000 -n myscaler &
DYNAMODB_PORT_FORWARD_PID=$!
sleep 2

export AWS_PAGER=""
TABLE_NAME="calendar_events"

# 1st: Apply ScaledObject with targetAttribute
echo "[INFO] Applying ScaledObject with targetAttribute..."
kubectl apply -f "$(dirname "$0")/resources/scaledobject-dynamodb-targetattr.yaml"

# Insert test record for targetAttribute scenario
START1=$(date '+%Y-%m-%dT%H:%M:%S%:z')
END1=$(date -d "+2 min" '+%Y-%m-%dT%H:%M:%S%:z')
aws dynamodb put-item \
  --table-name "$TABLE_NAME" \
  --item "{\"EventName\": {\"S\": \"event1\"}, \"startEvent\": {\"S\": \"$START1\"}, \"endEvent\": {\"S\": \"$END1\"}, \"desiredReplicas\": {\"N\": \"3\"}, \"targetWorkload\": {\"S\": \"myscaler/scale-target-dynamodb-targetattr\"}}" \
  --endpoint-url http://localhost:8000

echo "[INFO] Inserted test record for targetAttribute scenario."

# Observe scaling for 2 minutes
for i in {1..18}; do
  kubectl get deployment scale-target --no-headers -n myscaler
  sleep 10
done

# Delete test record
aws dynamodb delete-item \
  --table-name "$TABLE_NAME" \
  --key '{"EventName": {"S": "event1"}}' \
  --endpoint-url http://localhost:8000

echo "[INFO] Deleted test record for targetAttribute scenario."

# Delete ScaledObject
echo "[INFO] Deleting ScaledObject with targetAttribute..."
kubectl delete -f "$(dirname "$0")/resources/scaledobject-dynamodb-targetattr.yaml"

# 2nd: Apply ScaledObject without targetAttribute
echo "[INFO] Applying ScaledObject without targetAttribute..."
kubectl apply -f "$(dirname "$0")/resources/scaledobject-dynamodb-notargetattr.yaml"

# Insert test record for no targetAttribute scenario
START2=$(date '+%Y-%m-%dT%H:%M:%S%:z')
END2=$(date -d "+2 min" '+%Y-%m-%dT%H:%M:%S%:z')
aws dynamodb put-item \
  --table-name "$TABLE_NAME" \
  --item "{\"EventName\": {\"S\": \"event2\"}, \"startEvent\": {\"S\": \"$START2\"}, \"endEvent\": {\"S\": \"$END2\"}, \"desiredReplicas\": {\"N\": \"2\"}}" \
  --endpoint-url http://localhost:8000

echo "[INFO] Inserted test record for no targetAttribute scenario."

# Observe scaling for 2 minutes
for i in {1..18}; do
  kubectl get deployment scale-target --no-headers -n myscaler
  sleep 10
done

# Delete test record
aws dynamodb delete-item \
  --table-name "$TABLE_NAME" \
  --key '{"EventName": {"S": "event2"}}' \
  --endpoint-url http://localhost:8000

echo "[INFO] Deleted test record for no targetAttribute scenario."

# Delete ScaledObject
echo "[INFO] Deleting ScaledObject without targetAttribute..."
kubectl delete -f "$(dirname "$0")/resources/scaledobject-dynamodb-notargetattr.yaml"

# Final cleanup (no TRUNCATE in DynamoDB, so nothing to do)
echo "[INFO] DynamoDB e2e test completed."

# Kill port-forward after test
disown $DYNAMODB_PORT_FORWARD_PID
kill $DYNAMODB_PORT_FORWARD_PID 2>/dev/null || true
