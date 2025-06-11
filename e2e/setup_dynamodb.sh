#!/bin/bash
set -e

# Port-forward DynamoDB Local (if running in Kubernetes)
# This assumes a service named dynamodb in the myscaler namespace (see deploy-dynamodb.yaml)
export AWS_PAGER=""
kubectl port-forward svc/dynamodb-local 8000:8000 -n myscaler &
DYNAMODB_PORT_FORWARD_PID=$!
sleep 2

TABLE_NAME="calendar_events"

# Create DynamoDB table if not exists
aws dynamodb create-table \
  --table-name "$TABLE_NAME" \
  --attribute-definitions AttributeName=EventName,AttributeType=S \
  --key-schema AttributeName=EventName,KeyType=HASH \
  --billing-mode PAY_PER_REQUEST \
  --endpoint-url http://localhost:8000 || true

# Truncate table (delete all items)
ITEMS=$(aws dynamodb scan --table-name "$TABLE_NAME" --attributes-to-get EventName --query 'Items[*].EventName.S' --output text --endpoint-url http://localhost:8000)
if [ -n "$ITEMS" ]; then
  echo "$ITEMS" | while read -r EVENTNAME; do
    aws dynamodb delete-item --table-name "$TABLE_NAME" --key "{\"EventName\": {\"S\": \"$EVENTNAME\"}}" --endpoint-url http://localhost:8000 || true
  done
fi

# Optionally, insert initial test data here if needed for e2e

echo "[INFO] DynamoDB table initialized."

# Kill port-forward after setup
disown $DYNAMODB_PORT_FORWARD_PID
kill $DYNAMODB_PORT_FORWARD_PID 2>/dev/null || true
