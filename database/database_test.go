package database

import (
	pb "calendar-scaler/externalscaler"
	"testing"
)

func TestNewDatabaseUnsupportedType(t *testing.T) {
	_, err := NewDatabase("unknown", &pb.ScaledObjectRef{})
	if err == nil {
		t.Error("Expected error for unsupported database type")
	}
}
