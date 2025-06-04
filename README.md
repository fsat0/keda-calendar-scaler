# KEDA Calendar External Scaler

KEDA external scaler for scaling Kubernetes workloads based on calendar events stored in PostgreSQL or DynamoDB.

## Trigger Specification

This scaler allows you to scale your workloads according to calendar-based schedules defined in your database. It supports both PostgreSQL and DynamoDB as event sources.

---

## Example

### PostgreSQL

#### PostgreSQL Parameters

| Parameter                | Description                                                                                 | Required | Example                |
|--------------------------|---------------------------------------------------------------------------------------------|----------|------------------------|
| `type`                   | Database type. Must be `postgresql`                                                         | Yes      | `postgresql`           |
| `scalerAddress`          | Address of the external scaler service                                                      | Yes      | `calendar-scaler.myscaler.svc.cluster.local:6000` |
| `host`                   | PostgreSQL host                                                                            | Yes      | `postgres`             |
| `port`                   | PostgreSQL port                                                                            | Yes      | `5432`                 |
| `database`               | PostgreSQL database name                                                                    | Yes      | `calendar`             |
| `user`                   | PostgreSQL user                                                                             | Yes      | `postgres`             |
| `passwordEnv`            | Name of the environment variable for PostgreSQL password                                    | Yes      | `POSTGRES_PASSWORD`    |
| `table`                  | Table name                                                                                  | Yes      | `calendar_events`      |
| `startColumn`            | Column name of the start time                                                               | Yes      | `startEvent`           |
| `endColumn`              | Column name of the end time                                                                 | Yes      | `endEvent`             |
| `desiredReplicasColumn`  | Column name of the desired replicas                                                         | Yes      | `desiredReplicas`      |
| `timezone`               | Timezone name (e.g., `Asia/Tokyo`)                                                         | Yes      | `Asia/Tokyo`           |
| `scaleToZeroOnNoEvents`  | (Optional) Controls whether to scale to zero when no events are found. Set to `false` to always keep minimum replicas (default: `true`) | No | `false` |

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

#### DynamoDB Parameters

| Parameter                   | Description                                                                                 | Required | Example                |
|-----------------------------|---------------------------------------------------------------------------------------------|----------|------------------------|
| `type`                      | Database type. Must be `dynamodb`                                                           | Yes      | `dynamodb`             |
| `scalerAddress`             | Address of the external scaler service                                                      | Yes      | `calendar-scaler.myscaler.svc.cluster.local:6000` |
| `region`                    | AWS Region                                                                                  | Yes      | `ap-northeast-1`       |
| `table`                     | Table name of DynamoDB                                                                     | Yes      | `calendar_events`      |
| `startAttribute`            | Field name of the start time                                                                | Yes      | `startEvent`           |
| `endAttribute`              | Field name of the end time                                                                  | Yes      | `endEvent`             |
| `desiredReplicasAttribute`  | Field name of desired replicas                                                              | Yes      | `desiredReplicas`      |
| `timezone`                  | Timezone (e.g., `Asia/Tokyo`)                                                               | Yes      | `Asia/Tokyo`           |
| `scaleToZeroOnNoEvents`     | (Optional) Controls whether to scale to zero when no events are found. Set to `false` to always keep minimum replicas (default: `true`) | No | `false` |

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

---

## Authentication Parameters

- **PostgreSQL:** Use the `passwordEnv` parameter to specify the environment variable containing the database password.
- **DynamoDB:** Use AWS credentials via environment variables (`AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`).

## Usage

1. Deploy the external scaler and your database (PostgreSQL or DynamoDB).
2. Configure your KEDA `ScaledObject` to use the external scaler trigger with the appropriate metadata.
3. Ensure your event table/attributes are populated with calendar events.
