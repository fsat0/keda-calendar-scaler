// このファイルはPostgreSQLからイベントデータを取得し、
// それを元にスケーリングの判断を行う外部スケーラーの実装です。

package main

import (
	pb "calendar-scaler/externalscaler"
	"context"
	"fmt"
	"log"
	"net"
	"os"

	"time"

	"database/sql"

	_ "github.com/lib/pq"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"errors"
	"reflect"
)

type ExternalScaler struct {
	pb.UnimplementedExternalScalerServer
}

type Event struct {
	StartTime       time.Time
	EndTime         time.Time
	DesiredReplicas int
}

type PostgreSQLMetadata struct {
	Host     string `validate:"required" default:"localhost"`
	Port     string `validate:"required" default:"5432"`
	User     string `validate:"required"`
	Password string `validate:"required"`
	Database string `validate:"required"`
	Table    string `validate:"required"`
	TimeZone string `validate:"required"`

	DesiredReplicasColumn string `validate:"required"`
	StartTimeColumn       string `validate:"required"`
	EndTimeColumn         string `validate:"required"`
}

func NewPostgreSQLMetadata(scaledObject *pb.ScaledObjectRef) (*PostgreSQLMetadata, error) {
	scalerMetadata := &PostgreSQLMetadata{
		Host:                  scaledObject.GetScalerMetadata()["host"],
		Port:                  scaledObject.GetScalerMetadata()["port"],
		User:                  scaledObject.GetScalerMetadata()["username"],
		Password:              os.Getenv(scaledObject.GetScalerMetadata()["passwordEnv"]),
		Database:              scaledObject.GetScalerMetadata()["database"],
		Table:                 scaledObject.GetScalerMetadata()["table"],
		TimeZone:              scaledObject.GetScalerMetadata()["timezone"],
		DesiredReplicasColumn: scaledObject.GetScalerMetadata()["desiredReplicasColumn"],
		StartTimeColumn:       scaledObject.GetScalerMetadata()["startColumn"],
		EndTimeColumn:         scaledObject.GetScalerMetadata()["endColumn"],
	}
	if err := scalerMetadata.ValidateAndSetDefaults(scalerMetadata); err != nil {
		return nil, err
	}
	return scalerMetadata, nil
}

func (*PostgreSQLMetadata) ValidateAndSetDefaults(metadata interface{}) error {
	v := reflect.ValueOf(metadata).Elem()
	t := v.Type()

	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		tag := t.Field(i).Tag

		if tag.Get("validate") == "required" && field.String() == "" {
			return errors.New(t.Field(i).Name + " is required")
		}

		if field.String() == "" {
			defaultValue := tag.Get("default")
			if defaultValue != "" {
				field.SetString(defaultValue)
			}
		}
	}

	return nil
}

func (m *PostgreSQLMetadata) GetConnectionString() string {
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		m.Host,
		m.Port,
		m.User,
		m.Password,
		m.Database,
	)
}

type PostgresDB struct {
	Conn *sql.DB
	Meta *PostgreSQLMetadata
}

func NewPostgresDB(metadata *PostgreSQLMetadata) (*PostgresDB, error) {
	connStr := metadata.GetConnectionString()
	conn, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, err
	}
	if err := conn.Ping(); err != nil {
		return nil, err
	}
	return &PostgresDB{Conn: conn, Meta: metadata}, nil
}

func (db *PostgresDB) GetEvents() ([]Event, error) {
	location, err := time.LoadLocation(db.Meta.TimeZone)
	if err != nil {
		return nil, err
	}
	now := time.Now().In(location)

	query := fmt.Sprintf(
		"SELECT %s, %s, %s FROM %s WHERE %s <= $1 AND $1 <= %s",
		db.Meta.StartTimeColumn, db.Meta.EndTimeColumn, db.Meta.DesiredReplicasColumn,
		db.Meta.Table,
		db.Meta.StartTimeColumn, db.Meta.EndTimeColumn,
	)
	rows1, err1 := db.Conn.Query("SELECT NOW()")
	if err1 != nil {
		return nil, err1
	}
	defer rows1.Close()

	rows, err := db.Conn.Query(query, now)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var events []Event
	for rows.Next() {
		var event Event
		if err := rows.Scan(&event.StartTime, &event.EndTime, &event.DesiredReplicas); err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return events, nil
}

func (e *ExternalScaler) IsActive(ctx context.Context, scaledObject *pb.ScaledObjectRef) (*pb.IsActiveResponse, error) {
	metadata, err := NewPostgreSQLMetadata(scaledObject)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	db, err := NewPostgresDB(metadata)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	defer db.Conn.Close()

	events, err := db.GetEvents()
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &pb.IsActiveResponse{
		Result: events != nil,
	}, nil
}

func (e *ExternalScaler) GetMetricSpec(context.Context, *pb.ScaledObjectRef) (*pb.GetMetricSpecResponse, error) {
	return &pb.GetMetricSpecResponse{
		MetricSpecs: []*pb.MetricSpec{{
			MetricName: "eventTerm",
			TargetSize: 1,
		}},
	}, nil
}

func (e *ExternalScaler) GetMetrics(_ context.Context, metricRequest *pb.GetMetricsRequest) (*pb.GetMetricsResponse, error) {
	metadata, err := NewPostgreSQLMetadata(metricRequest.ScaledObjectRef)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	db, err := NewPostgresDB(metadata)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	defer db.Conn.Close()

	events, err := db.GetEvents()
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	maxDesiredReplicas := 0
	for _, event := range events {
		if event.DesiredReplicas > maxDesiredReplicas {
			maxDesiredReplicas = event.DesiredReplicas
		}
	}

	return &pb.GetMetricsResponse{
		MetricValues: []*pb.MetricValue{{
			MetricName:  "eventTerm",
			MetricValue: int64(maxDesiredReplicas),
		}},
	}, nil
}

func (e *ExternalScaler) StreamIsActive(scaledObject *pb.ScaledObjectRef, epsServer pb.ExternalScaler_StreamIsActiveServer) error {
	return status.Error(codes.Internal, "The external-push is not implemented.")
}

func main() {
	grpcServer := grpc.NewServer()
	lis, _ := net.Listen("tcp", ":6000")
	pb.RegisterExternalScalerServer(grpcServer, &ExternalScaler{})

	fmt.Println("listenting on :6000")
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatal(err)
	}
}
