package database

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	_ "github.com/lib/pq"

	"errors"
	"os"
	"reflect"

	pb "calendar-scaler/externalscaler"
)

type PostgreSQLMetadata struct {
	Host     string `validate:"optional" default:"localhost"`
	Port     string `validate:"optional" default:"5432"`
	User     string `validate:"required"`
	Password string `validate:"required"`
	Database string `validate:"required"`
	Table    string `validate:"required"`
	TimeZone string `validate:"required"`

	DesiredReplicasColumn string `validate:"required"`
	StartTimeColumn       string `validate:"required"`
	EndTimeColumn         string `validate:"required"`
	TargetColumn          string `validate:"optional"`

	Namespace    string `validate:"optional"`
	ScaledObject string `validate:"optional"`
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
		TargetColumn:          scaledObject.GetScalerMetadata()["targetColumn"],
		Namespace:             scaledObject.GetNamespace(),
		ScaledObject:          scaledObject.GetName(),
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
	if db.Meta.TargetColumn == "" {
		return db.getEventsNoTargetColumn()
	} else {
		return db.getEventsWithTargetColumn()
	}
}

func (db *PostgresDB) getEventsNoTargetColumn() ([]Event, error) {
	location, err := time.LoadLocation(db.Meta.TimeZone)
	if err != nil {
		fmt.Printf("[PostgreSQL Error] failed to load timezone '%s': %v\n", db.Meta.TimeZone, err)
		return nil, err
	}
	now := time.Now().In(location)
	query := fmt.Sprintf(
		"SELECT %s, %s, %s FROM %s WHERE %s <= $1 AND $1 <= %s",
		db.Meta.StartTimeColumn, db.Meta.EndTimeColumn, db.Meta.DesiredReplicasColumn,
		db.Meta.Table,
		db.Meta.StartTimeColumn, db.Meta.EndTimeColumn,
	)
	args := []interface{}{now}
	rows, err := db.Conn.Query(query, args...)
	if err != nil {
		fmt.Printf("[PostgreSQL Error] failed to execute query: %v\n", err)
		return nil, err
	}
	defer rows.Close()
	var events []Event
	for rows.Next() {
		var start time.Time
		var end time.Time
		var desiredReplicas int
		if err := rows.Scan(&start, &end, &desiredReplicas); err != nil {
			return nil, err
		}
		events = append(events, Event{
			StartTime:       start,
			EndTime:         end,
			DesiredReplicas: desiredReplicas,
		})
	}
	if err := rows.Err(); err != nil {
		fmt.Printf("[PostgreSQL Error] error loading events: %v\n", err)
		return nil, err
	}
	return events, nil
}

func (db *PostgresDB) getEventsWithTargetColumn() ([]Event, error) {
	location, err := time.LoadLocation(db.Meta.TimeZone)
	if err != nil {
		fmt.Printf("[PostgreSQL Error] failed to load timezone '%s': %v\n", db.Meta.TimeZone, err)
		return nil, err
	}
	now := time.Now().In(location)
	query := fmt.Sprintf(
		"SELECT %s, %s, %s, %s FROM %s WHERE %s <= $1 AND $1 <= %s",
		db.Meta.StartTimeColumn, db.Meta.EndTimeColumn, db.Meta.DesiredReplicasColumn, db.Meta.TargetColumn,
		db.Meta.Table,
		db.Meta.StartTimeColumn, db.Meta.EndTimeColumn,
	)
	args := []interface{}{now}
	rows, err := db.Conn.Query(query, args...)
	if err != nil {
		fmt.Printf("[PostgreSQL Error] failed to execute query: %v\n", err)
		return nil, err
	}
	defer rows.Close()
	var events []Event
	targetKey := db.Meta.Namespace + "/" + db.Meta.ScaledObject
	for rows.Next() {
		var start time.Time
		var end time.Time
		var desiredReplicas int
		var targets string
		if err := rows.Scan(&start, &end, &desiredReplicas, &targets); err != nil {
			fmt.Printf("[PostgreSQL Error] failed to scan row: %v\n", err)
			return nil, err
		}
		found := false
		for _, t := range strings.Split(targets, ",") {
			if strings.TrimSpace(t) == targetKey {
				found = true
				break
			}
		}
		if !found {
			continue
		}
		events = append(events, Event{
			StartTime:       start,
			EndTime:         end,
			DesiredReplicas: desiredReplicas,
		})
	}
	if err := rows.Err(); err != nil {
		fmt.Printf("[PostgreSQL Error] error loading events: %v\n", err)
		return nil, err
	}
	return events, nil
}

func (db *PostgresDB) Close() error {
	return db.Conn.Close()
}
