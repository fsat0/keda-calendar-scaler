package database

import (
	pb "calendar-scaler/externalscaler"
	"fmt"
	"time"
)

// Event構造体
type Event struct {
	StartTime       time.Time
	EndTime         time.Time
	DesiredReplicas int
}

// Databaseインターフェース
type Database interface {
	GetEvents() ([]Event, error)
	Close() error
}

func NewDatabase(dbType string, metadata *pb.ScaledObjectRef) (Database, error) {
	switch dbType {
	case "postgresql":
		metadata, err := NewPostgreSQLMetadata(metadata)
		if err != nil {
			return nil, err
		}
		return NewPostgresDB(metadata)
	case "dynamodb":
		metadata, err := NewDynamoDBMetadata(metadata)
		if err != nil {
			return nil, err
		}
		return NewDynamoDB(metadata)
	default:
		return nil, fmt.Errorf("unsupported database type: %s", dbType)
	}
}
