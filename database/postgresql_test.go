package database

import (
	pb "calendar-scaler/externalscaler"
	"os"
	"testing"
)

func TestNewPostgreSQLMetadata_RequiredFields(t *testing.T) {
	scaledObject := &pb.ScaledObjectRef{
		ScalerMetadata: map[string]string{
			"host":                  "localhost",
			"port":                  "5432",
			"username":              "user",
			"passwordEnv":           "PGPASSWORD",
			"database":              "testdb",
			"table":                 "events",
			"timezone":              "Asia/Tokyo",
			"desiredReplicasColumn": "desired_replicas",
			"startColumn":           "start_time",
			"endColumn":             "end_time",
		},
	}
	os.Setenv("PGPASSWORD", "secret")
	t.Cleanup(func() {
		os.Unsetenv("PGPASSWORD")
	})
	meta, err := NewPostgreSQLMetadata(scaledObject)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if meta.Password != "secret" {
		t.Errorf("expected password to be 'secret', got '%s'", meta.Password)
	}
}

func TestPostgreSQLMetadata_ValidateAndSetDefaults(t *testing.T) {
	meta := &PostgreSQLMetadata{
		Host:                  "",
		Port:                  "",
		User:                  "user",
		Password:              "pass",
		Database:              "db",
		Table:                 "table",
		TimeZone:              "Asia/Tokyo",
		DesiredReplicasColumn: "desired",
		StartTimeColumn:       "start",
		EndTimeColumn:         "end",
	}
	err := meta.ValidateAndSetDefaults(meta)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if meta.Host != "localhost" {
		t.Errorf("expected default host to be 'localhost', got '%s'", meta.Host)
	}
	if meta.Port != "5432" {
		t.Errorf("expected default port to be '5432', got '%s'", meta.Port)
	}
}

func TestPostgresDB_GetConnectionString(t *testing.T) {
	meta := &PostgreSQLMetadata{
		Host: "h", Port: "p", User: "u", Password: "pw", Database: "d",
	}
	connStr := meta.GetConnectionString()
	if connStr == "" {
		t.Error("connection string should not be empty")
	}
}
