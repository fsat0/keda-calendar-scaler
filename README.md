## What is This?

This Keda External scaler is a calendar scheduler that scales pods according to calendar events.

## How to Use

Prepare the following information as a record.

* start: The start time of the scale
* end: The end time of the scale
* desiredReplicas: Replicas to scale

## Example

### PostgreSQL

Required parameters

* type: "postgresql"
* host: host name
* port: port number
* database: database name
* user: user name
* passwordEnv: environment variable name for password
* table: table name
* startColumn: column name of the start time
* endColumn: column name of the end time
* desiredReplicasColumn: column name of the desired replicas
* timezone: timezone name (e.g., "Asia/Tokyo")
* scaleToZeroOnNoEvents: (Optional) Controls whether to scale to zero when no events are found. Set to "false" to always keep minimum replicas (default: "true")

```yaml
triggers:
- type: external
  metadata:
    scalerAddress: calendar-scaler.myscaler.svc.cluster.local:6000
    type: postgresql
    host: <host>
    port: <port>
    database: <database>
    user: <user>
    passwordFromEnv: <password>
    table: <table>
    timezone: <timezone>
    startColumn: <start_column>
    endColumn: <end_column>
    desiredReplicasColumn: <desired_replicas_column>
    scaleToZeroOnNoEvents: "false"  # Optional (default: true): set to "false" to prevent scaling to zero
```

### DynamoDB

* type: `dynamodb`
* region: AWS Region
* table: Table name of dynamodb.
* startAttribute: Field name of the start time.
* endAttribute: Field name of the end time.
* desiredReplicasAttribute: Field name of desired replicas.
* timezone: Timezone(ex. Asia/Tokyo)
* scaleToZeroOnNoEvents: (Optional) Controls whether to scale to zero when no events are found. Set to "false" to always keep minimum replicas (default: "true")

```yaml
triggers:
- type: external
  metadata:
    scalerAddress: calendar-scaler.myscaler.svc.cluster.local:6000
    type: dynamodb
    region: <region>
    table: <table>
    timezone: <timezone>
    startAttribute: <start_attribute>
    endAttribute: <end_attribute>
    desiredReplicasAttribute: <desired_replicas_attribute>
    scaleToZeroOnNoEvents: "false"  # Optional (default: true): set to "false" to prevent scaling to zero
```

> Note: startAttribute, endAttribute cannot be set to the reserved keyword of DynamoDB such as `start`, `end`, etc.
